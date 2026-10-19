package linter

import (
	"strings"
	"testing"

	"github.com/emlang-project/emlang/internal/ast"
	"github.com/emlang-project/emlang/internal/parser"
)

func parse(input string) (*ast.Document, error) {
	return parser.Parse(strings.NewReader(input))
}

func mustParse(t *testing.T, input string) *ast.Document {
	doc, err := parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return doc
}

func TestLintValidSlice(t *testing.T) {
	input := `
slices:
  order-slice:
    - t: User/SubmitOrder
    - c: CreateOrder
    - e: OrderCreated
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d:", len(issues))
		for _, issue := range issues {
			t.Errorf("  %s", issue)
		}
	}
}

func TestLintEmptySliceIsParseError(t *testing.T) {
	input := `
slices:
  empty-slice: []
`
	_, err := parse(input)
	if err == nil {
		t.Fatal("expected parse error for empty slice")
	}
}

func TestParseTriggerInTestGiven(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      my-test:
        given:
          - t: User/Click
        when:
          - c: DoSomething
        then:
          - e: SomethingDone
`
	_, err := parse(input)
	if err == nil {
		t.Fatal("expected parse error for trigger in given")
	}
}

func TestLintTestWithoutWhenIsValid(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      no-when-test:
        given:
          - e: SomeEvent
        then:
          - e: AnotherEvent
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func TestLintTestWithoutThenIsValid(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      no-outcome-test:
        when:
          - c: DoSomething
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func TestLintValidTest(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: PlaceOrder
      - e: OrderPlaced
    tests:
      valid-test:
        given:
          - e: UserLoggedIn
          - v: CartItems
        when:
          - c: PlaceOrder
        then:
          - e: OrderPlaced
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func TestParseExceptionInGiven(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      bad-given-test:
        given:
          - x: ErrorState
        when:
          - c: DoSomething
        then:
          - e: SomethingDone
`
	_, err := parse(input)
	if err == nil {
		t.Fatal("expected parse error for exception in given")
	}
}

func TestParseCommandInThen(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      bad-then-test:
        when:
          - c: DoSomething
        then:
          - c: AnotherCommand
`
	_, err := parse(input)
	if err == nil {
		t.Fatal("expected parse error for command in then")
	}
}

func TestLintSliceMissingEvent(t *testing.T) {
	input := `
slices:
  no-event-slice:
    - t: User/Click
    - c: DoSomething
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	found := false
	for _, issue := range issues {
		if issue.Rule == "slice-missing-event" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'slice-missing-event' issue")
	}
}

func TestLintCommandWithoutEvent(t *testing.T) {
	input := `
slices:
  dangling-command:
    - c: FirstCommand
    - c: SecondCommand
    - e: OnlyForSecond
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	found := false
	for _, issue := range issues {
		if issue.Rule == "command-without-event" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'command-without-event' issue")
	}
}

func TestLintOrphanException(t *testing.T) {
	input := `
slices:
  orphan-exception:
    - x: ErrorWithoutCommand
    - c: Command
    - e: Event
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	found := false
	for _, issue := range issues {
		if issue.Rule == "orphan-exception" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'orphan-exception' issue")
	}
}

func TestLintValidSliceWithException(t *testing.T) {
	input := `
slices:
  valid-exception:
    - c: RiskyCommand
    - e: SuccessEvent
    - x: FailureException
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	for _, issue := range issues {
		if issue.Rule == "orphan-exception" {
			t.Error("should not have 'orphan-exception' for valid exception placement")
		}
	}
}

func TestIssueSeverityString(t *testing.T) {
	if SeverityWarning.String() != "warning" {
		t.Errorf("expected 'warning', got %q", SeverityWarning.String())
	}
	if SeverityError.String() != "error" {
		t.Errorf("expected 'error', got %q", SeverityError.String())
	}
}

func TestIssueString(t *testing.T) {
	issue := Issue{
		Rule:     "test-rule",
		Message:  "test message",
		Line:     10,
		Column:   5,
		Severity: SeverityError,
	}

	expected := "10:5: error: test message (test-rule)"
	if issue.String() != expected {
		t.Errorf("expected %q, got %q", expected, issue.String())
	}
}

func TestLintTestWithViewInThen(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
      - v: UpdatedView
    tests:
      view-in-then:
        when:
          - c: DoSomething
        then:
          - e: SomethingDone
          - v: UpdatedView
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func TestLintTestWithExceptionInThen(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      exception-in-then:
        when:
          - c: DoSomething
        then:
          - x: SomethingFailed
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func TestParseInvalidWhenType(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      invalid-when:
        when:
          - e: ShouldBeCommand
        then:
          - e: SomethingDone
`
	_, err := parse(input)
	if err == nil {
		t.Fatal("expected parse error for event in when")
	}
}

func TestLintExtendedSliceWithValidTests(t *testing.T) {
	input := `
slices:
  UserRegistration:
    steps:
      - t: Customer/Form
      - c: RegisterUser
      - e: UserRegistered
    tests:
      EmailUnique:
        when:
          - c: RegisterUser
        then:
          - x: EmailAlreadyInUse
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func TestParseTriggerInExtendedTestGiven(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      InvalidTest:
        given:
          - t: TriggerNotAllowed
        when:
          - c: DoSomething
        then:
          - e: SomethingDone
`
	_, err := parse(input)
	if err == nil {
		t.Fatal("expected parse error for trigger in given")
	}
}

func TestLintIgnoredRulesAreSuppressed(t *testing.T) {
	input := `
slices:
  dangling-command:
    - c: FirstCommand
    - c: SecondCommand
    - e: OnlyForSecond
`
	doc := mustParse(t, input)

	linter := New()
	linter.IgnoreRules["command-without-event"] = true
	issues := linter.Lint(doc)

	for _, issue := range issues {
		if issue.Rule == "command-without-event" {
			t.Error("expected 'command-without-event' to be suppressed")
		}
	}
}

func TestLintEmptyTestIsValid(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      TodoTest:
`
	doc := mustParse(t, input)

	linter := New()
	issues := linter.Lint(doc)

	var errors []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d:", len(errors))
		for _, issue := range errors {
			t.Errorf("  %s", issue)
		}
	}
}

func unknownTestElements(issues []Issue) []Issue {
	var found []Issue
	for _, issue := range issues {
		if issue.Rule == "unknown-test-element" {
			found = append(found, issue)
		}
	}
	return found
}

func TestLintUnknownTestElement(t *testing.T) {
	input := `
slices:
  Registration:
    - c: Register user
    - e: User/User registered
  Login:
    steps:
      - c: Log in
      - e: User/Logged in
      - v: Dashboard
    tests:
      Known elements:
        given:
          - e: User registered
          - e: User/User registered
        when:
          - c: Log in
        then:
          - v: Dashboard
          - x: Bad password
      Unknown elements:
        given:
          - e: User registred
          - e: Admin/User registered
          - v: Register user
        when:
          - c: Delete account
        then:
          - e: logged in
`
	doc := mustParse(t, input)
	got := unknownTestElements(New().Lint(doc))

	want := []string{
		`event "User registred" is not a step of any slice (did you mean "User/User registered"?)`,
		`event "Admin/User registered" is not a step of any slice (did you mean "User/User registered"?)`,
		`view "Register user" is not a step of any slice`,
		`command "Delete account" is not a step of any slice`,
		`event "logged in" is not a step of any slice (did you mean "User/Logged in"?)`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d issues, want %d: %v", len(got), len(want), got)
	}
	for i, issue := range got {
		if issue.Message != want[i] {
			t.Errorf("issue %d: got %q, want %q", i, issue.Message, want[i])
		}
		if issue.Severity != SeverityWarning {
			t.Errorf("issue %d: got severity %s, want warning", i, issue.Severity)
		}
	}
	if got[0].Line != 23 || got[0].Column != 13 {
		t.Errorf("got position %d:%d, want 23:13", got[0].Line, got[0].Column)
	}
}

func TestLintUnknownTestElementPerDocument(t *testing.T) {
	input := `
slices:
  Registration:
    - c: Register user
    - e: User registered
---
slices:
  Login:
    steps:
      - c: Log in
      - e: Logged in
    tests:
      Needs registration:
        given:
          - e: User registered
        when:
          - c: Log in
`
	doc := mustParse(t, input)
	got := unknownTestElements(New().Lint(doc))

	if len(got) != 1 || got[0].Message != `event "User registered" is not a step of any slice` {
		t.Errorf("got %v, want one issue for \"User registered\"", got)
	}
}
