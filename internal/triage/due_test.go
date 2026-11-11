package triage

import (
	"testing"
	"time"
)

func TestDueLabel(t *testing.T) {
	today := time.Date(2026, 11, 11, 15, 0, 0, 0, time.Local)
	day := func(month time.Month, d, year int) time.Time {
		return time.Date(year, month, d, 0, 0, 0, 0, time.Local)
	}
	tests := []struct {
		name string
		due  *Due
		want string
	}{
		{"no date", nil, ""},
		{"days overdue", &Due{Day: day(11, 8, 2026)}, "3 days overdue"},
		{"one day overdue", &Due{Day: day(11, 10, 2026)}, "1 day overdue"},
		{"a year overdue", &Due{Day: day(11, 11, 2025)}, "365 days overdue"},
		{"today", &Due{Day: day(11, 11, 2026)}, "today"},
		{"tomorrow", &Due{Day: day(11, 12, 2026)}, "tomorrow"},
		{"later this year", &Due{Day: day(11, 16, 2026)}, "Mon 16 Nov"},
		{"another year", &Due{Day: day(1, 4, 2027)}, "Mon 4 Jan 2027"},
		{"with a time", &Due{Day: day(11, 11, 2026), At: time.Date(2026, 11, 11, 9, 30, 0, 0, time.Local)}, "today 9:30am"},
		{"recurring", &Due{Day: day(11, 9, 2026), Rule: "every monday"}, "2 days overdue · every monday"},
		{"recurring with a time", &Due{Day: day(11, 12, 2026), At: time.Date(2026, 11, 12, 18, 0, 0, 0, time.Local), Rule: "every day at 6pm"}, "tomorrow 6:00pm · every day at 6pm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.due.Label(today); got != tt.want {
				t.Errorf("Label = %q, want %q", got, tt.want)
			}
		})
	}
}

// Across a change to daylight saving a day is 23 or 25 hours; the count is
// of days on the calendar.
func TestDueLabelAcrossDaylightSaving(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone database:", err)
	}
	// Clocks went back on 1 November 2026 in New York.
	today := time.Date(2026, 11, 2, 0, 30, 0, 0, newYork)
	due := &Due{Day: time.Date(2026, 10, 31, 0, 0, 0, 0, newYork)}
	if got := due.Label(today); got != "2 days overdue" {
		t.Errorf("Label = %q", got)
	}
}
