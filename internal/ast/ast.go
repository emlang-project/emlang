package ast

// SubDoc represents a single YAML document (separated by ---).
type SubDoc struct {
	Slices     map[string]*Slice // slices in this sub-document
	SliceOrder []string          // insertion order of slice names
	Comment    string            // comment above the document, without '#'
}

// Document is the root node of an Emlang YAML document.
// A document may contain multiple YAML documents (separated by ---),
// each with a slices: key. Slices from all documents are merged.
type Document struct {
	Slices    map[string]*Slice // merged (backwards compat)
	SubDocs   []*SubDoc         // per YAML document
	RawSource []byte            // raw YAML input
}

// Slice represents a named slice (sequence of elements).
// Supports both direct form (just elements) and extended form (steps + tests).
type Slice struct {
	Name      string
	Elements  []*Element       // slice steps
	Tests     map[string]*Test // attached tests (extended form only)
	TestOrder []string         // insertion order of test names
	Comment   string           // comment above the slice, without '#'
}

// Test represents a test with Given-When-Then structure.
type Test struct {
	Name     string
	Given    []*Element // pre-conditions (events, views)
	When     []*Element // commands being tested
	Then     []*Element // expected results (events, views, exceptions)
	HasGiven bool       // true if given key was present in source
	HasWhen  bool       // true if when key was present in source
	HasThen  bool       // true if then key was present in source
	Comment  string     // comment above the test, without '#'
}

// ElementType represents the type of an element.
type ElementType int

const (
	ElementTrigger ElementType = iota
	ElementCommand
	ElementEvent
	ElementException
	ElementView
)

func (t ElementType) String() string {
	switch t {
	case ElementTrigger:
		return "trigger"
	case ElementCommand:
		return "command"
	case ElementEvent:
		return "event"
	case ElementException:
		return "exception"
	case ElementView:
		return "view"
	default:
		return "unknown"
	}
}

// elementKeys maps YAML keys to element types.
var elementKeys = map[string]ElementType{
	"t":         ElementTrigger,
	"trg":       ElementTrigger,
	"trigger":   ElementTrigger,
	"c":         ElementCommand,
	"cmd":       ElementCommand,
	"command":   ElementCommand,
	"e":         ElementEvent,
	"evt":       ElementEvent,
	"event":     ElementEvent,
	"x":         ElementException,
	"err":       ElementException,
	"exception": ElementException,
	"v":         ElementView,
	"view":      ElementView,
}

// ElementTypeFromKey returns the element type for a YAML key (any alias).
func ElementTypeFromKey(key string) (ElementType, bool) {
	t, ok := elementKeys[key]
	return t, ok
}

// PropEntry is a key-value pair that preserves insertion order.
type PropEntry struct {
	Key   string
	Value interface{}
}

// Element represents an element in a slice or test.
type Element struct {
	Type     ElementType
	Name     string      // element name (may include Swimlane/Name)
	Swimlane string      // extracted swimlane if present
	Props    []PropEntry // free-form properties (ordered)
	Line     int         // source line (1-based)
	Column   int         // source column (1-based)
}

// ParseSwimlane extracts swimlane from element name if present.
// Format: "Swimlane/ElementName" -> swimlane="Swimlane", name="ElementName"
func (e *Element) ParseSwimlane() {
	for i, c := range e.Name {
		if c == '/' {
			e.Swimlane = e.Name[:i]
			e.Name = e.Name[i+1:]
			return
		}
	}
}
