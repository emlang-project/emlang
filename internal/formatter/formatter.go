package formatter

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/emlang-project/emlang/internal/ast"
	"gopkg.in/yaml.v3"
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

// Format renders the source of a parsed document as canonical YAML.
// The document must come from parser.Parse, so its source is valid.
// Only the YAML node tree is rewritten: comments, quoting and value
// styles are kept as written.
func Format(doc *ast.Document, opts Options) ([]byte, error) {
	if opts.KeyStyle == "" {
		opts.KeyStyle = "short"
	}

	f := &formatter{style: opts.KeyStyle}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	dec := yaml.NewDecoder(bytes.NewReader(doc.RawSource))
	for {
		var root yaml.Node
		err := dec.Decode(&root)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("yaml parse error: %w", err)
		}
		if len(root.Content) == 0 {
			continue
		}

		f.formatDocument(root.Content[0])

		if err := enc.Encode(&root); err != nil {
			return nil, err
		}
	}

	if err := enc.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type formatter struct {
	style string
}

func (f *formatter) formatDocument(node *yaml.Node) {
	_, sliceMap := mappingEntry(node, "slices")
	if sliceMap == nil || sliceMap.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i+1 < len(sliceMap.Content); i += 2 {
		sliceMap.Content[i+1] = f.formatSlice(sliceMap.Content[i], sliceMap.Content[i+1])
	}
}

// formatSlice returns the formatted value of a slice.
func (f *formatter) formatSlice(key, node *yaml.Node) *yaml.Node {
	switch node.Kind {
	case yaml.SequenceNode:
		// Direct form: list of elements
		f.formatElementList(node)
		return node

	case yaml.MappingNode:
		_, tests := mappingEntry(node, "tests")
		if tests == nil || tests.Kind != yaml.MappingNode || len(tests.Content) == 0 {
			return f.directForm(key, node)
		}

		// Extended form: steps + tests
		reorderKeys(node, "steps", "tests")
		_, steps := mappingEntry(node, "steps")
		f.formatElementList(steps)
		f.formatTests(tests)
		return node
	}

	return node
}

// directForm replaces an extended slice without tests by its steps.
// The comments of the dropped nodes are moved around the steps.
func (f *formatter) directForm(key, node *yaml.Node) *yaml.Node {
	stepsKey, steps := mappingEntry(node, "steps")
	if steps == nil {
		return node
	}
	f.formatElementList(steps)

	head := []string{
		node.HeadComment, node.LineComment,
		stepsKey.HeadComment, stepsKey.LineComment,
		steps.HeadComment, steps.LineComment,
	}
	foot := []string{steps.FootComment, stepsKey.FootComment}
	if testsKey, tests := mappingEntry(node, "tests"); testsKey != nil {
		foot = append(foot,
			testsKey.HeadComment, testsKey.LineComment, testsKey.FootComment,
			tests.HeadComment, tests.LineComment, tests.FootComment,
		)
	}
	foot = append(foot, node.FootComment)

	steps.HeadComment, steps.LineComment, steps.FootComment = "", "", ""

	if steps.Kind == yaml.SequenceNode && len(steps.Content) > 0 {
		first := steps.Content[0]
		first.HeadComment = joinComments(append(head, first.HeadComment)...)

		last := lastKey(steps)
		if last == nil {
			last = steps.Content[len(steps.Content)-1]
		}
		last.FootComment = joinComments(append([]string{last.FootComment}, foot...)...)
	} else {
		key.FootComment = joinComments(append(append([]string{key.FootComment}, head...), foot...)...)
	}

	return steps
}

func (f *formatter) formatTests(node *yaml.Node) {
	for i := 1; i < len(node.Content); i += 2 {
		test := node.Content[i]
		if test.Kind != yaml.MappingNode {
			continue
		}

		reorderKeys(test, "given", "when", "then")
		for j := 1; j < len(test.Content); j += 2 {
			f.formatElementList(test.Content[j])
		}
	}
}

func (f *formatter) formatElementList(node *yaml.Node) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, elem := range node.Content {
		f.formatElement(elem)
	}
}

func (f *formatter) formatElement(node *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		return
	}

	var key string
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if t, ok := ast.ElementTypeFromKey(k.Value); ok {
			key = typeKey(t, f.style)
			k.Value = key
			v.Value = normalizeName(v.Value)
		}
	}

	reorderKeys(node, key, "props")
}

// normalizeName trims the element name and its swimlane.
func normalizeName(s string) string {
	elem := &ast.Element{Name: strings.TrimSpace(s)}
	elem.ParseSwimlane()

	name := strings.TrimSpace(elem.Name)
	if lane := strings.TrimSpace(elem.Swimlane); lane != "" {
		return lane + "/" + name
	}
	return name
}

// lastKey returns the deepest last mapping key under node, or nil.
// When decoding, yaml.v3 attaches the comments that follow a block to
// this key, and the encoder only writes them back correctly there.
func lastKey(node *yaml.Node) *yaml.Node {
	var key *yaml.Node
	for {
		switch {
		case node.Kind == yaml.SequenceNode && len(node.Content) > 0:
			node = node.Content[len(node.Content)-1]
		case node.Kind == yaml.MappingNode && len(node.Content) > 1:
			key = node.Content[len(node.Content)-2]
			node = node.Content[len(node.Content)-1]
		default:
			return key
		}
	}
}

// mappingEntry returns the key and value nodes of a mapping entry, or nil.
func mappingEntry(m *yaml.Node, name string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == name {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
}

// reorderKeys moves the given keys first, in order, followed by the others.
func reorderKeys(m *yaml.Node, order ...string) {
	content := make([]*yaml.Node, 0, len(m.Content))
	for _, name := range order {
		if k, v := mappingEntry(m, name); k != nil {
			content = append(content, k, v)
		}
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if !slices.Contains(order, m.Content[i].Value) {
			content = append(content, m.Content[i], m.Content[i+1])
		}
	}
	m.Content = content
}

// joinComments joins the non-empty comments with newlines.
func joinComments(comments ...string) string {
	var parts []string
	for _, c := range comments {
		if c != "" {
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, "\n")
}
