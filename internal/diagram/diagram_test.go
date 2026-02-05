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
	assertContains(t, out, `repeat(3, auto)`)
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
	assertContains(t, out, `repeat(5, auto)`)

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
		"--trigger-color": "#f0f0f0",
		"--command-color": "#ddeeff",
	}
	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	out := string(html)

	assertContains(t, out, `--trigger-color: #f0f0f0;`)
	assertContains(t, out, `--command-color: #ddeeff;`)
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
