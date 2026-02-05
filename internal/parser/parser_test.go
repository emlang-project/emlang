package parser

import (
	"strings"
	"testing"

	"github.com/emlang-project/emlang/internal/ast"
)

func TestParseEmptyDocument(t *testing.T) {
	input := ``
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected document, got nil")
	}
}

func TestParseSimpleSlice(t *testing.T) {
	input := `
slices:
  user-registration:
    - t: User/ClickRegister
    - c: RegisterUser
    - e: UserRegistered
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(doc.Slices))
	}

	slice, ok := doc.Slices["user-registration"]
	if !ok {
		t.Fatal("expected slice 'user-registration'")
	}

	if len(slice.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(slice.Elements))
	}

	// Check trigger
	if slice.Elements[0].Type != ast.ElementTrigger {
		t.Errorf("expected trigger, got %s", slice.Elements[0].Type)
	}
	if slice.Elements[0].Swimlane != "User" {
		t.Errorf("expected swimlane 'User', got %q", slice.Elements[0].Swimlane)
	}
	if slice.Elements[0].Name != "ClickRegister" {
		t.Errorf("expected name 'ClickRegister', got %q", slice.Elements[0].Name)
	}

	// Check command
	if slice.Elements[1].Type != ast.ElementCommand {
		t.Errorf("expected command, got %s", slice.Elements[1].Type)
	}

	// Check event
	if slice.Elements[2].Type != ast.ElementEvent {
		t.Errorf("expected event, got %s", slice.Elements[2].Type)
	}
}

func TestParseError_AnonymousSlice(t *testing.T) {
	input := `
slices:
  - t: Click
  - c: DoSomething
  - e: SomethingDone
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for anonymous slice (sequence)")
	}
}

func TestParseSliceWithProps(t *testing.T) {
	input := `
slices:
  checkout:
    - c: PlaceOrder
      props:
        customer_id: "123"
        total: 99.99
    - e: OrderPlaced
      props:
        order_id: "456"
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slice := doc.Slices["checkout"]
	if slice == nil {
		t.Fatal("expected slice 'checkout'")
	}

	cmd := slice.Elements[0]
	if cmd.Props == nil {
		t.Fatal("expected props on command")
	}
	if cmd.Props["customer_id"] != "123" {
		t.Errorf("expected customer_id '123', got %v", cmd.Props["customer_id"])
	}
}

func TestParseAllPrefixes(t *testing.T) {
	tests := []struct {
		prefix   string
		expected ast.ElementType
	}{
		{"t", ast.ElementTrigger},
		{"trg", ast.ElementTrigger},
		{"trigger", ast.ElementTrigger},
		{"c", ast.ElementCommand},
		{"cmd", ast.ElementCommand},
		{"command", ast.ElementCommand},
		{"e", ast.ElementEvent},
		{"evt", ast.ElementEvent},
		{"event", ast.ElementEvent},
		{"x", ast.ElementException},
		{"err", ast.ElementException},
		{"exception", ast.ElementException},
		{"v", ast.ElementView},
		{"view", ast.ElementView},
	}

	for _, tc := range tests {
		t.Run(tc.prefix, func(t *testing.T) {
			input := "slices:\n  test:\n    - " + tc.prefix + ": TestElement\n"
			doc, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			slice := doc.Slices["test"]
			if slice == nil {
				t.Fatal("expected slice 'test'")
			}
			if len(slice.Elements) != 1 {
				t.Fatalf("expected 1 element, got %d", len(slice.Elements))
			}
			if slice.Elements[0].Type != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, slice.Elements[0].Type)
			}
		})
	}
}

func TestParseError_InvalidYAML(t *testing.T) {
	input := `
slices:
  test:
    - invalid yaml here
      broken: indentation
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseError_ElementWithoutName(t *testing.T) {
	input := `
slices:
  test:
    - e:
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for element without name")
	}
}

func TestParseError_EmptyNameAfterSwimlane(t *testing.T) {
	input := `
slices:
  test:
    - e: ff/
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for empty name after swimlane")
	}
}

