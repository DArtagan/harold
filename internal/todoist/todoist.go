// Package todoist is Harold's client for Todoist's API v1: the one module
// that calls it.
//
// It only reads. Harold writes nothing to Todoist until a spec says it may
// (AGENTS.md), and a test fails if this package sends anything but a GET.
package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// BaseURL is Todoist's API v1.
const BaseURL = "https://api.todoist.com/api/v1"

// pageSize is the largest page Todoist serves.
const pageSize = 200

// Task is a task as the API returns it, with the fields Harold reads.
type Task struct {
	ID          string   `json:"id"`
	Content     string   `json:"content"`
	Description string   `json:"description"`
	ProjectID   string   `json:"project_id"`
	Labels      []string `json:"labels"`
	Priority    int      `json:"priority"`
	Due         *Due     `json:"due"`
}

// Due is a task's due date. String is as the user wrote it ("every monday").
type Due struct {
	Date        string `json:"date"`
	String      string `json:"string"`
	IsRecurring bool   `json:"is_recurring"`
}

// Project is a project as the API returns it.
type Project struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	InboxProject bool   `json:"inbox_project"`
	// ParentID is empty for a top-level project.
	ParentID string `json:"parent_id"`
	// ChildOrder is the project's place among its siblings; OrderKey breaks
	// a tie between two that share one.
	ChildOrder int    `json:"child_order"`
	OrderKey   string `json:"order_key"`
}

// SidebarOrder ranks projects as Todoist's sidebar lists them: the Inbox
// first, then the project tree top to bottom, each project followed by its
// subprojects, siblings by ChildOrder and then OrderKey. A project whose
// parent is not among projects is ranked as a top-level one.
//
// Measured on Will's account, 2026-11-11: across 58 projects, ChildOrder and
// OrderKey put every set of siblings in the same order, and one pair shared a
// ChildOrder.
func SidebarOrder(projects []Project) map[string]int {
	byID := make(map[string]bool, len(projects))
	for _, p := range projects {
		byID[p.ID] = true
	}
	children := map[string][]Project{}
	for _, p := range projects {
		parent := p.ParentID
		if !byID[parent] {
			parent = ""
		}
		children[parent] = append(children[parent], p)
	}
	for _, siblings := range children {
		sort.SliceStable(siblings, func(i, j int) bool {
			a, b := siblings[i], siblings[j]
			if a.InboxProject != b.InboxProject {
				return a.InboxProject
			}
			if a.ChildOrder != b.ChildOrder {
				return a.ChildOrder < b.ChildOrder
			}
			return a.OrderKey < b.OrderKey
		})
	}

	rank := make(map[string]int, len(projects))
	var visit func(parent string)
	visit = func(parent string) {
		for _, p := range children[parent] {
			if _, seen := rank[p.ID]; seen {
				continue
			}
			rank[p.ID] = len(rank)
			visit(p.ID)
		}
	}
	visit("")
	return rank
}

// SortForTriage orders tasks for the card (docs/design.md, Filters):
//
//  1. by their projects' places in the sidebar; a project not among projects
//     comes last;
//  2. within a project, by priority, p1 first;
//  3. then by due date, earliest first, and no date last. On the same day a
//     task with a time comes before an all-day one, and timed tasks go by
//     time;
//  4. then as Todoist returned them.
//
// Times are compared in loc, which is where a floating time is read.
func SortForTriage(tasks []Task, projects []Project, loc *time.Location) {
	rank := SidebarOrder(projects)
	place := func(t Task) int {
		if r, ok := rank[t.ProjectID]; ok {
			return r
		}
		return len(rank)
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if pa, pb := place(a), place(b); pa != pb {
			return pa < pb
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		return a.Due.before(b.Due, loc)
	})
}

// before reports whether d is due before other. A nil or unreadable due date
// is after every readable one.
func (d *Due) before(other *Due, loc *time.Location) bool {
	dayA, timeA, okA := d.When(loc)
	dayB, timeB, okB := other.When(loc)
	switch {
	case !okA || !okB:
		return okA && !okB
	case !dayA.Equal(dayB):
		return dayA.Before(dayB)
	case timeA.IsZero() != timeB.IsZero():
		return !timeA.IsZero()
	default:
		return timeA.Before(timeB)
	}
}

// When reads a due date in loc: the day it falls on, and its time, which is
// zero for an all-day date, and false if there is no date it can read.
// Todoist writes an all-day date as YYYY-MM-DD, a floating time as
// YYYY-MM-DDTHH:MM:SS in the user's own zone, and a fixed one in UTC, ending
// in Z (API documentation, Due dates).
func (d *Due) When(loc *time.Location) (day, at time.Time, ok bool) {
	if d == nil || d.Date == "" {
		return time.Time{}, time.Time{}, false
	}
	if day, err := time.ParseInLocation(time.DateOnly, d.Date, loc); err == nil {
		return day, time.Time{}, true
	}
	var err error
	if strings.HasSuffix(d.Date, "Z") {
		at, err = time.Parse(time.RFC3339Nano, d.Date)
	} else {
		at, err = time.ParseInLocation("2006-01-02T15:04:05.999999999", d.Date, loc)
	}
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	at = at.In(loc)
	y, m, dd := at.Date()
	return time.Date(y, m, dd, 0, 0, 0, 0, loc), at, true
}

// Label is a personal label as the API returns it.
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Client calls the API through an HTTP client that adds the token, such as
// one from golang.org/x/oauth2.
type Client struct {
	http    *http.Client
	baseURL string
}

// New returns a client for Todoist's API.
func New(httpClient *http.Client) *Client {
	return NewAt(httpClient, BaseURL)
}

// NewAt returns a client for the API at baseURL, for tests.
func NewAt(httpClient *http.Client, baseURL string) *Client {
	return &Client{http: httpClient, baseURL: strings.TrimSuffix(baseURL, "/")}
}

// TasksByFilter returns every task matching a filter query, in Todoist's
// order, following the cursor to the last page.
func (c *Client) TasksByFilter(ctx context.Context, query string) ([]Task, error) {
	return list[Task](ctx, c, "/tasks/filter", url.Values{"query": {query}})
}

// Projects returns every project.
func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	return list[Project](ctx, c, "/projects", nil)
}

// Labels returns every personal label.
func (c *Client) Labels(ctx context.Context) ([]Label, error) {
	return list[Label](ctx, c, "/labels", nil)
}

type page[T any] struct {
	Results    []T     `json:"results"`
	NextCursor *string `json:"next_cursor"`
}

func list[T any](ctx context.Context, c *Client, path string, params url.Values) ([]T, error) {
	var all []T
	cursor := ""
	for {
		query := url.Values{}
		for key, values := range params {
			query[key] = values
		}
		query.Set("limit", fmt.Sprint(pageSize))
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		var p page[T]
		if err := c.get(ctx, path, query, &p); err != nil {
			return nil, err
		}
		all = append(all, p.Results...)
		if p.NextCursor == nil || *p.NextCursor == "" {
			return all, nil
		}
		cursor = *p.NextCursor
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("todoist: %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		// The body is Todoist's error message; it never holds the token.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("todoist: %s: %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		return fmt.Errorf("todoist: %s: reading the reply: %w", path, err)
	}
	return nil
}
