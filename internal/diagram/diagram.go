package diagram

import (
	"crypto/sha1"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/emlang-project/emlang/internal/ast"
)

// Generator generates HTML diagrams from an AST.
type Generator struct {
	CSSOverrides map[string]string
}

// New creates a new diagram Generator.
func New() *Generator {
	return &Generator{}
}

// writer wraps a strings.Builder and emits formatted whitespace.
type writer struct {
	b *strings.Builder
}

func (w *writer) write(s string) {
	w.b.WriteString(s)
}

func (w *writer) nl() {
	w.b.WriteByte('\n')
}

func (w *writer) indent(level int) {
	for i := 0; i < level*4; i++ {
		w.b.WriteByte(' ')
	}
}

// indentNL writes indent then content then newline.
func (w *writer) line(level int, s string) {
	w.indent(level)
	w.b.WriteString(s)
	w.nl()
}

// contentHash returns the first 12 hex characters of the SHA-1 hash of raw.
func contentHash(raw []byte) string {
	h := sha1.Sum(raw)
	return fmt.Sprintf("%x", h)[:12]
}

// documentID returns the HTML id for a subdocument,
// e.g. "emlang-document-2fd4e1c67a2d-0".
func documentID(hash string, idx int) string {
	return fmt.Sprintf("emlang-document-%s-%d", hash, idx)
}

// Generate creates an HTML diagram from the given document.
func (g *Generator) Generate(doc *ast.Document) ([]byte, error) {
	subDocs := doc.SubDocs
	if len(subDocs) == 0 {
		return []byte(""), nil
	}

	var b strings.Builder
	w := &writer{b: &b}

	hash := contentHash(doc.RawSource)

	// Generate per-document CSS
	w.write("<style>")
	w.nl()
	writeCommonCSS(w, g.CSSOverrides)
	for i, sd := range subDocs {
		writeDocumentCSS(w, hash, i, sd)
	}
	w.write("</style>")
	w.nl()

	// Generate HTML
	w.write(`<div class="emlang-documents">`)
	w.nl()
	for i, sd := range subDocs {
		writeDocument(w, hash, i, sd)
	}
	w.write("</div>")
	w.nl()

	return []byte(b.String()), nil
}

// layout holds precomputed layout info for a subdocument.
type layout struct {
	sliceOrder    []string
	sliceWidths   map[string]int // number of elements per slice
	totalColumns  int            // (1 swimlane if any) + sum of widths
	sliceStartCol map[string]int // grid-column start for each slice's div
	triggerLanes  []string       // unique swimlanes for triggers, in order
	eventLanes    []string       // unique swimlanes for events/exceptions, in order
	hasSwimlanes  bool           // true if any element has a swimlane
	hasMainRow    bool           // true if any element is a command or view
}

func computeLayout(sd *ast.SubDoc) *layout {
	l := &layout{
		sliceOrder:    sd.SliceOrder,
		sliceWidths:   make(map[string]int),
		sliceStartCol: make(map[string]int),
	}

	totalWidth := 0
	for _, name := range sd.SliceOrder {
		slice := sd.Slices[name]
		w := len(slice.Elements)
		if w == 0 {
			w = 1
		}
		l.sliceWidths[name] = w
		totalWidth += w
	}

	// Collect unique swimlanes by order of appearance
	triggerSeen := map[string]bool{}
	eventSeen := map[string]bool{}
	for _, name := range sd.SliceOrder {
		slice := sd.Slices[name]
		for _, elem := range slice.Elements {
			if elem.Swimlane != "" {
				l.hasSwimlanes = true
			}
			switch elem.Type {
			case ast.ElementTrigger:
				lane := elem.Swimlane
				if !triggerSeen[lane] {
					triggerSeen[lane] = true
					l.triggerLanes = append(l.triggerLanes, lane)
				}
			case ast.ElementCommand, ast.ElementView:
				l.hasMainRow = true
			case ast.ElementEvent, ast.ElementException:
				lane := elem.Swimlane
				if !eventSeen[lane] {
					eventSeen[lane] = true
					l.eventLanes = append(l.eventLanes, lane)
				}
			}
		}
	}

	// Swimlane column only when swimlanes are present
	if l.hasSwimlanes {
		l.totalColumns = 1 + totalWidth
		col := 2
		for _, name := range sd.SliceOrder {
			l.sliceStartCol[name] = col
			col += l.sliceWidths[name]
		}
	} else {
		l.totalColumns = totalWidth
		col := 1
		for _, name := range sd.SliceOrder {
			l.sliceStartCol[name] = col
			col += l.sliceWidths[name]
		}
	}

	return l
}

