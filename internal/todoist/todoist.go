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
	"strings"
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
