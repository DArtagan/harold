package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/DArtagan/harold/internal/demo"
	"github.com/DArtagan/harold/internal/triage"
)

// screen drives the model through msgs, as the program would, and returns
// the screen as text: what a person would read, without the styling.
func screen(t *testing.T, width, height int, load Loader, msgs ...tea.Msg) string {
	t.Helper()
	var m tea.Model = New(load)
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	if load != nil {
		m, _ = m.Update(m.Init()())
	}
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	content := m.View().Content
	if lines := strings.Count(content, "\n") + 1; content != "" && lines != height && width >= minWidth && height >= minHeight {
		t.Errorf("%d lines on a %d-line screen", lines, height)
	}
	for i, line := range strings.Split(ansi.Strip(content), "\n") {
		if w := ansi.StringWidth(line); w > width {
			t.Errorf("line %d is %d wide on a %d-wide screen: %q", i, w, width, line)
		}
	}
	return ansi.Strip(content)
}

func demoSession(context.Context) (triage.Session, error) {
	return triage.New(demo.Query, demo.Tasks()), nil
}

var enter = tea.KeyPressMsg{Code: tea.KeyEnter}

func presses(n int) []tea.Msg {
	msgs := make([]tea.Msg, n)
	for i := range msgs {
		msgs[i] = enter
	}
	return msgs
}

// The screen, as golden files under testdata/. `go test ./... -update`
// rewrites them; read the diff before committing it.
func TestScreen(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		load          Loader
		msgs          []tea.Msg
	}{
		{"loading", 60, 20, nil, nil},
		{"first_card", 60, 20, demoSession, nil},
		{"after_two", 60, 20, demoSession, presses(2)},
		{"long_description", 60, 20, demoSession, presses(4)},
		{"everything", 60, 20, demoSession, presses(6)},
		{"tall", 72, 40, demoSession, presses(3)},
		{"narrow", 30, 14, demoSession, presses(4)},
		{"too_small", 20, 6, demoSession, nil},
		{"nothing_matches", 60, 16, func(context.Context) (triage.Session, error) {
			return triage.New("#Garden & overdue", nil), nil
		}, nil},
		{"failed", 60, 16, func(context.Context) (triage.Session, error) {
			return triage.Session{}, errors.New("todoist: /tasks/filter: 400 Bad Request: Invalid filter")
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			golden.RequireEqual(t, screen(t, tt.width, tt.height, tt.load, tt.msgs...))
		})
	}
}

func TestEnterDoesNothingBeforeTheTasksLoad(t *testing.T) {
	var m tea.Model = New(demoSession)
	m, _ = m.Update(enter)
	if _, ok := m.(Model).Session(); ok {
		t.Error("loaded without a load")
	}
}

func TestQuitKeys(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: 'q', Text: "q"}, {Code: tea.KeyEscape}, {Code: 'c', Mod: tea.ModCtrl}} {
		_, cmd := New(demoSession).Update(key)
		if cmd == nil {
			t.Errorf("%s does not quit", key)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s does not quit", key)
		}
	}
}