// elementIndex returns the 1-based position of an element within its slice.
func elementIndex(slice *ast.Slice, elem *ast.Element) int {
	for i, e := range slice.Elements {
		if e == elem {
			return i + 1
		}
	}
	return 1
}

// cssVariables contains the CSS custom properties for .emlang-documents.
// Overrides from config are injected after these variables.
const cssVariables = `    .emlang-documents {
        --text-color: #212529;
        --border-color: #ced4da;

        --trigger-color: #e9ecef;
        --command-color: #a5d8ff;
        --event-color: #ffd8a8;
        --exception-color: #ffc9c9;
        --view-color: #b2f2bb;
        --item-border-radius: 0.5em;

        --font-family-normal: system-ui;
        --font-family-props: monospace;

        --font-size-slicename: 2em;
        --font-weight-slicename: normal;
        --font-size-swimlane: 1.5em;
        --font-weight-swimlane: normal;
        --font-size-testname: 1em;
        --font-weight-testname: bold;
        --font-size-label: 0.75em;
        --font-weight-label: normal;
        --font-size-props: 0.75em;
        --font-weight-props: normal;
`

// cssRules contains the rest of the common CSS after variables and overrides.
const cssRules = `
        align-items: flex-start;
        color: var(--text-color);
        display: inline-flex;
        flex-direction: column;
        gap: 2em;
    }

    .emlang-document {
        *, *:after, *:before {
            box-sizing: border-box;
        }

        display: inline-grid;
        font-family: var(--font-family-normal), system-ui;

        .emlang-row {
            display: contents;

            & > div {
                align-items: flex-start;
                display: grid;
                gap: 1em;
                padding: 0.5em;

                &:not(:first-child) {
                    border-left: 1px solid var(--border-color);
                }
            }

            &:not(:last-child) > div {
                border-bottom: 1px solid var(--border-color);
            }

            &:not(.emlang-row-tests) > div {
                grid-template-columns: subgrid;
            }
        }

        .emlang-slicename {
            font-size: var(--font-size-slicename);
            font-weight: var(--font-weight-slicename);
            grid-column: 1 / -1;
        }

        .emlang-swimlane {
            font-size: var(--font-size-swimlane);
            font-weight: var(--font-weight-swimlane);
        }

        .emlang-trigger,
        .emlang-command,
        .emlang-view,
        .emlang-event,
        .emlang-exception {
            border-radius: var(--item-border-radius);
            display: inline-flex;
            flex-direction: column;
            gap: 0.5em;
            padding: 0.5em;
        }

        .emlang-trigger { background-color: var(--trigger-color); }
        .emlang-command { background-color: var(--command-color); }
        .emlang-view { background-color: var(--view-color); }
        .emlang-event { background-color: var(--event-color); }
        .emlang-exception { background-color: var(--exception-color); }

        .emlang-props {
            column-gap: 0.5em;
            display: inline-grid;
            grid-template-columns: auto auto;
            margin: 0;

            dt:after {
                content: ': ';
            }

            * {
                font-family: var(--font-family-props), monospace;
                font-size: var(--font-size-props);
                font-weight: var(--font-weight-props);
                margin: 0;
            }
        }

        .emlang-test {
            display: inline-grid;
            gap: 1em;
            grid-template-columns: auto 1fr;

            & > span:first-child {
                font-size: var(--font-size-testname);
                font-weight: var(--font-weight-testname);
                grid-column: 1/-1;
            }

            & > span:not(:first-child) {
                font-size: var(--font-size-label);
                font-weight: var(--font-weight-label);
            }

            &:not(:last-child) {
                border-bottom: 1px solid var(--border-color);
                padding-bottom: 1em;
            }

            div {
                align-items: flex-start;
                display: flex;
                flex-direction: column;
                gap: 0.5em;
            }
        }

    }
`

