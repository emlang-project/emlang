package diagram

import (
	"bytes"
	"crypto/sha1"
	"embed"
	"fmt"
	"html/template"
	"sort"

	"github.com/emlang-project/emlang/internal/ast"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.gohtml"))

// Generator generates HTML diagrams from an AST.
type Generator struct {
	CSSOverrides map[string]string
}

// New creates a new diagram Generator.
func New() *Generator {
	return &Generator{}
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

// --- Template data structures ---

type diagramData struct {
	Overrides []cssOverride
	Documents []documentData
}

type cssOverride struct {
	Key   template.CSS
	Value template.CSS
}

type documentData struct {
	ID           string
	TotalColumns int
	HasSwimlanes bool
	SliceColumns []sliceColumnData
	SliceNames   []sliceNameData
	Rows         []rowData
}

type sliceColumnData struct {
	ChildIndex int
	StartCol   int
	Span       int
}

type sliceNameData struct {
	DisplayName string
}

type rowData struct {
	Class        string
	HasSwimlanes bool
	Swimlane     string
	Slices       []rowSliceData
}

type rowSliceData struct {
	Elements []elementData
	Tests    []testData
}

type elementData struct {
	CSSClass string
	Name     string
	GridCol  int
	Props    []propData
}

type testData struct {
	Name     string
	HasGiven bool
	Given    []elementData
	HasWhen  bool
	When     []elementData
	HasThen  bool
	Then     []elementData
}

type propData struct {
	Key   string
	Value string
}

// --- Build template data ---

func (g *Generator) buildDiagramData(doc *ast.Document) diagramData {
	hash := contentHash(doc.RawSource)

	var overrides []cssOverride
	if len(g.CSSOverrides) > 0 {
		keys := make([]string, 0, len(g.CSSOverrides))
		for k := range g.CSSOverrides {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			overrides = append(overrides, cssOverride{Key: template.CSS(k), Value: template.CSS(g.CSSOverrides[k])})
		}
	}

	var docs []documentData
	for i, sd := range doc.SubDocs {
		docs = append(docs, buildDocumentData(hash, i, sd))
	}

	return diagramData{
		Overrides: overrides,
		Documents: docs,
	}
}

func buildDocumentData(hash string, idx int, sd *ast.SubDoc) documentData {
	l := computeLayout(sd)

	// Slice columns for CSS
	var cols []sliceColumnData
	if l.hasSwimlanes {
		cols = append(cols, sliceColumnData{ChildIndex: 1, StartCol: 1, Span: 1})
		for i, name := range sd.SliceOrder {
			cols = append(cols, sliceColumnData{
				ChildIndex: i + 2,
				StartCol:   l.sliceStartCol[name],
				Span:       l.sliceWidths[name],
			})
		}
	} else {
		for i, name := range sd.SliceOrder {
			cols = append(cols, sliceColumnData{
				ChildIndex: i + 1,
				StartCol:   l.sliceStartCol[name],
				Span:       l.sliceWidths[name],
			})
		}
	}

	// Slice names
	var names []sliceNameData
	for _, name := range l.sliceOrder {
		displayName := name
		if displayName == "" {
			displayName = "(anonymous)"
		}
		names = append(names, sliceNameData{DisplayName: displayName})
	}

	// Rows
	var rows []rowData

	// Trigger rows (one per swimlane)
	for _, lane := range l.triggerLanes {
		rows = append(rows, buildElementRow(l, sd, "emlang-row-triggers", lane, func(e *ast.Element) bool {
			return e.Type == ast.ElementTrigger && e.Swimlane == lane
		}))
	}

	// Main row (commands + views)
	if l.hasMainRow {
		rows = append(rows, buildElementRow(l, sd, "emlang-row-main", "", func(e *ast.Element) bool {
			return e.Type == ast.ElementCommand || e.Type == ast.ElementView
		}))
	}

	// Event rows (one per swimlane)
	for _, lane := range l.eventLanes {
		rows = append(rows, buildElementRow(l, sd, "emlang-row-events", lane, func(e *ast.Element) bool {
			return (e.Type == ast.ElementEvent || e.Type == ast.ElementException) && e.Swimlane == lane
		}))
	}

	// Tests row
	if hasTests(sd) {
		rows = append(rows, buildTestsRow(l, sd))
	}

	return documentData{
		ID:           documentID(hash, idx),
		TotalColumns: l.totalColumns,
		HasSwimlanes: l.hasSwimlanes,
		SliceColumns: cols,
		SliceNames:   names,
		Rows:         rows,
	}
}

func buildElementRow(l *layout, sd *ast.SubDoc, class string, lane string, match func(*ast.Element) bool) rowData {
	var slices []rowSliceData
	for _, name := range l.sliceOrder {
		slice := sd.Slices[name]
		var elems []elementData
		for _, elem := range slice.Elements {
			if match(elem) {
				elems = append(elems, elementData{
					CSSClass: "emlang-" + elem.Type.String(),
					Name:     elem.Name,
					GridCol:  elementIndex(slice, elem),
					Props:    buildProps(elem.Props),
				})
			}
		}
		slices = append(slices, rowSliceData{Elements: elems})
	}
	return rowData{
		Class:        class,
		HasSwimlanes: l.hasSwimlanes,
		Swimlane:     lane,
		Slices:       slices,
	}
}

func hasTests(sd *ast.SubDoc) bool {
	for _, name := range sd.SliceOrder {
		if len(sd.Slices[name].Tests) > 0 {
			return true
		}
	}
	return false
}

func buildTestsRow(l *layout, sd *ast.SubDoc) rowData {
	var slices []rowSliceData
	for _, name := range l.sliceOrder {
		slice := sd.Slices[name]
		var tests []testData
		for _, tn := range slice.TestOrder {
			test := slice.Tests[tn]
			tests = append(tests, testData{
				Name:     test.Name,
				HasGiven: test.HasGiven,
				Given:    buildTestElements(test.Given),
				HasWhen:  test.HasWhen,
				When:     buildTestElements(test.When),
				HasThen:  test.HasThen,
				Then:     buildTestElements(test.Then),
			})
		}
		slices = append(slices, rowSliceData{Tests: tests})
	}
	return rowData{
		Class:        "emlang-row-tests",
		HasSwimlanes: l.hasSwimlanes,
		Slices:       slices,
	}
}

func buildTestElements(elems []*ast.Element) []elementData {
	var result []elementData
	for _, elem := range elems {
		result = append(result, elementData{
			CSSClass: "emlang-" + elem.Type.String(),
			Name:     elem.Name,
			Props:    buildProps(elem.Props),
		})
	}
	return result
}

func buildProps(props []ast.PropEntry) []propData {
	if len(props) == 0 {
		return nil
	}
	result := make([]propData, len(props))
	for i, p := range props {
		result[i] = propData{
			Key:   p.Key,
			Value: fmt.Sprintf("%v", p.Value),
		}
	}
	return result
}

// Generate creates an HTML diagram from the given document.
func (g *Generator) Generate(doc *ast.Document) ([]byte, error) {
	if len(doc.SubDocs) == 0 {
		return []byte(""), nil
	}

	data := g.buildDiagramData(doc)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "diagram", data); err != nil {
		return nil, fmt.Errorf("executing diagram template: %w", err)
	}

	return buf.Bytes(), nil
}
