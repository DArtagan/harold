// Package filter turns harold's arguments into one Todoist filter query.
//
// The rules are in docs/design.md, under Filters. Resolve is pure: the
// account's projects and labels are passed in, so the rules are tested without
// Todoist.
package filter

import (
	"fmt"
	"strings"
)

// DefaultQuery is the query when harold is given no arguments: what Todoist's
// own Today view shows, which includes overdue tasks. The filter `today` alone
// does not.
const DefaultQuery = "today | overdue"

// MaxQueryLength is the longest query Todoist accepts.
const MaxQueryLength = 1024

// operators are the characters that mark an argument as Todoist's own syntax,
// passed through verbatim.
const operators = "#@&|!(),:"

// Account is what Resolve needs to know about the account.
type Account struct {
	// Inbox is the name of the account's Inbox project, which the word
	// "inbox" names whatever the project is called.
	Inbox    string
	Projects []string
	Labels   []string
}

// term is one operand of the final query.
type term struct {
	text string
	// verbatim is true for an argument passed through as Todoist syntax,
	// which is parenthesised when combined, so its own | binds inside it.
	verbatim bool
}

// Resolve returns the query for args.
//
//   - An argument containing one of Todoist's operators passes through
//     verbatim, parenthesised when it is combined with anything else.
//   - The longest run of words naming a project or label, compared
//     case-insensitively, becomes #Project or @label. A run that names both
//     is an error rather than a guess.
//   - Adjacent words that name neither stay together as one phrase, so
//     "next week" reaches Todoist as a date.
//   - The terms are combined with &.
func Resolve(args []string, account Account) (string, error) {
	if len(args) == 0 {
		return DefaultQuery, nil
	}

	var terms []term
	var words []string
	flush := func() error {
		resolved, err := resolveWords(words, account)
		terms = append(terms, resolved...)
		words = nil
		return err
	}
	for _, arg := range args {
		if strings.ContainsAny(arg, operators) {
			if err := flush(); err != nil {
				return "", err
			}
			terms = append(terms, term{text: strings.TrimSpace(arg), verbatim: true})
			continue
		}
		words = append(words, strings.Fields(arg)...)
	}
	if err := flush(); err != nil {
		return "", err
	}
	if len(terms) == 0 {
		return DefaultQuery, nil
	}

	parts := make([]string, len(terms))
	for i, t := range terms {
		parts[i] = t.text
		if t.verbatim && len(terms) > 1 {
			parts[i] = "(" + t.text + ")"
		}
	}
	query := strings.Join(parts, " & ")
	if len(query) > MaxQueryLength {
		return "", fmt.Errorf("the query is %d characters, and Todoist accepts at most %d", len(query), MaxQueryLength)
	}
	return query, nil
}

func resolveWords(words []string, account Account) ([]term, error) {
	var terms []term
	var phrase []string
	endPhrase := func() {
		if len(phrase) > 0 {
			terms = append(terms, term{text: strings.Join(phrase, " ")})
			phrase = nil
		}
	}
	for i := 0; i < len(words); {
		name, n, err := longestName(words[i:], account)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			phrase = append(phrase, words[i])
			i++
			continue
		}
		endPhrase()
		terms = append(terms, term{text: name})
		i += n
	}
	endPhrase()
	return terms, nil
}

// longestName finds the longest run of words at the start of words that names
// a project or label, and returns its term and the number of words it took.
// It returns 0 words when the first word starts no name.
func longestName(words []string, account Account) (string, int, error) {
	for n := len(words); n > 0; n-- {
		candidate := strings.Join(words[:n], " ")
		project, isProject := lookup(candidate, account.Projects)
		if strings.EqualFold(candidate, "inbox") && account.Inbox != "" {
			project, isProject = account.Inbox, true
		}
		label, isLabel := lookup(candidate, account.Labels)
		switch {
		case isProject && isLabel:
			return "", 0, fmt.Errorf("%q names both the project %q and the label %q: write %s or %s",
				candidate, project, label, projectTerm(project), labelTerm(label))
		case isProject:
			return projectTerm(project), n, nil
		case isLabel:
			return labelTerm(label), n, nil
		}
	}
	return "", 0, nil
}

func lookup(name string, names []string) (string, bool) {
	for _, candidate := range names {
		if strings.EqualFold(candidate, name) {
			return candidate, true
		}
	}
	return "", false
}

func projectTerm(name string) string { return "#" + escape(name) }

func labelTerm(name string) string { return "@" + escape(name) }

// escape backslash-escapes the characters Todoist would read as operators
// inside a name.
func escape(name string) string {
	var b strings.Builder
	for _, r := range name {
		if strings.ContainsRune(`&|!(),\`, r) {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
