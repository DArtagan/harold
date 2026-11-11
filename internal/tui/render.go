package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/DArtagan/harold/internal/triage"
)

// Styles. Plain terminal attributes, no brand colours (AGENTS.md).
var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	faintStyle  = lipgloss.NewStyle().Faint(true)
	borderStyle = lipgloss.NewStyle().Faint(true)
)

// The smallest screen Render lays out; below it, it says so.
const (
	minWidth  = 24
	minHeight = 8
)

// Render draws the screen: the ledger above, filling from the bottom, and the
// card across the foot, its title above its description. It is a function of
// its arguments alone, which is what the golden files test.
func Render(width, height int, state State, session triage.Session, today time.Time, err error) string {
	if width == 0 || height == 0 {
		return ""
	}
	if width < minWidth || height < minHeight {
		return ansi.Truncate("Harold needs a larger window.", width, "…")
	}

	// The card is as tall as its text, up to two-thirds of the screen, so
	// the ledger keeps a third. Its three borders come out of that share.
	room := height*2/3 - 3
	card := renderCard(width, room, state, session, today, err)
	// A blank line between the ledger and the card (Will, 2026-11-11: "a
	// bit more whitespace between the box and the first entry").
	ledger := renderLedger(width, height-len(card)-1, session.Ledger())
	ledger = append(ledger, "")
	return strings.Join(append(ledger, card...), "\n")
}

func renderCard(width, room int, state State, session triage.Session, today time.Time, err error) []string {
	inner := width - 4 // the border and a space either side

	var title, description, due string
	switch state {
	case Loading:
		title = "Loading…"
	case Failed:
		title, description = "Harold could not load the tasks.", err.Error()
	case Ready:
		if task, ok := session.Card(); ok {
			title, description = task.Title, task.Description
			due = task.Due.Label(today)
		} else if session.Total() == 0 {
			title = "Nothing matches " + session.Query() + "."
		} else {
			title = "That's everything."
			description = fmt.Sprintf("%d considered.", session.Total())
		}
	}

	query, count := session.Query(), ""
	if state == Ready && session.Total() > 0 {
		shown := min(session.Considered()+1, session.Total())
		count = fmt.Sprintf("%d of %d", shown, session.Total())
	}

	titleLines, descriptionLines := wrap(title, inner), wrap(description, inner)
	titleRows, descriptionRows := share(len(titleLines), len(descriptionLines), room)

	// The top edge carries the date and the count, the bottom edge the keys
	// and the query (Will, 2026-11-11). The date and the query are what a
	// short edge keeps.
	lines := []string{border("╭", due, count, "╮", width, keepLeft)}
	for _, line := range fit(titleLines, inner, titleRows) {
		lines = append(lines, row(titleStyle.Render(line), inner))
	}
	lines = append(lines, border("├", "", "", "┤", width, keepLeft))
	for _, line := range fit(descriptionLines, inner, descriptionRows) {
		lines = append(lines, row(line, inner))
	}
	return append(lines, border("╰", "enter next · q quit", query, "╯", width, keepRight))
}

// renderLedger fills rows lines with the considered tasks, the newest at the
// bottom, like a teletype's paper. Older tasks scroll off the top.
func renderLedger(width, rows int, ledger []triage.Task) []string {
	lines := make([]string, rows)
	start := max(0, len(ledger)-rows)
	shown := ledger[start:]
	offset := rows - len(shown)
	for i, task := range shown {
		lines[offset+i] = ledgerLine(task, width)
	}
	return lines
}

func ledgerLine(task triage.Task, width int) string {
	inner := width - 4
	project := task.Project
	if ansi.StringWidth(project) > inner/3 {
		project = ansi.Truncate(project, inner/3, "…")
	}
	room := inner - ansi.StringWidth(project) - 2
	title := ansi.Truncate(task.Title, room, "…")
	gap := inner - ansi.StringWidth(title) - ansi.StringWidth(project)
	return "  " + faintStyle.Render(title+strings.Repeat(" ", gap)+project)
}

// side is which of an edge's two labels it keeps when it is short.
type side int

const (
	keepLeft side = iota
	keepRight
)

// border draws an edge of the card with labels set into it, one at each end.
// The kept label gets all the room the other leaves; when it would still be
// cut, the other is dropped to make room for it, and only then is it cut.
func border(leftCorner, left, right, rightCorner string, width int, keep side) string {
	room := width - 2
	kept, other := left, right
	if keep == keepRight {
		kept, other = right, left
	}
	// A label is set in with a dash and a space outside it and a space
	// inside: "─ label " at the left, " label ─" at the right. At least one
	// dash separates the two.
	framed := func(label string) int {
		if label == "" {
			return 0
		}
		return ansi.StringWidth(label) + 3
	}
	keptRoom := room - framed(other) - 1 - 3
	if other != "" && ansi.StringWidth(kept) > keptRoom {
		other, keptRoom = "", room-1-3
	}
	if kept != "" {
		kept = ansi.Truncate(kept, max(1, keptRoom), "…")
	}
	if keep == keepRight {
		left, right = other, kept
	} else {
		left, right = kept, other
	}

	if left != "" {
		left = "─ " + left + " "
	}
	if right != "" {
		right = " " + right + " ─"
	}
	fill := max(0, room-ansi.StringWidth(left)-ansi.StringWidth(right))
	return borderStyle.Render(leftCorner + left + strings.Repeat("─", fill) + right + rightCorner)
}

// row is one line inside the card, padded to its width.
func row(content string, inner int) string {
	pad := inner - ansi.StringWidth(content)
	return borderStyle.Render("│") + " " + content + strings.Repeat(" ", max(0, pad)) + " " + borderStyle.Render("│")
}

// wrap breaks text into lines no wider than width, keeping its own line
// breaks.
func wrap(text string, width int) []string {
	var wrapped []string
	if text != "" {
		for _, paragraph := range strings.Split(text, "\n") {
			wrapped = append(wrapped, strings.Split(ansi.Wrap(paragraph, width, ""), "\n")...)
		}
	}
	return wrapped
}

// share divides room lines between the title and the description by what
// each needs (Will, 2026-11-11: "dynamically size the two boxes to the text
// contents of the task"). Each gets at least one line, so the card is always
// divided in two. When both need more than there is, the title gets at least
// half, and whichever needs less gives the other its spare.
func share(titleNeeds, descriptionNeeds, room int) (int, int) {
	titleNeeds, descriptionNeeds = max(1, titleNeeds), max(1, descriptionNeeds)
	room = max(2, room)
	if titleNeeds+descriptionNeeds <= room {
		return titleNeeds, descriptionNeeds
	}
	title := min(titleNeeds, max((room+1)/2, room-descriptionNeeds))
	return title, room - title
}

// fit makes wrapped exactly lines long, padding with blank lines, or cutting
// it and ending the last line kept with an ellipsis.
func fit(wrapped []string, width, lines int) []string {
	wrapped = append([]string(nil), wrapped...)
	if len(wrapped) > lines {
		last := strings.TrimRight(wrapped[lines-1], " ")
		if ansi.StringWidth(last) >= width {
			last = ansi.Truncate(last, width-1, "")
		}
		wrapped = append(wrapped[:lines-1], last+"…")
	}
	for len(wrapped) < lines {
		wrapped = append(wrapped, "")
	}
	return wrapped
}
