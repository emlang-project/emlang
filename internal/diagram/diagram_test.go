package diagram

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"testing"

	"github.com/emlang-project/emlang/internal/parser"
)

func TestSimpleSlice(t *testing.T) {
	input := `
slices:
  user-registration:
    - t: ClickRegister
    - c: RegisterUser
    - e: UserRegistered
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `class="emlang-documents"`)
	assertContains(t, out, `class="emlang-document"`)
	assertContains(t, out, `class="emlang-row emlang-row-slices"`)
	assertContains(t, out, `class="emlang-slicename">user-registration</span>`)
	assertContains(t, out, `class="emlang-trigger"`)
	assertContains(t, out, `class="emlang-command"`)
	assertContains(t, out, `class="emlang-event"`)
	assertContains(t, out, `>ClickRegister</span>`)
	assertContains(t, out, `>RegisterUser</span>`)
	assertContains(t, out, `>UserRegistered</span>`)

	// grid-column positions (no swimlane column since no swimlanes)
	assertContains(t, out, `grid-column: 1`)
	assertContains(t, out, `grid-column: 2`)
	assertContains(t, out, `grid-column: 3`)

	// CSS: total columns = 3 (elements, no swimlane column)
	assertContains(t, out, `repeat(3, minmax(min-content, auto))`)
}

func TestExtendedSliceWithTests(t *testing.T) {
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
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `class="emlang-row emlang-row-tests"`)
	assertContains(t, out, `class="emlang-test"`)
	assertContains(t, out, `>EmailMustBeUnique</span>`)
	assertContains(t, out, `>GIVEN</span>`)
	assertContains(t, out, `>WHEN</span>`)
	assertContains(t, out, `>THEN</span>`)
	assertContains(t, out, `class="emlang-exception"`)
	assertContains(t, out, `>EmailAlreadyInUse</span>`)
}

func TestMultiDocuments(t *testing.T) {
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
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	hash := testHash(input)

	// Two documents with content-hash IDs
	assertContains(t, out, fmt.Sprintf(`id="emlang-document-%s-0"`, hash))
	assertContains(t, out, fmt.Sprintf(`id="emlang-document-%s-1"`, hash))

	// Both slice names present
	assertContains(t, out, `>UserRegistration</span>`)
	assertContains(t, out, `>UserDeletion</span>`)

	// Two separate CSS blocks for grid columns
	assertContains(t, out, fmt.Sprintf(`#emlang-document-%s-0`, hash))
	assertContains(t, out, fmt.Sprintf(`#emlang-document-%s-1`, hash))
}

func TestSwimlanes(t *testing.T) {
	input := `
slices:
  checkout:
    - t: Customer/ClickCheckout
    - c: PlaceOrder
    - e: Warehouse/OrderReady
    - e: Billing/InvoiceSent
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	// Trigger swimlane
	assertContains(t, out, `class="emlang-swimlane">Customer</span>`)

	// Event swimlanes
	assertContains(t, out, `class="emlang-swimlane">Warehouse</span>`)
	assertContains(t, out, `class="emlang-swimlane">Billing</span>`)

	// Multiple trigger rows and event rows
	count := strings.Count(out, `emlang-row-triggers`)
	if count < 1 {
		t.Errorf("expected at least 1 trigger row, got %d", count)
	}

	eventRowCount := strings.Count(out, `emlang-row-events`)
	if eventRowCount < 2 {
		t.Errorf("expected at least 2 event rows (for 2 swimlanes), got %d", eventRowCount)
	}
}

func TestProps(t *testing.T) {
	input := `
slices:
  checkout:
    - c: PlaceOrder
      props:
        customer_id: string
        total: number
    - e: OrderPlaced
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `class="emlang-props"`)
	assertContains(t, out, `<dt>customer_id</dt>`)
	assertContains(t, out, `<dd>string</dd>`)
	assertContains(t, out, `<dt>total</dt>`)
}

