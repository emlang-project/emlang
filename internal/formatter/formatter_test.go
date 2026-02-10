package formatter

import (
	"strings"
	"testing"

	"github.com/emlang-project/emlang/internal/parser"
)

func TestRoundtrip_DirectForm(t *testing.T) {
	input := `slices:
  Registration:
    - trigger: UserClicksRegister
    - command: RegisterUser
    - event: UserRegistered
    - view: UserProfile
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := Format(doc, Options{KeyStyle: "long"})

	doc2, err := parser.Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := Format(doc2, Options{KeyStyle: "long"})
	if string(out) != string(out2) {
		t.Errorf("roundtrip mismatch:\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}

func TestRoundtrip_ExtendedForm(t *testing.T) {
	input := `slices:
  Payment:
    steps:
      - command: ProcessPayment
      - event: PaymentProcessed
    tests:
      happy-path:
        given:
          - event: UserRegistered
        when:
          - command: ProcessPayment
        then:
          - event: PaymentProcessed
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := Format(doc, Options{KeyStyle: "long"})

	doc2, err := parser.Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := Format(doc2, Options{KeyStyle: "long"})
	if string(out) != string(out2) {
		t.Errorf("roundtrip mismatch:\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}

func TestAliasNormalization_ShortToLong(t *testing.T) {
	input := `slices:
  s:
    - t: Foo
    - c: Bar
    - e: Baz
    - x: Err
    - v: MyView
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "long"}))

	expected := `slices:
  s:
    - trigger: Foo
    - command: Bar
    - event: Baz
    - exception: Err
    - view: MyView
`

	if out != expected {
		t.Errorf("alias normalization failed:\ngot:\n%s\nwant:\n%s", out, expected)
	}
}

func TestAliasNormalization_LongToShort(t *testing.T) {
	input := `slices:
  s:
    - trigger: Foo
    - command: Bar
    - event: Baz
    - exception: Err
    - view: MyView
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "short"}))

	expected := `slices:
  s:
    - t: Foo
    - c: Bar
    - e: Baz
    - x: Err
    - v: MyView
`

	if out != expected {
		t.Errorf("short key normalization failed:\ngot:\n%s\nwant:\n%s", out, expected)
	}
}

func TestRoundtrip_Swimlane(t *testing.T) {
	input := `slices:
  s:
    - trigger: Web/UserClicks
    - command: Backend/DoThing
    - event: Backend/ThingDone
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "long"}))

	if !strings.Contains(out, "Web/UserClicks") {
		t.Errorf("expected swimlane preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "Backend/DoThing") {
		t.Errorf("expected swimlane preserved, got:\n%s", out)
	}
}

func TestRoundtrip_MultiDocument(t *testing.T) {
	input := `slices:
  a:
    - trigger: Foo
---
slices:
  b:
    - command: Bar
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "long"}))

	if !strings.Contains(out, "---") {
		t.Errorf("expected multi-document separator, got:\n%s", out)
	}

	doc2, err := parser.Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := string(Format(doc2, Options{KeyStyle: "long"}))
	if out != out2 {
		t.Errorf("roundtrip mismatch:\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}

func TestRoundtrip_Props(t *testing.T) {
	input := `slices:
  s:
    - command: CreateUser
      props:
        email: test@example.com
        required: true
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "long"}))

	if !strings.Contains(out, "props:") {
		t.Errorf("expected props in output, got:\n%s", out)
	}
	if !strings.Contains(out, "email: test@example.com") {
		t.Errorf("expected email prop, got:\n%s", out)
	}

	// Roundtrip
	doc2, err := parser.Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := string(Format(doc2, Options{KeyStyle: "long"}))
	if out != out2 {
		t.Errorf("roundtrip mismatch:\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}

func TestRoundtrip_TestSectionsPreserved(t *testing.T) {
	input := `slices:
  s:
    steps:
      - command: Foo
    tests:
      mytest:
        when:
          - command: Foo
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "long"}))

	// Should have "when:" but not "given:" or "then:" since they weren't in source
	if strings.Contains(out, "given:") {
		t.Errorf("should not have given section, got:\n%s", out)
	}
	if !strings.Contains(out, "when:") {
		t.Errorf("should have when section, got:\n%s", out)
	}
	if strings.Contains(out, "then:") {
		t.Errorf("should not have then section, got:\n%s", out)
	}
}

func TestDefaultKeyStyle(t *testing.T) {
	input := `slices:
  s:
    - t: Foo
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Default (empty) should behave as "short"
	out := string(Format(doc, Options{}))
	if !strings.Contains(out, "t: Foo") {
		t.Errorf("default key style should be short, got:\n%s", out)
	}
}

func TestMediumAliases_NormalizedToLong(t *testing.T) {
	input := `slices:
  s:
    - trg: Foo
    - cmd: Bar
    - evt: Baz
    - err: Qux
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := string(Format(doc, Options{KeyStyle: "long"}))

	expected := `slices:
  s:
    - trigger: Foo
    - command: Bar
    - event: Baz
    - exception: Qux
`

	if out != expected {
		t.Errorf("medium alias normalization:\ngot:\n%s\nwant:\n%s", out, expected)
	}
}