func TestParseError_WhitespaceNameAfterSwimlane(t *testing.T) {
	input := `
slices:
  test:
    - e: "ff/ "
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for whitespace-only name after swimlane")
	}
}

func TestParseEmptySlicePlaceholder(t *testing.T) {
	input := `
slices:
  FooBar:
  BarBaz:
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(doc.Slices))
	}

	foo := doc.Slices["FooBar"]
	if foo == nil {
		t.Fatal("expected slice 'FooBar'")
	}
	if len(foo.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(foo.Elements))
	}

	bar := doc.Slices["BarBaz"]
	if bar == nil {
		t.Fatal("expected slice 'BarBaz'")
	}
	if len(bar.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(bar.Elements))
	}
}

func TestParseError_UnknownKey(t *testing.T) {
	input := `
slices:
  test:
    - unknown: Element
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestParseError_MultipleTypes(t *testing.T) {
	input := `
slices:
  test:
    - t: Trigger
      c: Command
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for multiple type keys")
	}
}

func TestParseWhenMultipleCommands(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      multi-when:
        when:
          - c: First
          - c: Second
        then:
          - e: Done
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["MySlice"].Tests["multi-when"]
	if test == nil {
		t.Fatal("expected test 'multi-when'")
	}
	if len(test.When) != 2 {
		t.Errorf("expected 2 when elements, got %d", len(test.When))
	}
}

func TestParseEmptyTest(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      TodoTest:
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["MySlice"].Tests["TodoTest"]
	if test == nil {
		t.Fatal("expected test 'TodoTest'")
	}
	if len(test.Given) != 0 {
		t.Errorf("expected 0 given elements, got %d", len(test.Given))
	}
	if len(test.When) != 0 {
		t.Errorf("expected 0 when elements, got %d", len(test.When))
	}
	if len(test.Then) != 0 {
		t.Errorf("expected 0 then elements, got %d", len(test.Then))
	}
}

func TestParseTestWithoutWhen(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      GivenThenOnly:
        given:
          - e: SomeEvent
        then:
          - e: AnotherEvent
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["MySlice"].Tests["GivenThenOnly"]
	if test == nil {
		t.Fatal("expected test 'GivenThenOnly'")
	}
	if len(test.Given) != 1 {
		t.Errorf("expected 1 given element, got %d", len(test.Given))
	}
	if len(test.When) != 0 {
		t.Errorf("expected 0 when elements, got %d", len(test.When))
	}
	if len(test.Then) != 1 {
		t.Errorf("expected 1 then element, got %d", len(test.Then))
	}
}

func TestParseTestWithoutThen(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      WhenOnly:
        when:
          - c: DoSomething
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["MySlice"].Tests["WhenOnly"]
	if test == nil {
		t.Fatal("expected test 'WhenOnly'")
	}
	if len(test.When) != 1 {
		t.Errorf("expected 1 when element, got %d", len(test.When))
	}
	if len(test.Then) != 0 {
		t.Errorf("expected 0 then elements, got %d", len(test.Then))
	}
}

func TestParseSwimlane(t *testing.T) {
	input := `
slices:
  test:
    - t: Customer/ClickButton
    - e: System/EventFired
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slice := doc.Slices["test"]

	trigger := slice.Elements[0]
	if trigger.Swimlane != "Customer" {
		t.Errorf("expected swimlane 'Customer', got %q", trigger.Swimlane)
	}
	if trigger.Name != "ClickButton" {
		t.Errorf("expected name 'ClickButton', got %q", trigger.Name)
	}

	event := slice.Elements[1]
	if event.Swimlane != "System" {
		t.Errorf("expected swimlane 'System', got %q", event.Swimlane)
	}
	if event.Name != "EventFired" {
		t.Errorf("expected name 'EventFired', got %q", event.Name)
	}
}

func TestParseMultipleSlices(t *testing.T) {
	input := `
slices:
  slice-one:
    - c: CommandOne
    - e: EventOne
  slice-two:
    - c: CommandTwo
    - e: EventTwo
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(doc.Slices))
	}

	if doc.Slices["slice-one"] == nil {
		t.Error("expected slice 'slice-one'")
	}
	if doc.Slices["slice-two"] == nil {
		t.Error("expected slice 'slice-two'")
	}
}

