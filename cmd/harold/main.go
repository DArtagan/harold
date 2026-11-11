// Command harold is Harold for Todoist: a terminal UI for triaging a Todoist
// list one task at a time.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/oauth2"

	"github.com/DArtagan/harold/internal/auth"
	"github.com/DArtagan/harold/internal/demo"
	"github.com/DArtagan/harold/internal/filter"
	"github.com/DArtagan/harold/internal/todoist"
	"github.com/DArtagan/harold/internal/triage"
	"github.com/DArtagan/harold/internal/tui"
)

// version is set at build time from the release tag; see docs/releasing.md.
var version = "dev"

// tokenEnv, when set, is the token Harold uses instead of the keyring's: for
// where there is no keyring. It is never refreshed.
const tokenEnv = "HAROLD_TODOIST_TOKEN"

const usage = `Harold for Todoist: triage a Todoist list one task at a time.
Harold is not created by, affiliated with, or supported by Todoist.

Usage:
  harold [flags] [filter…]   triage the tasks a filter matches (default: today)
  harold login               sign in to Todoist in the browser
  harold logout              forget the sign-in

A filter is Todoist's own syntax ('#Work | @waiting'), or bare words: a
project, a label, or a date (harold inbox, harold work next week).

Flags:
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("harold", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print the version")
	useDemo := flags.Bool("demo", false, "show synthetic tasks, with no account")
	devState := flags.String("dev-state", "", "keep the session's position in `file` across restarts (the dev loop)")
	flags.Usage = func() {
		_, _ = fmt.Fprint(stderr, usage)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	words := flags.Args()

	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}
	if len(words) == 1 {
		switch words[0] {
		case "login":
			return login(stdout, stderr)
		case "logout":
			return logout(stdout, stderr)
		}
	}

	var load tui.Loader
	if *useDemo {
		load = demoLoader(words)
	} else {
		client, err := apiClient()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harold:", err)
			return 1
		}
		load = todoistLoader(todoist.New(client), words)
	}

	position := readPosition(*devState, stderr)
	final, err := tea.NewProgram(tui.New(restoring(load, position))).Run()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "harold:", err)
		return 1
	}
	if session, ok := final.(tui.Model).Session(); ok && *devState != "" {
		writePosition(*devState, session.Position(), stderr)
	}
	return 0
}

// apiClient is an HTTP client carrying the token: the environment's if set,
// the keyring's otherwise.
func apiClient() (*http.Client, error) {
	ctx := context.Background()
	if token := os.Getenv(tokenEnv); token != "" {
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})), nil
	}
	source, err := auth.New(auth.Keyring{}).TokenSource(ctx)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, source), nil
}

// todoistLoader resolves the words against the account, then fetches the
// tasks the query matches.
func todoistLoader(client *todoist.Client, words []string) tui.Loader {
	return func(ctx context.Context) (triage.Session, error) {
		projects, err := client.Projects(ctx)
		if err != nil {
			return triage.Session{}, err
		}
		labels, err := client.Labels(ctx)
		if err != nil {
			return triage.Session{}, err
		}
		account := filter.Account{}
		names := map[string]string{}
		for _, project := range projects {
			names[project.ID] = project.Name
			account.Projects = append(account.Projects, project.Name)
			if project.InboxProject {
				account.Inbox = project.Name
			}
		}
		for _, label := range labels {
			account.Labels = append(account.Labels, label.Name)
		}
		query, err := filter.Resolve(words, account)
		if err != nil {
			return triage.Session{}, err
		}
		found, err := client.TasksByFilter(ctx, query)
		if err != nil {
			return triage.Session{}, err
		}
		tasks := make([]triage.Task, len(found))
		for i, task := range found {
			tasks[i] = triage.Task{
				ID:          task.ID,
				Title:       task.Content,
				Description: task.Description,
				Project:     names[task.ProjectID],
				Labels:      task.Labels,
			}
			if task.Due != nil {
				tasks[i].Due = task.Due.String
			}
		}
		return triage.New(query, tasks), nil
	}
}

// demoLoader shows the demo tasks under the query the words would make,
// resolved against the demo tasks' own projects and labels.
func demoLoader(words []string) tui.Loader {
	return func(context.Context) (triage.Session, error) {
		tasks := demo.Tasks()
		account := filter.Account{Inbox: "Inbox"}
		seen := map[string]bool{}
		for _, task := range tasks {
			if !seen["#"+task.Project] {
				seen["#"+task.Project] = true
				account.Projects = append(account.Projects, task.Project)
			}
			for _, label := range task.Labels {
				if !seen["@"+label] {
					seen["@"+label] = true
					account.Labels = append(account.Labels, label)
				}
			}
		}
		query, err := filter.Resolve(words, account)
		if err != nil {
			return triage.Session{}, err
		}
		return triage.New(query+" (demo)", tasks), nil
	}
}

// restoring wraps load so the session it returns resumes from position.
func restoring(load tui.Loader, position *triage.Position) tui.Loader {
	if position == nil {
		return load
	}
	return func(ctx context.Context) (triage.Session, error) {
		session, err := load(ctx)
		if err != nil {
			return session, err
		}
		return triage.Restore(session.Query(), allTasks(session), *position), nil
	}
}

// allTasks is a fresh session's tasks, in order.
func allTasks(s triage.Session) []triage.Task {
	var tasks []triage.Task
	for task, ok := s.Card(); ok; task, ok = s.Card() {
		tasks = append(tasks, task)
		s = s.Next()
	}
	return tasks
}

func readPosition(path string, stderr io.Writer) *triage.Position {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	var position triage.Position
	if err == nil {
		err = json.Unmarshal(data, &position)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harold: ignoring %s: %v\n", path, err)
		return nil
	}
	return &position
}

func writePosition(path string, position triage.Position, stderr io.Writer) {
	data, err := json.Marshal(position)
	if err == nil {
		err = os.WriteFile(path, data, 0o600)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harold: keeping the position: %v\n", err)
	}
}

func login(stdout, stderr io.Writer) int {
	if os.Getenv(tokenEnv) != "" {
		_, _ = fmt.Fprintf(stderr, "harold: note: %s is set, and is used instead of this sign-in while it is.\n", tokenEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := auth.New(auth.Keyring{}).Login(ctx, openBrowser, stdout)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "harold:", err)
		return 1
	}
	if result.Refreshable {
		_, _ = fmt.Fprintln(stdout, "Signed in. Harold keeps the sign-in fresh by itself.")
	} else {
		_, _ = fmt.Fprintf(stdout, "Signed in, but Todoist issued no refresh token: this sign-in lasts until %s.\n",
			result.Expiry.Local().Format(time.RFC1123))
	}
	return 0
}

func logout(stdout, stderr io.Writer) int {
	if err := auth.New(auth.Keyring{}).Logout(); err != nil {
		_, _ = fmt.Fprintln(stderr, "harold:", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "Signed out.")
	return 0
}

func openBrowser(url string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("no %s", command)
	}
	cmd := exec.Command(command, url)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
