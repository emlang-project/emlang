package linter

import (
	"fmt"
	"strings"

	"github.com/emlang-project/emlang/internal/ast"
)

// Severity represents the severity level of a linting issue.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Issue represents a linting issue found in the code.
type Issue struct {
	Rule     string
	Message  string
	Line     int
	Column   int
	Severity Severity
}

func (i Issue) String() string {
	return fmt.Sprintf("%d:%d: %s: %s (%s)", i.Line, i.Column, i.Severity, i.Message, i.Rule)
}

// Linter analyzes an AST for potential issues.
type Linter struct {
	issues      []Issue
	IgnoreRules map[string]bool
}

// New creates a new Linter.
func New() *Linter {
	return &Linter{
		issues:      []Issue{},
		IgnoreRules: map[string]bool{},
	}
}

// Lint analyzes the given document and returns any issues found.
func (l *Linter) Lint(doc *ast.Document) []Issue {
	l.issues = []Issue{}

	for _, sd := range doc.SubDocs {
		for _, name := range sd.SliceOrder {
			l.lintSlice(name, sd.Slices[name])
		}
		l.lintTestElements(sd)
	}

	return l.issues
}

func (l *Linter) addIssue(rule, message string, line, column int, severity Severity) {
	if l.IgnoreRules[rule] {
		return
	}
	l.issues = append(l.issues, Issue{
		Rule:     rule,
		Message:  message,
		Line:     line,
		Column:   column,
		Severity: severity,
	})
}

func (l *Linter) lintSlice(name string, slice *ast.Slice) {
	// Empty slice is valid (placeholder)
	if len(slice.Elements) == 0 {
		return
	}

	// Check slice structure
	hasEvent := false
	hasCommandInSeq := false

	for i, elem := range slice.Elements {
		if elem.Type == ast.ElementEvent {
			hasEvent = true
		}

		if elem.Type == ast.ElementCommand {
			hasCommandInSeq = true
			if !l.isFollowedByEventOrException(slice.Elements, i) {
				l.addIssue("command-without-event",
					"command should be followed by an event or exception",
					elem.Line, elem.Column, SeverityWarning)
			}
		}

		if elem.Type == ast.ElementException {
			if !hasCommandInSeq {
				l.addIssue("orphan-exception",
					"exception without preceding command",
					elem.Line, elem.Column, SeverityWarning)
			}
		}
	}

	if !hasEvent {
		l.addIssue("slice-missing-event",
			fmt.Sprintf("slice %q has no events", name),
			0, 0, SeverityWarning)
	}

}

func (l *Linter) isFollowedByEventOrException(elements []*ast.Element, index int) bool {
	for i := index + 1; i < len(elements); i++ {
		switch elements[i].Type {
		case ast.ElementEvent, ast.ElementException:
			return true
		case ast.ElementCommand:
			return false
		}
	}
	return false
}

// lintTestElements reports each test element that matches no step of any
// slice of the document, which is most often a typo. Each YAML document is
// checked on its own, as SPEC.md (Document Structure) makes them
// independent. Exceptions in then are not checked: a failure case usually
// has its exception in tests only, as in the examples of SPEC.md (Tests).
func (l *Linter) lintTestElements(sd *ast.SubDoc) {
	var steps []*ast.Element
	for _, name := range sd.SliceOrder {
		steps = append(steps, sd.Slices[name].Elements...)
	}

	for _, name := range sd.SliceOrder {
		slice := sd.Slices[name]
		for _, testName := range slice.TestOrder {
			test := slice.Tests[testName]
			var elems []*ast.Element
			elems = append(elems, test.Given...)
			elems = append(elems, test.When...)
			elems = append(elems, test.Then...)
			for _, elem := range elems {
				if elem.Type == ast.ElementException || matchesStep(elem, steps) {
					continue
				}
				message := fmt.Sprintf("%s %q is not a step of any slice", elem.Type, fullName(elem))
				if suggestion := closestStep(elem, steps); suggestion != "" {
					message += fmt.Sprintf(" (did you mean %q?)", suggestion)
				}
				l.addIssue("unknown-test-element", message, elem.Line, elem.Column, SeverityWarning)
			}
		}
	}
}

// matchesStep reports whether a step has the type and name of elem. A test
// element without a swimlane matches a step in any swimlane.
func matchesStep(elem *ast.Element, steps []*ast.Element) bool {
	for _, step := range steps {
		if step.Type == elem.Type && step.Name == elem.Name &&
			(elem.Swimlane == "" || step.Swimlane == elem.Swimlane) {
			return true
		}
	}
	return false
}

// closestStep returns the step of the same type whose name is closest to
// the name of elem, or "" when no step is close enough to be a likely typo.
// The comparison ignores case, so a difference in case only is suggested.
func closestStep(elem *ast.Element, steps []*ast.Element) string {
	target := strings.ToLower(comparedName(elem, elem))
	maxDistance := max(1, len([]rune(target))/3)

	best, bestDistance := "", maxDistance+1
	for _, step := range steps {
		if step.Type != elem.Type {
			continue
		}
		d := distance(target, strings.ToLower(comparedName(elem, step)))
		if d < bestDistance {
			best, bestDistance = fullName(step), d
		}
	}
	return best
}

// comparedName returns the name of e as compared with elem: with its
// swimlane when elem has one, without it otherwise.
func comparedName(elem, e *ast.Element) string {
	if elem.Swimlane == "" {
		return e.Name
	}
	return fullName(e)
}

// fullName returns the name of e as written in the source, swimlane first.
func fullName(e *ast.Element) string {
	if e.Swimlane == "" {
		return e.Name
	}
	return e.Swimlane + "/" + e.Name
}

// distance returns the Levenshtein distance between a and b, in runes.
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
