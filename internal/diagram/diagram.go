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
	Comment      template.HTML
	TotalColumns int
	HasSwimlanes bool
	SliceColumns []sliceColumnData
	SliceNames   []sliceNameData
	Rows         []rowData
	Connectors   []connectorData
}

// connectorData links two consecutive elements of a slice. From and To are
// anchor names; Direction is "down", "up" or "flat", from From to To.
type connectorData struct {
	From      template.CSS
	To        template.CSS
	Direction string
}

type sliceColumnData struct {
	ChildIndex int
	StartCol   int
	Span       int
}

type sliceNameData struct {
	DisplayName string
	Comment     template.HTML
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
	Anchor   template.CSS
	Props    []propData
}

type testData struct {
	Name     string
	Comment  template.HTML
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

func (g *Generator) buildDiagramData(doc *ast.Document) (diagramData, error) {
	hash := contentHash(doc.RawSource)

	var overrides []cssOverride
	if len(g.CSSOverrides) > 0 {
		css := renameCSSVariables(g.CSSOverrides)
		keys := make([]string, 0, len(css))
		for k := range css {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			overrides = append(overrides, cssOverride{Key: template.CSS(k), Value: template.CSS(css[k])})
		}
	}

	var docs []documentData
	for i, sd := range doc.SubDocs {
		d, err := buildDocumentData(hash, i, sd)
		if err != nil {
			return diagramData{}, err
		}
		docs = append(docs, d)
	}

	return diagramData{
		Overrides: overrides,
		Documents: docs,
	}, nil
}

func buildDocumentData(hash string, idx int, sd *ast.SubDoc) (documentData, error) {
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
		comment, err := renderComment(sd.Slices[name].Comment)
		if err != nil {
			return documentData{}, err
		}
		names = append(names, sliceNameData{DisplayName: displayName, Comment: comment})
	}

	docID := documentID(hash, idx)
	anchors := elementAnchors(docID, sd)

	// Rows
	var rows []rowData

	// Trigger rows (one per swimlane)
	for _, lane := range l.triggerLanes {
		rows = append(rows, buildElementRow(l, sd, anchors, "emlang-row-triggers", lane, func(e *ast.Element) bool {
			return e.Type == ast.ElementTrigger && e.Swimlane == lane
		}))
	}

	// Main row (commands + views)
	if l.hasMainRow {
		rows = append(rows, buildElementRow(l, sd, anchors, "emlang-row-main", "", func(e *ast.Element) bool {
			return e.Type == ast.ElementCommand || e.Type == ast.ElementView
		}))
	}

	// Event rows (one per swimlane)
	for _, lane := range l.eventLanes {
		rows = append(rows, buildElementRow(l, sd, anchors, "emlang-row-events", lane, func(e *ast.Element) bool {
			return (e.Type == ast.ElementEvent || e.Type == ast.ElementException) && e.Swimlane == lane
		}))
	}

	// Tests row
	if hasTests(sd) {
		row, err := buildTestsRow(l, sd)
		if err != nil {
			return documentData{}, err
		}
		rows = append(rows, row)
	}

	comment, err := renderComment(sd.Comment)
	if err != nil {
		return documentData{}, err
	}

	return documentData{
		ID:           docID,
		Comment:      comment,
		TotalColumns: l.totalColumns,
		HasSwimlanes: l.hasSwimlanes,
		SliceColumns: cols,
		SliceNames:   names,
		Rows:         rows,
		Connectors:   buildConnectors(l, sd, anchors),
	}, nil
}

// elementAnchors gives each element of sd a CSS anchor name, unique on the
// page because it starts with the document id. The name is made of the id
// and a counter only, which makes it safe to insert as template.CSS.
func elementAnchors(docID string, sd *ast.SubDoc) map[*ast.Element]template.CSS {
	anchors := map[*ast.Element]template.CSS{}
	for _, name := range sd.SliceOrder {
		for _, elem := range sd.Slices[name].Elements {
			anchors[elem] = template.CSS(fmt.Sprintf("--%s-%d", docID, len(anchors)+1))
		}
	}
	return anchors
}

// rowIndex returns the position of the row holding elem, counted from the
// top, with the row order of buildDocumentData.
func rowIndex(l *layout, elem *ast.Element) int {
	lanePos := func(lanes []string, lane string) int {
		for i, s := range lanes {
			if s == lane {
				return i
			}
		}
		return 0
	}
	switch elem.Type {
	case ast.ElementTrigger:
		return lanePos(l.triggerLanes, elem.Swimlane)
	case ast.ElementCommand, ast.ElementView:
		return len(l.triggerLanes)
	default:
		return len(l.triggerLanes) + 1 + lanePos(l.eventLanes, elem.Swimlane)
	}
}

// buildConnectors links each element to the next one in its slice. The
// browser places each connector between its two elements with CSS anchor
// positioning, so the HTML needs no script. The generator only knows
// whether the connector goes down, up or stays in the same row, and picks the
// diagonal to draw from that.
func buildConnectors(l *layout, sd *ast.SubDoc, anchors map[*ast.Element]template.CSS) []connectorData {
	var connectors []connectorData
	for _, name := range sd.SliceOrder {
		elems := sd.Slices[name].Elements
		for i := 1; i < len(elems); i++ {
			from, to := rowIndex(l, elems[i-1]), rowIndex(l, elems[i])
			dir := "flat"
			if to > from {
				dir = "down"
			} else if to < from {
				dir = "up"
			}
			connectors = append(connectors, connectorData{
				From:      anchors[elems[i-1]],
				To:        anchors[elems[i]],
				Direction: dir,
			})
		}
	}
	return connectors
}

func buildElementRow(l *layout, sd *ast.SubDoc, anchors map[*ast.Element]template.CSS, class string, lane string, match func(*ast.Element) bool) rowData {
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
					Anchor:   anchors[elem],
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

func buildTestsRow(l *layout, sd *ast.SubDoc) (rowData, error) {
	var slices []rowSliceData
	for _, name := range l.sliceOrder {
		slice := sd.Slices[name]
		var tests []testData
		for _, tn := range slice.TestOrder {
			test := slice.Tests[tn]
			comment, err := renderComment(test.Comment)
			if err != nil {
				return rowData{}, err
			}
			tests = append(tests, testData{
				Name:     test.Name,
				Comment:  comment,
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
	}, nil
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

	data, err := g.buildDiagramData(doc)
	if err != nil {
		return nil, fmt.Errorf("rendering comment: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "diagram", data); err != nil {
		return nil, fmt.Errorf("executing diagram template: %w", err)
	}

	return buf.Bytes(), nil
}
