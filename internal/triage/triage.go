// Package triage is a triage session: the queue of tasks still to consider,
// the task on the card, and the ledger of those already considered.
//
// It is pure and knows nothing of terminals or Todoist, so the terminal UI
// and the planned web app share it (docs/design.md, Architecture). A Session
// is a value: every transition returns a new one.
package triage

import (
	"fmt"
	"strings"
	"time"
)

// Task is a task as Harold shows it.
type Task struct {
	ID          string
	Title       string
	Description string
	Project     string
	Labels      []string
	// Due is nil for a task with no date.
	Due *Due
}

// Due is when a task is due, read in the local zone.
type Due struct {
	// Day is midnight on the day it is due.
	Day time.Time
	// At is its time, and zero for an all-day task.
	At time.Time
	// Rule is a recurring task's rule as Todoist words it ("every monday"),
	// and empty for a task that does not recur.
	Rule string
}

// Label is how a due date reads on the card, given the day it is: how many
// days overdue, today, tomorrow, or the date; then its time, if it has one;
// then a recurring task's rule (Will, 2026-11-11).
func (d *Due) Label(today time.Time) string {
	if d == nil {
		return ""
	}
	var label string
	switch days := dayNumber(today) - dayNumber(d.Day); {
	case days > 1:
		label = fmt.Sprintf("%d days overdue", days)
	case days == 1:
		label = "1 day overdue"
	case days == 0:
		label = "today"
	case days == -1:
		label = "tomorrow"
	case d.Day.Year() == today.Year():
		label = d.Day.Format("Mon 2 Jan")
	default:
		label = d.Day.Format("Mon 2 Jan 2006")
	}
	if !d.At.IsZero() {
		label += " " + d.At.Format("3:04pm")
	}
	if d.Rule != "" {
		label += " · " + strings.TrimSpace(d.Rule)
	}
	return label
}

// dayNumber counts days on the calendar, so the difference between two is
// the days between them whatever the clock and daylight saving say.
func dayNumber(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// Session is one pass through the tasks a query returned.
type Session struct {
	query string
	// queue holds the tasks not yet considered; its first is on the card.
	queue []Task
	// ledger holds the tasks considered, oldest first.
	ledger []Task
	total  int
}

// New starts a session over tasks, in the order Todoist returned them.
func New(query string, tasks []Task) Session {
	return Session{query: query, queue: clone(tasks), total: len(tasks)}
}

// Query is the Todoist filter query the session's tasks came from.
func (s Session) Query() string { return s.query }

// Card is the task under consideration, and false once none is left.
func (s Session) Card() (Task, bool) {
	if len(s.queue) == 0 {
		return Task{}, false
	}
	return s.queue[0], true
}

// Ledger is the tasks considered so far, oldest first.
func (s Session) Ledger() []Task { return clone(s.ledger) }

// Total is the number of tasks in the session.
func (s Session) Total() int { return s.total }

// Considered is the number of tasks moved into the ledger.
func (s Session) Considered() int { return len(s.ledger) }

// Next moves the card's task into the ledger, bringing the next to the card.
// With nothing on the card it returns the session unchanged.
func (s Session) Next() Session {
	if len(s.queue) == 0 {
		return s
	}
	next := s
	next.ledger = append(clone(s.ledger), s.queue[0])
	next.queue = clone(s.queue[1:])
	return next
}

// Position is where a session stands, by task ID, so it can be restored over
// a fresh copy of the same tasks.
type Position struct {
	Query  string   `json:"query"`
	Ledger []string `json:"ledger"`
}

// Position returns where the session stands.
func (s Session) Position() Position {
	ids := make([]string, len(s.ledger))
	for i, task := range s.ledger {
		ids[i] = task.ID
	}
	return Position{Query: s.query, Ledger: ids}
}

// Restore starts a session over tasks and replays pos onto it: each task
// pos names, in pos's order, goes to the ledger. A task pos names that is no
// longer among tasks is skipped. A position for another query is ignored.
func Restore(query string, tasks []Task, pos Position) Session {
	s := New(query, tasks)
	if pos.Query != query {
		return s
	}
	byID := make(map[string]Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	considered := make(map[string]bool, len(pos.Ledger))
	for _, id := range pos.Ledger {
		task, ok := byID[id]
		if !ok || considered[id] {
			continue
		}
		considered[id] = true
		s.ledger = append(s.ledger, task)
	}
	s.queue = nil
	for _, task := range tasks {
		if !considered[task.ID] {
			s.queue = append(s.queue, task)
		}
	}
	return s
}

func clone(tasks []Task) []Task {
	if len(tasks) == 0 {
		return nil
	}
	return append([]Task(nil), tasks...)
}
