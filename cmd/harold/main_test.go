package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DArtagan/harold/internal/todoist"
	"github.com/DArtagan/harold/internal/triage"
)

// fakeAPI serves a small account: an Inbox and a Home project with a Garden
// subproject, a label, and the tasks a query matched, in an order Todoist
// might return them.
func fakeAPI(t *testing.T, queries *[]string) *todoist.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/projects":
			_, _ = io.WriteString(w, `{"results": [
				{"id": "garden", "name": "Garden", "parent_id": "home", "child_order": 1},
				{"id": "home", "name": "Home", "child_order": 1},
				{"id": "inbox", "name": "Inbox", "inbox_project": true, "child_order": 0}
			], "next_cursor": null}`)
		case "/labels":
			_, _ = io.WriteString(w, `{"results": [{"id": "l", "name": "waiting"}], "next_cursor": null}`)
		case "/tasks/filter":
			*queries = append(*queries, r.URL.Query().Get("query"))
			_, _ = io.WriteString(w, `{"results": [
				{"id": "g1", "content": "Sow the beans", "project_id": "garden", "priority": 1,
				 "due": {"date": "2026-11-10", "string": "every tuesday", "is_recurring": true}},
				{"id": "h1", "content": "Oil the hinge", "project_id": "home", "priority": 1,
				 "due": {"date": "2026-11-11T09:30:00", "string": "today at 9:30", "is_recurring": false}},
				{"id": "h2", "content": "Fix the gutter", "project_id": "home", "priority": 4, "labels": ["waiting"],
				 "due": {"date": "2026-11-12", "string": "tomorrow", "is_recurring": false}},
				{"id": "i1", "content": "Sort the post", "project_id": "inbox", "priority": 1, "due": null}
			], "next_cursor": null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return todoist.NewAt(server.Client(), server.URL)
}

func TestTodoistLoader(t *testing.T) {
	var queries []string
	session, err := todoistLoader(fakeAPI(t, &queries), []string{"home", "waiting"})(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := "#Home & @waiting"; len(queries) != 1 || queries[0] != want {
		t.Errorf("queries sent %q, want [%q]", queries, want)
	}
	if session.Query() != "#Home & @waiting" {
		t.Errorf("session query %q", session.Query())
	}

	tasks := allTasks(session)
	var order []string
	for _, task := range tasks {
		order = append(order, task.ID)
	}
	// The sidebar's order (Inbox, Home, Garden), and within Home p1 first.
	if got := strings.Join(order, " "); got != "i1 h2 h1 g1" {
		t.Errorf("order %s", got)
	}

	byID := map[string]triage.Task{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	if byID["g1"].Project != "Garden" || byID["h2"].Labels[0] != "waiting" {
		t.Errorf("project or labels: %+v %+v", byID["g1"], byID["h2"])
	}
	if byID["i1"].Due != nil {
		t.Errorf("a task with no date has %+v", byID["i1"].Due)
	}
	// Only a recurring task carries its rule; a one-off's due string is not one.
	if byID["g1"].Due.Rule != "every tuesday" || byID["h2"].Due.Rule != "" {
		t.Errorf("rules %q %q", byID["g1"].Due.Rule, byID["h2"].Due.Rule)
	}
	at := byID["h1"].Due.At
	if at.Hour() != 9 || at.Minute() != 30 || at.Location() != time.Local {
		t.Errorf("time %v", at)
	}
}

func TestRestoringResumesAPosition(t *testing.T) {
	var queries []string
	load := todoistLoader(fakeAPI(t, &queries), nil)
	first, err := load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	position := first.Next().Position()

	resumed, err := restoring(load, &position)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Considered() != 1 {
		t.Errorf("resumed with %d considered", resumed.Considered())
	}
	firstCard, _ := first.Card()
	if card, _ := resumed.Card(); card.ID == firstCard.ID {
		t.Error("resumed on the first card")
	}
}
