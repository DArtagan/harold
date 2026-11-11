// Package demo holds synthetic tasks, for `harold --demo`, golden files and
// screenshots. No real task is ever committed (AGENTS.md); these stand in.
package demo

import (
	"github.com/DArtagan/harold/internal/filter"
	"github.com/DArtagan/harold/internal/triage"
)

// Query is the query the demo tasks are shown under: harold's default.
const Query = filter.DefaultQuery

// Tasks returns the demo tasks. Between them they cover what the screen has to
// handle: no description, a long one, several lines, a long title.
func Tasks() []triage.Task {
	return []triage.Task{
		{
			ID:          "demo-1",
			Title:       "Water the ferns",
			Description: "The big one by the window drinks twice what the others do.",
			Project:     "Home",
			Due:         "today",
		},
		{
			ID:      "demo-2",
			Title:   "Return the library books",
			Project: "Errands",
			Labels:  []string{"out"},
			Due:     "today",
		},
		{
			ID:    "demo-3",
			Title: "Ask Doris about the Saturday market",
			Description: "There was mention of a stall that sells seedlings in spring. " +
				"Find out which one, and whether they take cards, and whether " +
				"it is the first Saturday of the month or every Saturday.",
			Project: "Inbox",
			Labels:  []string{"waiting"},
			Due:     "today",
		},
		{
			ID:          "demo-4",
			Title:       "Fix the squeaky hinge on the back gate",
			Description: "WD-40 is under the sink.\nIf that fails, the hardware shop on Elm Street.",
			Project:     "Home Repairs",
			Due:         "today",
		},
		{
			ID: "demo-5",
			Title: "Draft the newsletter for the neighbourhood association, " +
				"including the summer picnic date and the call for volunteers",
			Description: "Last year's is in the shared folder.",
			Project:     "Volunteering",
			Due:         "today",
		},
		{
			ID:      "demo-6",
			Title:   "Book the car in for its service",
			Project: "Errands",
			Due:     "today",
		},
	}
}