func writeCommonCSS(w *writer, overrides map[string]string) {
	w.write(cssVariables)
	if len(overrides) > 0 {
		keys := make([]string, 0, len(overrides))
		for k := range overrides {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			w.line(2, fmt.Sprintf(`%s: %s;`, k, overrides[k]))
		}
	}
	w.write(cssRules)
}

func writeDocumentCSS(w *writer, hash string, idx int, sd *ast.SubDoc) {
	l := computeLayout(sd)
	w.line(1, fmt.Sprintf("#%s {", documentID(hash, idx)))
	w.line(2, fmt.Sprintf("grid-template-columns: repeat(%d, auto);", l.totalColumns))
	w.nl()
	w.line(2, ".emlang-row {")

	if l.hasSwimlanes {
		// First child is column 1 (swimlane label)
		w.line(3, "& > div:nth-child(1) {")
		w.line(4, "grid-column: 1/2;")
		w.line(3, "}")

		// Each slice gets its column range
		for i, name := range sd.SliceOrder {
			w.nl()
			w.line(3, fmt.Sprintf("& > div:nth-child(%d) {", i+2))
			w.line(4, fmt.Sprintf("grid-column: %d / span %d;", l.sliceStartCol[name], l.sliceWidths[name]))
			w.line(3, "}")
		}
	} else {
		// No swimlane column; slices start at child 1
		for i, name := range sd.SliceOrder {
			if i > 0 {
				w.nl()
			}
			w.line(3, fmt.Sprintf("& > div:nth-child(%d) {", i+1))
			w.line(4, fmt.Sprintf("grid-column: %d / span %d;", l.sliceStartCol[name], l.sliceWidths[name]))
			w.line(3, "}")
		}
	}

	w.line(2, "}")
	w.line(1, "}")
	w.nl()
}

func writeDocument(w *writer, hash string, idx int, sd *ast.SubDoc) {
	l := computeLayout(sd)

	w.line(1, fmt.Sprintf(`<div id="%s" class="emlang-document">`, documentID(hash, idx)))

	// Row: slice names
	writeSliceNamesRow(w, l, sd)

	// Rows: triggers (one per swimlane)
	for _, lane := range l.triggerLanes {
		writeElementRow(w, l, sd, "emlang-row-triggers", lane, func(e *ast.Element) bool {
			return e.Type == ast.ElementTrigger && e.Swimlane == lane
		})
	}

	// Row: main (commands + views)
	if l.hasMainRow {
		writeElementRow(w, l, sd, "emlang-row-main", "", func(e *ast.Element) bool {
			return e.Type == ast.ElementCommand || e.Type == ast.ElementView
		})
	}

	// Rows: events (one per swimlane)
	for _, lane := range l.eventLanes {
		writeElementRow(w, l, sd, "emlang-row-events", lane, func(e *ast.Element) bool {
			return (e.Type == ast.ElementEvent || e.Type == ast.ElementException) && e.Swimlane == lane
		})
	}

	// Row: tests
	if hasTests(sd) {
		writeTestsRow(w, l, sd)
	}

	w.line(1, "</div>")
}

func writeSliceNamesRow(w *writer, l *layout, sd *ast.SubDoc) {
	w.line(2, `<div class="emlang-row emlang-row-slices">`)
	if l.hasSwimlanes {
		w.line(3, "<div></div>")
	}
	for _, name := range l.sliceOrder {
		displayName := name
		if displayName == "" {
			displayName = "(anonymous)"
		}
		w.line(3, "<div>")
		w.line(4, fmt.Sprintf(`<span class="emlang-slicename">%s</span>`, html.EscapeString(displayName)))
		w.line(3, "</div>")
	}
	w.line(2, "</div>")
}

// elementFilter returns true for elements that should appear in a row.
type elementFilter func(elem *ast.Element) bool

