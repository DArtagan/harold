package todoist

import (
	"strings"
	"testing"
	"time"
)

// A sidebar, as the API might return it, shuffled:
//
//	Inbox
//	Home
//	  Garden
//	    Seeds
//	  Repairs
//	Work
//	Errands   (ties with Work on child_order; order_key puts it after)
var sidebar = []Project{
	{ID: "work", Name: "Work", ChildOrder: 2, OrderKey: "b"},
	{ID: "seeds", Name: "Seeds", ParentID: "garden", ChildOrder: 1},
	{ID: "repairs", Name: "Repairs", ParentID: "home", ChildOrder: 2},
	{ID: "inbox", Name: "Inbox", InboxProject: true, ChildOrder: 5},
	{ID: "errands", Name: "Errands", ChildOrder: 2, OrderKey: "c"},
	{ID: "garden", Name: "Garden", ParentID: "home", ChildOrder: 1},
	{ID: "home", Name: "Home", ChildOrder: 1},
}

func TestSidebarOrder(t *testing.T) {
	rank := SidebarOrder(sidebar)
	names := make([]string, len(rank))
	for _, p := range sidebar {
		names[rank[p.ID]] = p.Name
	}
	if got := strings.Join(names, ", "); got != "Inbox, Home, Garden, Seeds, Repairs, Work, Errands" {
		t.Errorf("order: %s", got)
	}
}

func TestSidebarOrderRanksAnOrphanAtTheTop(t *testing.T) {
	rank := SidebarOrder([]Project{
		{ID: "a", ChildOrder: 2},
		{ID: "orphan", ParentID: "gone", ChildOrder: 1},
	})
	if rank["orphan"] != 0 || rank["a"] != 1 {
		t.Errorf("rank %v", rank)
	}
}

func order(tasks []Task) string {
	var ids []string
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return strings.Join(ids, " ")
}

func TestSortForTriageGoesProjectByProject(t *testing.T) {
	tasks := []Task{
		{ID: "w1", ProjectID: "work"},
		{ID: "x", ProjectID: "elsewhere"},
		{ID: "s1", ProjectID: "seeds"},
		{ID: "w2", ProjectID: "work"},
		{ID: "i1", ProjectID: "inbox"},
		{ID: "s2", ProjectID: "seeds"},
	}
	SortForTriage(tasks, sidebar, time.UTC)
	if got := order(tasks); got != "i1 s1 s2 w1 w2 x" {
		t.Errorf("order: %s", got)
	}
}

func TestSortForTriageWithinAProject(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone database:", err)
	}
	due := func(date string) *Due { return &Due{Date: date} }
	// Priority 4 is p1 in the app.
	tasks := []Task{
		{ID: "p4-undated", Priority: 1},
		{ID: "p4-12th", Priority: 1, Due: due("2026-11-12")},
		{ID: "p1-allday-11th", Priority: 4, Due: due("2026-11-11")},
		{ID: "p1-undated", Priority: 4},
		{ID: "p1-11th-9am", Priority: 4, Due: due("2026-11-11T09:00:00")},
		// 13:30 UTC is 08:30 in New York, before 09:00 floating.
		{ID: "p1-11th-0830-fixed", Priority: 4, Due: due("2026-11-11T13:30:00Z")},
		{ID: "p1-10th", Priority: 4, Due: due("2026-11-10")},
		{ID: "p2-9th", Priority: 3, Due: due("2026-11-09")},
		{ID: "p1-unreadable", Priority: 4, Due: due("someday")},
	}
	for i := range tasks {
		tasks[i].ProjectID = "work"
	}
	SortForTriage(tasks, sidebar, newYork)
	want := "p1-10th p1-11th-0830-fixed p1-11th-9am p1-allday-11th p1-undated p1-unreadable p2-9th p4-12th p4-undated"
	if got := order(tasks); got != want {
		t.Errorf("order:\n got %s\nwant %s", got, want)
	}
}

// A fixed time late in the UTC day falls on the previous day in New York, and
// is compared on that day.
func TestAFixedTimeIsReadInTheLocalDay(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone database:", err)
	}
	late := &Due{Date: "2026-11-12T02:00:00Z"} // 21:00 on the 11th in New York
	allDay12th := &Due{Date: "2026-11-12"}
	if !late.before(allDay12th, newYork) {
		t.Error("21:00 on the 11th sorted after the 12th")
	}
}