func TestParseExtendedSlice(t *testing.T) {
	input := `
slices:
  UserRegistration:
    steps:
      - t: Customer/RegistrationForm
      - c: RegisterUser
      - e: UserRegistered
    tests:
      EmailMustBeUnique:
        given:
          - e: UserRegistered
        when:
          - c: RegisterUser
        then:
          - x: EmailAlreadyInUse
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(doc.Slices))
	}

	slice := doc.Slices["UserRegistration"]
	if slice == nil {
		t.Fatal("expected slice 'UserRegistration'")
	}

	// Check steps
	if len(slice.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(slice.Elements))
	}
	if slice.Elements[0].Type != ast.ElementTrigger {
		t.Errorf("expected trigger, got %s", slice.Elements[0].Type)
	}
	if slice.Elements[1].Type != ast.ElementCommand {
		t.Errorf("expected command, got %s", slice.Elements[1].Type)
	}
	if slice.Elements[2].Type != ast.ElementEvent {
		t.Errorf("expected event, got %s", slice.Elements[2].Type)
	}

	// Check attached tests
	if len(slice.Tests) != 1 {
		t.Fatalf("expected 1 attached test, got %d", len(slice.Tests))
	}

	test := slice.Tests["EmailMustBeUnique"]
	if test == nil {
		t.Fatal("expected test 'EmailMustBeUnique'")
	}

	if len(test.Given) != 1 {
		t.Errorf("expected 1 given element, got %d", len(test.Given))
	}
	if len(test.When) != 1 {
		t.Errorf("expected 1 when element, got %d", len(test.When))
	}
	if len(test.Then) != 1 {
		t.Errorf("expected 1 then element, got %d", len(test.Then))
	}
	if test.Then[0].Type != ast.ElementException {
		t.Errorf("expected exception, got %s", test.Then[0].Type)
	}
}

func TestParseMixedSliceForms(t *testing.T) {
	input := `
slices:
  DirectSlice:
    - c: DoSomething
    - e: SomethingDone

  ExtendedSlice:
    steps:
      - c: DoOther
      - e: OtherDone
    tests:
      TestOther:
        when:
          - c: DoOther
        then:
          - e: OtherDone
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(doc.Slices))
	}

	// Direct form
	direct := doc.Slices["DirectSlice"]
	if direct == nil {
		t.Fatal("expected slice 'DirectSlice'")
	}
	if len(direct.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(direct.Elements))
	}
	if len(direct.Tests) != 0 {
		t.Errorf("expected 0 tests for direct slice, got %d", len(direct.Tests))
	}

	// Extended form
	extended := doc.Slices["ExtendedSlice"]
	if extended == nil {
		t.Fatal("expected slice 'ExtendedSlice'")
	}
	if len(extended.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(extended.Elements))
	}
	if len(extended.Tests) != 1 {
		t.Errorf("expected 1 test for extended slice, got %d", len(extended.Tests))
	}
}

func TestParseExtendedSliceWithoutTests(t *testing.T) {
	input := `
slices:
  SliceWithSteps:
    steps:
      - c: DoSomething
      - e: SomethingDone
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slice := doc.Slices["SliceWithSteps"]
	if slice == nil {
		t.Fatal("expected slice 'SliceWithSteps'")
	}
	if len(slice.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(slice.Elements))
	}
}

func TestParseError_ExtendedSliceMissingSteps(t *testing.T) {
	input := `
slices:
  Invalid:
    tests:
      TestOnly:
        when:
          - c: DoSomething
        then:
          - e: SomethingDone
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for extended slice without steps")
	}
}

func TestParseSliceTestWithoutGiven(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      MyTest:
        when:
          - c: DoSomething
        then:
          - e: SomethingDone
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["MySlice"].Tests["MyTest"]
	if test == nil {
		t.Fatal("expected test 'MyTest'")
	}
	if len(test.Given) != 0 {
		t.Errorf("expected 0 given elements, got %d", len(test.Given))
	}
	if len(test.When) != 1 {
		t.Errorf("expected 1 when element, got %d", len(test.When))
	}
	if len(test.Then) != 1 {
		t.Errorf("expected 1 then element, got %d", len(test.Then))
	}
}