func TestGridColumnLayout(t *testing.T) {
	input := `
slices:
  slice-a:
    - c: CmdA
    - e: EvtA
  slice-b:
    - t: TrgB
    - c: CmdB
    - e: EvtB
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	// Total columns = 2 (slice-a) + 3 (slice-b) = 5 (no swimlane column)
	assertContains(t, out, `repeat(5, minmax(min-content, auto))`)

	// slice-a starts at col 1, span 2
	assertContains(t, out, `grid-column: 1 / span 2`)
	// slice-b starts at col 3, span 3
	assertContains(t, out, `grid-column: 3 / span 3`)
}

func TestEmptyDocument(t *testing.T) {
	input := ``
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()
	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if string(html) != "" {
		t.Errorf("expected empty output for empty document, got %q", string(html))
	}
}

func TestViewInMainRow(t *testing.T) {
	input := `
slices:
  flow:
    - c: ProcessPayment
    - v: OrderDetails
    - e: PaymentProcessed
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `class="emlang-view"`)
	assertContains(t, out, `>OrderDetails</span>`)
	assertContains(t, out, `class="emlang-row emlang-row-main"`)
}

func TestExceptionInEventRow(t *testing.T) {
	input := `
slices:
  flow:
    - c: ProcessPayment
    - e: PaymentProcessed
    - x: PaymentFailed
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `class="emlang-exception"`)
	assertContains(t, out, `>PaymentFailed</span>`)
}

func TestCSSOverrides(t *testing.T) {
	input := `
slices:
  checkout:
    - c: PlaceOrder
    - e: OrderPlaced
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()

	gen.CSSOverrides = map[string]string{
		"--trigger-bg-color": "#f0f0f0",
		"--command-bg-color": "#ddeeff",
	}
	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `--trigger-bg-color: #f0f0f0;`)
	assertContains(t, out, `--command-bg-color: #ddeeff;`)
}

func TestItemWidthVariables(t *testing.T) {
	doc, err := parser.Parse(strings.NewReader("slices:\n  s:\n    - e: A\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	html, err := New().Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `--item-width: auto;`)
	assertContains(t, out, `--item-min-width: 8em;`)
	assertContains(t, out, `--item-max-width: none;`)
	assertContains(t, out, `width: var(--item-width);`)
	assertContains(t, out, `min-width: var(--item-min-width);`)
	assertContains(t, out, `max-width: var(--item-max-width);`)
}

func TestDeprecatedCSSVariables(t *testing.T) {
	input := `
slices:
  checkout:
    - c: PlaceOrder
    - e: OrderPlaced
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()
	gen.CSSOverrides = map[string]string{
		"--event-color":        "#111111",
		"--font-size-label":    "0.8em",
		"--command-color":      "#222222",
		"--command-bg-color":   "#333333",
		"--item-border-radius": "0",
	}
	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `--event-bg-color: #111111;`)
	assertContains(t, out, `--label-font-size: 0.8em;`)
	assertContains(t, out, `--command-bg-color: #333333;`)
	assertContains(t, out, `--item-border-radius: 0;`)
	for _, old := range []string{"--event-color:", "--font-size-label:", "--command-color:", "#222222"} {
		if strings.Contains(out, old) {
			t.Errorf("output should not contain %q", old)
		}
	}
}

func TestRenamedCSSVariableCoversTemplate(t *testing.T) {
	css, err := templateFS.ReadFile("templates/css.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	for old, current := range renamedCSSVariables {
		if !strings.Contains(string(css), current+":") {
			t.Errorf("%s is renamed to %s, which the CSS template does not define", old, current)
		}
	}
	if current, ok := RenamedCSSVariable("--item-border-radius"); ok {
		t.Errorf("--item-border-radius should not be deprecated, got %s", current)
	}
}

func TestContentHashID(t *testing.T) {
	input := `
slices:
  checkout:
    - c: PlaceOrder
    - e: OrderPlaced
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := New()
	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)
	hash := testHash(input)

	// Document ID uses content hash
	assertContains(t, out, fmt.Sprintf(`id="emlang-document-%s-0"`, hash))
	assertContains(t, out, fmt.Sprintf(`#emlang-document-%s-0`, hash))

	// Common classes stay unchanged
	assertContains(t, out, `class="emlang-documents"`)
	assertContains(t, out, `.emlang-documents {`)
	assertContains(t, out, `.emlang-document {`)
	assertContains(t, out, `class="emlang-document"`)

	// Different input produces different hash
	input2 := `
slices:
  other:
    - c: DoSomething
    - e: SomethingDone
`
	doc2, err := parser.Parse(strings.NewReader(input2))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	html2, err := gen.Generate(doc2)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out2 := string(html2)
	hash2 := testHash(input2)

	if hash == hash2 {
		t.Errorf("expected different hashes for different inputs")
	}

	assertContains(t, out2, fmt.Sprintf(`id="emlang-document-%s-0"`, hash2))
}

func testHash(input string) string {
	h := sha1.Sum([]byte(input))
	return fmt.Sprintf("%x", h)[:12]
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q", needle)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected output not to contain %q", needle)
	}
}

