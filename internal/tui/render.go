package tui

import (
	"fmt"
	"strings"

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
func Render(width, height int, state State, session triage.Session, err error) string {
	if width == 0 || height == 0 {
		return ""
	}
	if width < minWidth || height < minHeight {
		return ansi.Truncate("Harold needs a larger window.", width, "…")
	}

	// The card's two halves share its height equally: a third of the
	// screen, between 2 and 6 lines each.
	half := clamp((height/3-3)/2, 2, 6)
	card := renderCard(width, half, state, session, err)
	ledger := renderLedger(width, height-len(card), session.Ledger())
	return strings.Join(append(ledger, card...), "\n")
}

func renderCard(width, half int, state State, session triage.Session, err error) []string {
	inner := width - 4 // the border and a space either side

	var title, description string
	switch state {
	case Loading:
		title = "Loading…"
	case Failed:
		title, description = "Harold could not load the tasks.", err.Error()
	case Ready:
		if task, ok := session.Card(); ok {
			title, description = task.Title, task.Description
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

	lines := []string{border("╭", query, count, "╮", width)}
	for _, line := range block(title, inner, half) {
		lines = append(lines, row(titleStyle.Render(line), inner))
	}
	lines = append(lines, borderStyle.Render("├"+strings.Repeat("─", width-2)+"┤"))
	for _, line := range block(description, inner, half) {
		lines = append(lines, row(line, inner))
	}
	return append(lines, border("╰", "enter next · q quit", "", "╯", width))
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

// border draws a top or bottom edge with left and right labels set into it.
func border(leftCorner, left, right, rightCorner string, width int) string {
	room := width - 2
	if left != "" {
		left = "─ " + ansi.Truncate(left, room/2, "…") + " "
	}
	if right != "" {
		right = " " + right + " ─"
	}
	fill := room - ansi.StringWidth(left) - ansi.StringWidth(right)
	if fill < 0 {
		right, fill = "", room-ansi.StringWidth(left)
	}
	return borderStyle.Render(leftCorner + left + strings.Repeat("─", fill) + right + rightCorner)
}

// row is one line inside the card, padded to its width.
func row(content string, inner int) string {
	pad := inner - ansi.StringWidth(content)
	return borderStyle.Render("│") + " " + content + strings.Repeat(" ", max(0, pad)) + " " + borderStyle.Render("│")
}

// block wraps text to width and fits it to exactly lines lines, ending the
// last with an ellipsis when some is cut.
func block(text string, width, lines int) []string {
	var wrapped []string
	if text != "" {
		for _, paragraph := range strings.Split(text, "\n") {
			wrapped = append(wrapped, strings.Split(ansi.Wrap(paragraph, width, ""), "\n")...)
		}
	}
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

func clamp(v, low, high int) int { return max(low, min(high, v)) }