// v0.4.0: multi-document support

func TestParseMultiDocument(t *testing.T) {
	input := `---
slices:
  UserRegistration:
    - t: Customer/RegistrationForm
    - c: RegisterUser
    - e: UserRegistered
---
slices:
  UserDeletion:
    - t: Admin/UserManagement
    - c: DeleteUser
    - e: UserDeleted
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(doc.Slices))
	}

	if doc.Slices["UserRegistration"] == nil {
		t.Error("expected slice 'UserRegistration'")
	}
	if doc.Slices["UserDeletion"] == nil {
		t.Error("expected slice 'UserDeletion'")
	}

	// SubDocs
	if len(doc.SubDocs) != 2 {
		t.Fatalf("expected 2 SubDocs, got %d", len(doc.SubDocs))
	}
	if doc.SubDocs[0].Slices["UserRegistration"] == nil {
		t.Error("expected SubDoc[0] to contain 'UserRegistration'")
	}
	if doc.SubDocs[1].Slices["UserDeletion"] == nil {
		t.Error("expected SubDoc[1] to contain 'UserDeletion'")
	}
	if len(doc.SubDocs[0].SliceOrder) != 1 || doc.SubDocs[0].SliceOrder[0] != "UserRegistration" {
		t.Errorf("expected SubDoc[0] SliceOrder=['UserRegistration'], got %v", doc.SubDocs[0].SliceOrder)
	}
	if len(doc.SubDocs[1].SliceOrder) != 1 || doc.SubDocs[1].SliceOrder[0] != "UserDeletion" {
		t.Errorf("expected SubDoc[1] SliceOrder=['UserDeletion'], got %v", doc.SubDocs[1].SliceOrder)
	}
}

func TestParseMultiDocumentWithExtendedForm(t *testing.T) {
	input := `---
slices:
  UserRegistration:
    - t: Customer/RegistrationForm
    - c: RegisterUser
    - e: UserRegistered
---
slices:
  UserDeletion:
    steps:
      - t: Admin/UserManagement
      - c: DeleteUser
      - e: UserDeleted
    tests:
      CannotDeleteActiveUser:
        given:
          - e: UserAuthenticated
        when:
          - c: DeleteUser
        then:
          - x: UserCurrentlyActive
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Slices) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(doc.Slices))
	}

	deletion := doc.Slices["UserDeletion"]
	if deletion == nil {
		t.Fatal("expected slice 'UserDeletion'")
	}
	if len(deletion.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(deletion.Tests))
	}

	test := deletion.Tests["CannotDeleteActiveUser"]
	if test == nil {
		t.Fatal("expected test 'CannotDeleteActiveUser'")
	}
	if test.When == nil {
		t.Fatal("expected when element")
	}
	if len(test.Then) != 1 {
		t.Fatalf("expected 1 then element, got %d", len(test.Then))
	}
	if test.Then[0].Type != ast.ElementException {
		t.Errorf("expected exception, got %s", test.Then[0].Type)
	}
}

func TestParseError_UnknownTopLevelKey(t *testing.T) {
	input := `
slices:
  test:
    - c: DoSomething
    - e: SomethingDone
tests:
  my-test:
    when:
      - c: DoSomething
    then:
      - e: SomethingDone
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unknown top-level key 'tests'")
	}
}

func TestSubDocsBackwardsCompat(t *testing.T) {
	input := `
slices:
  slice-one:
    - c: CommandOne
    - e: EventOne
  slice-two:
    - c: CommandTwo
    - e: EventTwo
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Merged slices still work
	if len(doc.Slices) != 2 {
		t.Fatalf("expected 2 merged slices, got %d", len(doc.Slices))
	}

	// SubDocs populated
	if len(doc.SubDocs) != 1 {
		t.Fatalf("expected 1 SubDoc, got %d", len(doc.SubDocs))
	}

	sd := doc.SubDocs[0]
	if len(sd.Slices) != 2 {
		t.Fatalf("expected 2 slices in SubDoc, got %d", len(sd.Slices))
	}
	if len(sd.SliceOrder) != 2 {
		t.Fatalf("expected SliceOrder length 2, got %d", len(sd.SliceOrder))
	}

	// Same slice objects in both
	if doc.Slices["slice-one"] != sd.Slices["slice-one"] {
		t.Error("expected same Slice pointer in doc.Slices and SubDoc.Slices")
	}
}