func generate(t *testing.T, input string) string {
	t.Helper()
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	html, err := New().Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	return string(html)
}

func assertOrder(t *testing.T, out string, parts ...string) {
	t.Helper()
	pos := 0
	for _, p := range parts {
		i := strings.Index(out[pos:], p)
		if i < 0 {
			t.Fatalf("expected %q after position %d in output:\n%s", p, pos, out)
		}
		pos += i + len(p)
	}
}

func TestComments(t *testing.T) {
	out := generate(t, `# document comment
slices:
  # slice comment
  Slice1:
    steps:
      - c: Register user
      - e: User registered
    tests:
      # test comment
      Test1:
        when:
          - c: Register user
`)

	assertOrder(t, out,
		`<div class="emlang-comment emlang-document-comment"><p>document comment</p>`,
		`class="emlang-row emlang-row-slices"`,
		`<span class="emlang-slicename">Slice1</span>`,
		`<div class="emlang-comment"><p>slice comment</p>`,
		`<span>Test1</span>`,
		`<div class="emlang-comment"><p>test comment</p>`,
		`<span>WHEN</span>`,
	)
}

func TestNoComments(t *testing.T) {
	out := generate(t, `
slices:
  Slice1:
    - c: Register user
`)
	assertNotContains(t, out, `class="emlang-comment`)
}

func TestCommentMarkdown(t *testing.T) {
	out := generate(t, `slices:
  # **bold** and *italic*
  # see [the spec](https://example.com/spec)
  #
  # second paragraph
  Slice1:
    - c: Register user
`)
	assertContains(t, out, `<strong>bold</strong> and <em>italic</em><br>`)
	assertContains(t, out, `<a href="https://example.com/spec" target="_blank" rel="noopener noreferrer">the spec</a>`)
	assertContains(t, out, `<p>second paragraph</p>`)

	// Links take the comment text color
	assertOrder(t, out, `.emlang-comment {`, `a {`, `color: inherit;`)
}

func TestCommentGFM(t *testing.T) {
	out := generate(t, `slices:
  # | column 1 | column 2 |
  # | --- | --- |
  # | foo | bar |
  #
  # ~~removed~~ https://example.com
  #
  # - [x] done
  Slice1:
    - c: Register user
`)
	assertContains(t, out, `<th>column 1</th>`)
	assertContains(t, out, `<td>foo</td>`)
	assertContains(t, out, `<del>removed</del>`)
	assertContains(t, out, `<a href="https://example.com" target="_blank" rel="noopener noreferrer">https://example.com</a>`)
	assertContains(t, out, `type="checkbox"`)
}

func TestCommentUnsafe(t *testing.T) {
	out := generate(t, `slices:
  # <script>alert(1)</script>
  # [click](javascript:alert(1))
  Slice1:
    - c: Register user
`)
	assertNotContains(t, out, `<script>`)
	assertNotContains(t, out, `javascript:`)
}

func TestConnectors(t *testing.T) {
	input := `
slices:
  slice-a:
    - c: CmdA
    - e: EvtA
    - e: EvtB
    - v: ViewA
  slice-b:
    - c: CmdB
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	html, err := New().Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)
	id := documentID(contentHash([]byte(input)), 0)

	assertContains(t, out, `anchor-name: --`+id+`-1"`)
	assertContains(t, out, `anchor-name: --`+id+`-5"`)
	assertNotContains(t, out, `ZgotmplZ`)

	// Connectors link elements within a slice only.
	if n := strings.Count(out, `<div class="emlang-connector `); n != 3 {
		t.Errorf("got %d connectors, want 3", n)
	}
	assertContains(t, out, fmt.Sprintf(`class="emlang-connector emlang-connector-down" style="--connector-from: --%s-1; --connector-to: --%s-2"`, id, id))
	assertContains(t, out, fmt.Sprintf(`class="emlang-connector emlang-connector-flat" style="--connector-from: --%s-2; --connector-to: --%s-3"`, id, id))
	assertContains(t, out, fmt.Sprintf(`class="emlang-connector emlang-connector-up" style="--connector-from: --%s-3; --connector-to: --%s-4"`, id, id))
}
