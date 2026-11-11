package triage

import (
	"reflect"
	"testing"
)

var tasks = []Task{{ID: "1", Title: "one"}, {ID: "2", Title: "two"}, {ID: "3", Title: "three"}}

func ids(tasks []Task) []string {
	out := []string{}
	for _, task := range tasks {
		out = append(out, task.ID)
	}
	return out
}

func TestNextMovesTheCardIntoTheLedger(t *testing.T) {
	s := New("today", tasks)
	if card, _ := s.Card(); card.ID != "1" {
		t.Fatalf("first card %q", card.ID)
	}
	s = s.Next().Next()
	if card, _ := s.Card(); card.ID != "3" {
		t.Errorf("card %q, want 3", card.ID)
	}
	if got := ids(s.Ledger()); !reflect.DeepEqual(got, []string{"1", "2"}) {
		t.Errorf("ledger %v", got)
	}
	s = s.Next()
	if _, ok := s.Card(); ok {
		t.Error("a card after the last task")
	}
	if s.Next().Considered() != 3 {
		t.Error("Next past the end changed the session")
	}
}

func TestSessionsAreValues(t *testing.T) {
	first := New("today", tasks)
	_ = first.Next()
	if first.Considered() != 0 {
		t.Error("Next changed the session it was called on")
	}
}

func TestRestoreReplaysAPosition(t *testing.T) {
	pos := New("today", tasks).Next().Next().Position()

	// A fresh fetch: task 1 was completed elsewhere, and a task 4 is new.
	fresh := []Task{{ID: "2"}, {ID: "3"}, {ID: "4"}}
	s := Restore("today", fresh, pos)
	if got := ids(s.Ledger()); !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("ledger %v", got)
	}
	if card, _ := s.Card(); card.ID != "3" {
		t.Errorf("card %q, want 3", card.ID)
	}
}

func TestRestoreIgnoresAnotherQuerysPosition(t *testing.T) {
	pos := New("today", tasks).Next().Position()
	if s := Restore("inbox", tasks, pos); s.Considered() != 0 {
		t.Error("restored a position from another query")
	}
}