func TestSubDocsSliceOrder(t *testing.T) {
	input := `
slices:
  alpha:
    - c: A
  beta:
    - c: B
  gamma:
    - c: C
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.SubDocs) != 1 {
		t.Fatalf("expected 1 SubDoc, got %d", len(doc.SubDocs))
	}

	order := doc.SubDocs[0].SliceOrder
	expected := []string{"alpha", "beta", "gamma"}
	if len(order) != len(expected) {
		t.Fatalf("expected SliceOrder %v, got %v", expected, order)
	}
	for i, name := range expected {
		if order[i] != name {
			t.Errorf("expected SliceOrder[%d]=%q, got %q", i, name, order[i])
		}
	}
}

func TestParseError_NameEndingWithSlash(t *testing.T) {
	input := `
slices:
  test:
    - e: foo/
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for element name ending with /")
	}
}

func TestParseError_JustSlash(t *testing.T) {
	input := `
slices:
  test:
    - e: /
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for element name that is just /")
	}
}

func TestParseError_EmptyDirectSlice(t *testing.T) {
	input := `
slices:
  empty-slice: []
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for empty direct slice")
	}
}

func TestParseNullSlices(t *testing.T) {
	input := `
slices:
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Slices) != 0 {
		t.Errorf("expected 0 slices, got %d", len(doc.Slices))
	}
}

func TestParseNullSteps(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice := doc.Slices["MySlice"]
	if slice == nil {
		t.Fatal("expected slice 'MySlice'")
	}
	if len(slice.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(slice.Elements))
	}
}

func TestParseError_EmptyStepsList(t *testing.T) {
	input := `
slices:
  MySlice:
    steps: []
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for empty steps list")
	}
}

func TestParseNullTests(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice := doc.Slices["MySlice"]
	if slice == nil {
		t.Fatal("expected slice 'MySlice'")
	}
	if len(slice.Tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(slice.Tests))
	}
}

func TestParseEmptyTestsMapping(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests: {}
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice := doc.Slices["MySlice"]
	if slice == nil {
		t.Fatal("expected slice 'MySlice'")
	}
	if len(slice.Tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(slice.Tests))
	}
}

func TestParseNullGivenWhenThen(t *testing.T) {
	input := `
slices:
  MySlice:
    steps:
      - c: DoSomething
      - e: SomethingDone
    tests:
      NullSections:
        given:
        when:
        then:
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["MySlice"].Tests["NullSections"]
	if test == nil {
		t.Fatal("expected test 'NullSections'")
	}
	if len(test.Given) != 0 {
		t.Errorf("expected 0 given elements, got %d", len(test.Given))
	}
	if len(test.When) != 0 {
		t.Errorf("expected 0 when elements, got %d", len(test.When))
	}
	if len(test.Then) != 0 {
		t.Errorf("expected 0 then elements, got %d", len(test.Then))
	}
}

func TestParseTestWithException(t *testing.T) {
	input := `
slices:
  PaymentFlow:
    steps:
      - c: ProcessPayment
      - e: PaymentProcessed
      - x: PaymentFailed
    tests:
      payment-fails:
        given:
          - e: OrderCreated
        when:
          - c: ProcessPayment
        then:
          - x: PaymentFailed
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test := doc.Slices["PaymentFlow"].Tests["payment-fails"]
	if test == nil {
		t.Fatal("expected test 'payment-fails'")
	}

	if len(test.Then) != 1 {
		t.Fatalf("expected 1 then element, got %d", len(test.Then))
	}
	if test.Then[0].Type != ast.ElementException {
		t.Errorf("expected exception in then, got %s", test.Then[0].Type)
	}
}
