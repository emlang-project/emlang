package formatter

import (
	"strings"
	"testing"

	"github.com/emlang-project/emlang/internal/ast"
	"github.com/emlang-project/emlang/internal/parser"
)

func mustFormat(t *testing.T, doc *ast.Document, opts Options) []byte {
	t.Helper()
	out, err := Format(doc, opts)
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	return out
}

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

	out := mustFormat(t, doc, Options{KeyStyle: "long"})

	doc2, err := parser.Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := mustFormat(t, doc2, Options{KeyStyle: "long"})
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

	out := mustFormat(t, doc, Options{KeyStyle: "long"})

	doc2, err := parser.Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := mustFormat(t, doc2, Options{KeyStyle: "long"})
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

	out := string(mustFormat(t, doc, Options{KeyStyle: "long"}))

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

	out := string(mustFormat(t, doc, Options{KeyStyle: "short"}))

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

	out := string(mustFormat(t, doc, Options{KeyStyle: "long"}))

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

	out := string(mustFormat(t, doc, Options{KeyStyle: "long"}))

	if !strings.Contains(out, "---") {
		t.Errorf("expected multi-document separator, got:\n%s", out)
	}

	doc2, err := parser.Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	out2 := string(mustFormat(t, doc2, Options{KeyStyle: "long"}))
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

	out := string(mustFormat(t, doc, Options{KeyStyle: "long"}))

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

	out2 := string(mustFormat(t, doc2, Options{KeyStyle: "long"}))
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

	out := string(mustFormat(t, doc, Options{KeyStyle: "long"}))

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
	out := string(mustFormat(t, doc, Options{}))
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

	out := string(mustFormat(t, doc, Options{KeyStyle: "long"}))

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

func TestComments_StepsCollapsedToDirectForm(t *testing.T) {
	input := `# comment
slices:
  # comment
  SomeSlice:
    # comment
    steps:
      # comment
      - e: SomeEvent
`
	expected := `# comment
slices:
  # comment
  SomeSlice:
    # comment
    # comment
    - e: SomeEvent
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != expected {
		t.Errorf("unexpected output:\ngot:\n%s\nwant:\n%s", out, expected)
	}
}

func TestComments_Preserved(t *testing.T) {
	input := `# file header

# top
slices: # on slices
  # s1
  A: # on A
    # first elem
    - t: T1 # on T1
    # second elem
    - c: C1
      # on props
      props:
        # on k
        k: v # line k
  # before B
  B:
    steps:
      - e: E1
    # after steps
    tests: # on tests
      # on t1
      t1:
        # on given
        given: # line given
          - e: E0
        when:
          - c: C1
        then:
          # first then
          - e: E1
# end of file
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != input {
		t.Errorf("unexpected output:\ngot:\n%s\nwant:\n%s", out, input)
	}
}

func TestComments_Idempotent(t *testing.T) {
	input := `# header
slices:
  A:
    - c: C1
      props:
        k: v
        # after k
    # after C1
  # after A
---
# second doc
slices:
  B:
    - e: E1 # on E1
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out := mustFormat(t, doc, Options{KeyStyle: "short"})

	for _, c := range []string{"# header", "# after k", "# after C1", "# after A", "# second doc", "# on E1"} {
		if !strings.Contains(string(out), c) {
			t.Errorf("comment %q lost:\n%s", c, out)
		}
	}

	doc2, err := parser.Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	out2 := mustFormat(t, doc2, Options{KeyStyle: "short"})
	if string(out) != string(out2) {
		t.Errorf("roundtrip mismatch:\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}

func TestProps_ValuesPreserved(t *testing.T) {
	input := `slices:
  s:
    - c: C1
      props:
        a: "123"
        b: 'true'
        c: "x: y"
        d: [1, 2]
        e:
          nested: {k: v}
        f: |
          line1
          line2
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != input {
		t.Errorf("props changed:\ngot:\n%s\nwant:\n%s", out, input)
	}

	if _, err := parser.Parse(strings.NewReader(string(out))); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
}

func TestTestOrderPreserved(t *testing.T) {
	input := `slices:
  s:
    steps:
      - c: Foo
    tests:
      zeta:
        when:
          - c: Foo
      alpha:
        when:
          - c: Foo
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != input {
		t.Errorf("test order changed:\ngot:\n%s\nwant:\n%s", out, input)
	}
}

func TestKeysReordered(t *testing.T) {
	input := `slices:
  s:
    tests:
      t1:
        then:
          - e: Done
        when:
          - c: Do
        given:
          - e: Ready
    steps:
      - props:
          k: v
        c: Do
`
	expected := `slices:
  s:
    steps:
      - c: Do
        props:
          k: v
    tests:
      t1:
        given:
          - e: Ready
        when:
          - c: Do
        then:
          - e: Done
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != expected {
		t.Errorf("keys not reordered:\ngot:\n%s\nwant:\n%s", out, expected)
	}
}

func TestSwimlaneTrimmed(t *testing.T) {
	input := `slices:
  s:
    - c: " Lane / Name "
    - e: "  Other  "
`
	expected := `slices:
  s:
    - c: "Lane/Name"
    - e: "Other"
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != expected {
		t.Errorf("name not trimmed:\ngot:\n%s\nwant:\n%s", out, expected)
	}
}

func TestComments_DroppedTestsKey(t *testing.T) {
	input := `slices:
  A:
    steps:
      - c: C1
        props:
          q: "1"
    # on tests
    tests: # line tests
  B:
    - t: T
`
	expected := `slices:
  A:
    - c: C1
      props:
        q: "1"
        # on tests
        # line tests
  B:
    - t: T
`

	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out := mustFormat(t, doc, Options{KeyStyle: "short"})
	if string(out) != expected {
		t.Fatalf("unexpected output:\ngot:\n%s\nwant:\n%s", out, expected)
	}

	doc2, err := parser.Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if out2 := mustFormat(t, doc2, Options{KeyStyle: "short"}); string(out) != string(out2) {
		t.Errorf("roundtrip mismatch:\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}