// writeElementRow writes a row of elements filtered by the given predicate.
// lane is the swimlane label to display (empty string for no label). rowClass is the CSS class suffix.
func writeElementRow(w *writer, l *layout, sd *ast.SubDoc, rowClass string, lane string, match elementFilter) {
	w.line(2, fmt.Sprintf(`<div class="emlang-row %s">`, rowClass))
	if l.hasSwimlanes {
		w.indent(3)
		w.write("<div>")
		if lane != "" {
			w.nl()
			w.line(4, fmt.Sprintf(`<span class="emlang-swimlane">%s</span>`, html.EscapeString(lane)))
			w.indent(3)
		}
		w.write("</div>")
		w.nl()
	}

	for _, name := range l.sliceOrder {
		slice := sd.Slices[name]
		w.indent(3)
		w.write("<div>")
		hasContent := false
		for _, elem := range slice.Elements {
			if match(elem) {
				hasContent = true
				gridCol := elementIndex(slice, elem)
				cssClass := "emlang-" + elem.Type.String()
				w.nl()
				w.line(4, fmt.Sprintf(`<div class="%s" style="grid-column: %d">`, cssClass, gridCol))
				w.line(5, fmt.Sprintf("<span>%s</span>", html.EscapeString(elem.Name)))
				writeProps(w, elem.Props, 5)
				w.line(4, "</div>")
			}
		}
		if hasContent {
			w.indent(3)
		}
		w.write("</div>")
		w.nl()
	}
	w.line(2, "</div>")
}

func hasTests(sd *ast.SubDoc) bool {
	for _, name := range sd.SliceOrder {
		if len(sd.Slices[name].Tests) > 0 {
			return true
		}
	}
	return false
}

func writeTestsRow(w *writer, l *layout, sd *ast.SubDoc) {
	w.line(2, `<div class="emlang-row emlang-row-tests">`)
	if l.hasSwimlanes {
		w.line(3, "<div></div>")
	}

	for _, name := range l.sliceOrder {
		slice := sd.Slices[name]
		w.indent(3)
		w.write("<div>")
		if len(slice.Tests) > 0 {
			// Sort test names for deterministic output
			testNames := make([]string, 0, len(slice.Tests))
			for tn := range slice.Tests {
				testNames = append(testNames, tn)
			}
			sort.Strings(testNames)

			for _, tn := range testNames {
				test := slice.Tests[tn]
				w.nl()
				w.line(4, `<div class="emlang-test">`)
				w.line(5, fmt.Sprintf("<span>%s</span>", html.EscapeString(test.Name)))

				// GIVEN
				if test.HasGiven {
					w.line(5, "<span>GIVEN</span>")
					w.line(5, "<div>")
					for _, elem := range test.Given {
						writeTestElement(w, elem, 6)
					}
					w.line(5, "</div>")
				}

				// WHEN
				if test.HasWhen {
					w.line(5, "<span>WHEN</span>")
					w.line(5, "<div>")
					for _, elem := range test.When {
						writeTestElement(w, elem, 6)
					}
					w.line(5, "</div>")
				}

				// THEN
				if test.HasThen {
					w.line(5, "<span>THEN</span>")
					w.line(5, "<div>")
					for _, elem := range test.Then {
						writeTestElement(w, elem, 6)
					}
					w.line(5, "</div>")
				}

				w.line(4, "</div>")
			}
			w.indent(3)
		}
		w.write("</div>")
		w.nl()
	}
	w.line(2, "</div>")
}

func writeTestElement(w *writer, elem *ast.Element, level int) {
	w.line(level, fmt.Sprintf(`<div class="emlang-%s">`, elem.Type.String()))
	w.line(level+1, fmt.Sprintf("<span>%s</span>", html.EscapeString(elem.Name)))
	writeProps(w, elem.Props, level+1)
	w.line(level, "</div>")
}

func writeProps(w *writer, props map[string]interface{}, level int) {
	if len(props) == 0 {
		return
	}
	// Sort keys for deterministic output
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w.line(level, `<dl class="emlang-props">`)
	for _, k := range keys {
		v := props[k]
		w.line(level+1, fmt.Sprintf("<dt>%s</dt>", html.EscapeString(k)))
		w.line(level+1, fmt.Sprintf("<dd>%s</dd>", html.EscapeString(fmt.Sprintf("%v", v))))
	}
	w.line(level, "</dl>")
}
