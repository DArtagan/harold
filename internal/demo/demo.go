// Package demo holds synthetic tasks, for `harold --demo`, golden files and
// screenshots. No real task is ever committed (AGENTS.md); these stand in.
package demo

import (
	"time"

	"github.com/DArtagan/harold/internal/filter"
	"github.com/DArtagan/harold/internal/triage"
)

// Query is the query the demo tasks are shown under: harold's default.
const Query = filter.DefaultQuery

// Today is the day the golden files are drawn on, so they never change with
// the date.
var Today = time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)

// Tasks returns the demo tasks, dated around today. Between them they cover
// what the screen has to handle: no description, a long one, several lines, a
// long title; overdue by days and by one, today with a time, tomorrow, a later
// date, a recurring rule, and no date.
func Tasks(today time.Time) []triage.Task {
	y, m, d := today.Date()
	day := func(offset int) time.Time { return time.Date(y, m, d+offset, 0, 0, 0, 0, today.Location()) }
	return []triage.Task{
		{
			ID:          "demo-1",
			Title:       "Water the ferns",
			Description: "The big one by the window drinks twice what the others do.",
			Project:     "Home",
			Due:         &triage.Due{Day: day(-2), Rule: "every 3 days"},
		},
		{
			ID:      "demo-2",
			Title:   "Return the library books",
			Project: "Errands",
			Labels:  []string{"out"},
			Due:     &triage.Due{Day: day(-1)},
		},
		{
			ID:    "demo-3",
			Title: "Ask Doris about the Saturday market",
			Description: "There was mention of a stall that sells seedlings in spring. " +
				"Find out which one, and whether they take cards, and whether " +
				"it is the first Saturday of the month or every Saturday.",
			Project: "Inbox",
			Labels:  []string{"waiting"},
			Due: &triage.Due{
				Day: day(0),
				At:  time.Date(y, m, d, 17, 30, 0, 0, today.Location()),
			},
		},
		{
			ID:          "demo-4",
			Title:       "Fix the squeaky hinge on the back gate",
			Description: "WD-40 is under the sink.\nIf that fails, the hardware shop on Elm Street.",
			Project:     "Home Repairs",
			Due:         &triage.Due{Day: day(1)},
		},
		{
			ID: "demo-5",
			Title: "Draft the newsletter for the neighbourhood association, " +
				"including the summer picnic date and the call for volunteers",
			Description: "Last year's is in the shared folder.",
			Project:     "Volunteering",
			Due:         &triage.Due{Day: day(9)},
		},
		{
			ID:      "demo-6",
			Title:   "Book the car in for its service",
			Project: "Errands",
		},
	}
}
