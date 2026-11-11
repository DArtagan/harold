package filter

import (
	"strings"
	"testing"
)

var account = Account{
	Inbox:    "Inbox",
	Projects: []string{"Inbox", "Work", "Home Repairs", "Errands", "Q&A"},
	Labels:   []string{"waiting", "errands", "home"},
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments is Todoist's Today view", nil, "today | overdue"},
		{"today alone is strictly today", []string{"today"}, "today"},
		{"inbox names the Inbox project", []string{"inbox"}, "#Inbox"},
		{"a project, case-insensitively", []string{"work"}, "#Work"},
		{"a label", []string{"waiting"}, "@waiting"},
		{"words are combined with &", []string{"work", "today"}, "#Work & today"},
		{"a quoted name with spaces", []string{"home repairs"}, "#Home Repairs"},
		{"unquoted, the longest name wins", []string{"home", "repairs"}, "#Home Repairs"},
		{"a shorter name when the longer one is absent", []string{"home", "today"}, "@home & today"},
		{"adjacent unresolved words are one phrase", []string{"next", "week"}, "next week"},
		{"a phrase beside a name", []string{"work", "next", "week"}, "#Work & next week"},
		{"Todoist syntax alone passes verbatim", []string{"#Work | @waiting"}, "#Work | @waiting"},
		{"Todoist syntax combined is parenthesised", []string{"work", "@waiting | p1"}, "#Work & (@waiting | p1)"},
		{"a colon marks Todoist syntax", []string{"due before: friday"}, "due before: friday"},
		{"blank arguments are no arguments", []string{" "}, "today | overdue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args, account)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// "q&a" contains an operator, so it is passed through rather than resolved;
// a name holding an operator is reached through the words around it instead.
func TestResolveEscapesNamesItResolves(t *testing.T) {
	got, err := Resolve([]string{"q", "and", "a"}, Account{Projects: []string{"Q and A", "A|B"}})
	if err != nil || got != "#Q and A" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got := projectTerm("A|B (old)"); got != `#A\|B \(old\)` {
		t.Errorf("projectTerm = %q", got)
	}
}

func TestResolveRefusesANameThatIsBothProjectAndLabel(t *testing.T) {
	_, err := Resolve([]string{"errands"}, account)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"#Errands", "@errands"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not offer %s", err, want)
		}
	}
}

func TestResolveRefusesAnOverlongQuery(t *testing.T) {
	if _, err := Resolve([]string{strings.Repeat("x", MaxQueryLength+1)}, account); err == nil {
		t.Fatal("expected an error")
	}
}
