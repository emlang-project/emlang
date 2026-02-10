package formatter

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/emlang-project/emlang/internal/ast"
)

// Options controls formatting behaviour.
type Options struct {
	KeyStyle string // "short" or "long" (default "short")
}

// typeKey returns the YAML key for an element type based on key style.
func typeKey(t ast.ElementType, style string) string {
	if style == "short" {
		switch t {
		case ast.ElementTrigger:
			return "t"
		case ast.ElementCommand:
			return "c"
		case ast.ElementEvent:
			return "e"
		case ast.ElementException:
			return "x"
		case ast.ElementView:
			return "v"
		}
	}
	return t.String()
}

// Format renders the AST document as canonical YAML.
func Format(doc *ast.Document, opts Options) []byte {
	if opts.KeyStyle == "" {
		opts.KeyStyle = "short"
	}

	var buf bytes.Buffer
	w := &writer{buf: &buf, style: opts.KeyStyle}

	for i, sd := range doc.SubDocs {
		if i > 0 {
			w.raw("---\n")
		}
		w.writeSubDoc(sd)
	}

	return buf.Bytes()
}

type writer struct {
	buf   *bytes.Buffer
	style string
}

func (w *writer) raw(s string) {
	w.buf.WriteString(s)
}

func (w *writer) indent(level int) {
	for i := 0; i < level*2; i++ {
		w.buf.WriteByte(' ')
	}
}

func (w *writer) line(level int, s string) {
	w.indent(level)
	w.buf.WriteString(s)
	w.buf.WriteByte('\n')
}

func (w *writer) writeSubDoc(sd *ast.SubDoc) {
	w.raw("slices:\n")

	for _, name := range sd.SliceOrder {
		slice := sd.Slices[name]
		w.writeSlice(name, slice)
	}
}

func (w *writer) writeSlice(name string, slice *ast.Slice) {
	w.line(1, fmt.Sprintf("%s:", name))

	hasTests := len(slice.Tests) > 0

	if hasTests {
		// Extended form: steps + tests
		if len(slice.Elements) > 0 {
			w.line(2, "steps:")
			w.writeElementList(3, slice.Elements)
		}
		w.line(2, "tests:")
		w.writeTests(slice.Tests)
	} else {
		// Direct form: list of elements
		w.writeElementList(2, slice.Elements)
	}
}

func (w *writer) writeElementList(level int, elems []*ast.Element) {
	for _, elem := range elems {
		w.writeElement(level, elem)
	}
}

func (w *writer) writeElement(level int, elem *ast.Element) {
	name := elem.Name
	if elem.Swimlane != "" {
		name = elem.Swimlane + "/" + name
	}

	key := typeKey(elem.Type, w.style)

	if len(elem.Props) == 0 {
		w.indent(level)
		w.raw(fmt.Sprintf("- %s: %s\n", key, name))
		return
	}

	w.indent(level)
	w.raw(fmt.Sprintf("- %s: %s\n", key, name))
	w.indent(level + 1)
	w.raw("props:\n")
	w.writeProps(level+2, elem.Props)
}

func (w *writer) writeProps(level int, props []ast.PropEntry) {
	for _, p := range props {
		w.indent(level)
		w.raw(fmt.Sprintf("%s: %s\n", p.Key, formatValue(p.Value)))
	}
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (w *writer) writeTests(tests map[string]*ast.Test) {
	// Sort test names for deterministic output
	names := make([]string, 0, len(tests))
	for n := range tests {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		test := tests[name]
		w.writeTest(name, test)
	}
}

func (w *writer) writeTest(name string, test *ast.Test) {
	w.line(3, fmt.Sprintf("%s:", name))

	if test.HasGiven {
		if len(test.Given) == 0 {
			w.line(4, "given:")
		} else {
			w.line(4, "given:")
			w.writeElementList(5, test.Given)
		}
	}

	if test.HasWhen {
		if len(test.When) == 0 {
			w.line(4, "when:")
		} else {
			w.line(4, "when:")
			w.writeElementList(5, test.When)
		}
	}

	if test.HasThen {
		if len(test.Then) == 0 {
			w.line(4, "then:")
		} else {
			w.line(4, "then:")
			w.writeElementList(5, test.Then)
		}
	}
}
