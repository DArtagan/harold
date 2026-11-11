package todoist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// replay serves testdata/<name>.json for each request, picking the page by
// its cursor, and records every request's method.
type replay struct {
	t       *testing.T
	mu      sync.Mutex
	methods []string
	queries []string
}

func (r *replay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.methods = append(r.methods, req.Method)
	r.queries = append(r.queries, req.URL.RawQuery)
	r.mu.Unlock()

	name := strings.Trim(req.URL.Path, "/")
	if cursor := req.URL.Query().Get("cursor"); cursor != "" {
		name += "-" + cursor
	}
	body, err := os.ReadFile(filepath.Join("testdata", strings.ReplaceAll(name, "/", "-")+".json"))
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func serve(t *testing.T) (*Client, *replay) {
	r := &replay{t: t}
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	return NewAt(server.Client(), server.URL), r
}

func TestTasksByFilterFollowsTheCursor(t *testing.T) {
	client, r := serve(t)
	tasks, err := client.TasksByFilter(context.Background(), "#Work & today")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, task := range tasks {
		titles = append(titles, task.Content)
	}
	if got := strings.Join(titles, ", "); got != "Water the ferns, Return the library books, Book the car in" {
		t.Errorf("tasks %s", got)
	}
	if tasks[0].Due == nil || tasks[0].Due.String != "today" {
		t.Errorf("due %+v", tasks[0].Due)
	}
	if !strings.Contains(r.queries[0], "query=%23Work+%26+today") || !strings.Contains(r.queries[0], "limit=200") {
		t.Errorf("first request %q", r.queries[0])
	}
}

func TestProjectsAndLabels(t *testing.T) {
	client, _ := serve(t)
	projects, err := client.Projects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 || !projects[0].InboxProject || projects[1].Name != "Home Repairs" {
		t.Errorf("projects %+v", projects)
	}
	labels, err := client.Labels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 1 || labels[0].Name != "waiting" {
		t.Errorf("labels %+v", labels)
	}
}

func TestAnErrorCarriesTodoistsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"Invalid filter"}`, http.StatusBadRequest)
	}))
	defer server.Close()
	_, err := NewAt(server.Client(), server.URL).TasksByFilter(context.Background(), "((")
	if err == nil || !strings.Contains(err.Error(), "Invalid filter") {
		t.Fatalf("error %v", err)
	}
}

// Harold writes nothing to Todoist (AGENTS.md): every request is a GET.
func TestTheClientOnlyReads(t *testing.T) {
	client, r := serve(t)
	ctx := context.Background()
	_, _ = client.TasksByFilter(ctx, "today")
	_, _ = client.Projects(ctx)
	_, _ = client.Labels(ctx)
	for _, method := range r.methods {
		if method != http.MethodGet {
			t.Errorf("a %s request", method)
		}
	}
	if len(r.methods) == 0 {
		t.Fatal("no requests recorded")
	}
}
