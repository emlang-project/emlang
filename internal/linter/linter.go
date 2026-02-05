package linter

import (
	"fmt"

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
