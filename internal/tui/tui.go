// Package tui is Harold's terminal UI: a triage session in a Bubble Tea
// program, drawn as the card and the ledger (docs/design.md, The screen).
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/DArtagan/harold/internal/triage"
)

// Loader fetches the session to triage. It runs as a command, off the UI
// loop, so the screen draws while Todoist answers.
type Loader func(ctx context.Context) (triage.Session, error)

// Model is the whole of the UI's state.
type Model struct {
	load    Loader
	state   State
	session triage.Session
	err     error
	width   int
	height  int
}

// State is which of its phases the UI is in.
type State int

const (
	Loading State = iota
	Ready
	Failed
)

type loadedMsg struct{ session triage.Session }

type failedMsg struct{ err error }

// New returns the UI for the session load fetches.
func New(load Loader) Model {
	return Model{load: load}
}

// Session is the session as it stands, and false if it never loaded.
func (m Model) Session() (triage.Session, bool) {
	return m.session, m.state == Ready
}

func (m Model) Init() tea.Cmd {
	load := m.load
	return func() tea.Msg {
		session, err := load(context.Background())
		if err != nil {
			return failedMsg{err}
		}
		return loadedMsg{session}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case loadedMsg:
		m.state, m.session = Ready, msg.session
	case failedMsg:
		m.state, m.err = Failed, msg.err
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if m.state == Ready {
				m.session = m.session.Next()
			}
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	view := tea.NewView(Render(m.width, m.height, m.state, m.session, m.err))
	view.AltScreen = true
	view.WindowTitle = "Harold"
	return view
}
