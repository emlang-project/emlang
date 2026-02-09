package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/emlang-project/emlang/internal/ast"
	"gopkg.in/yaml.v3"
)

// elementPrefixes maps YAML keys to element types.
var elementPrefixes = map[string]ast.ElementType{
	"t":         ast.ElementTrigger,
	"trg":       ast.ElementTrigger,
	"trigger":   ast.ElementTrigger,
	"c":         ast.ElementCommand,
	"cmd":       ast.ElementCommand,
	"command":   ast.ElementCommand,
	"e":         ast.ElementEvent,
	"evt":       ast.ElementEvent,
	"event":     ast.ElementEvent,
	"x":         ast.ElementException,
	"err":       ast.ElementException,
	"exception": ast.ElementException,
	"v":         ast.ElementView,
	"view":      ast.ElementView,
}

// isNullNode returns true if the node represents a YAML null value.
func isNullNode(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null"
}

// Parse parses an Emlang YAML file from the reader.
// Supports multiple YAML documents separated by ---.
func Parse(r io.Reader) (*ast.Document, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))

	doc := &ast.Document{
		Slices:    make(map[string]*ast.Slice),
		RawSource: raw,
	}

	for {
		var root yaml.Node
		err := decoder.Decode(&root)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("yaml parse error: %w", err)
		}

		subDoc := &ast.SubDoc{
			Slices: make(map[string]*ast.Slice),
		}

		if err := parseDocument(&root, doc, subDoc); err != nil {
			return nil, err
		}

		doc.SubDocs = append(doc.SubDocs, subDoc)
	}

	return doc, nil
}

// parseDocument parses a single YAML document node and merges slices into doc.
func parseDocument(root *yaml.Node, doc *ast.Document, subDoc *ast.SubDoc) error {
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}

	docNode := root.Content[0]
	if docNode.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping at root, got %v", docNode.Kind)
	}

	for i := 0; i < len(docNode.Content); i += 2 {
		keyNode := docNode.Content[i]
		valueNode := docNode.Content[i+1]

		switch keyNode.Value {
		case "slices":
			slices, sliceOrder, err := parseSlices(valueNode)
			if err != nil {
				return err
			}
			for _, name := range sliceOrder {
				slice := slices[name]
				doc.Slices[name] = slice
				subDoc.Slices[name] = slice
			}
			subDoc.SliceOrder = sliceOrder

		default:
			return fmt.Errorf("unknown top-level key %q at line %d", keyNode.Value, keyNode.Line)
		}
	}

	return nil
}

// parseSlices parses the slices section.
func parseSlices(node *yaml.Node) (map[string]*ast.Slice, []string, error) {
	slices := make(map[string]*ast.Slice)
	var order []string

	if isNullNode(node) {
		return slices, order, nil
	}

	if node.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("slices must be a mapping at line %d", node.Line)
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		sliceName := keyNode.Value
		slice, err := parseSlice(sliceName, valueNode)
		if err != nil {
			return nil, nil, fmt.Errorf("slice %q: %w", sliceName, err)
		}
		slices[sliceName] = slice
		order = append(order, sliceName)
	}

	return slices, order, nil
}

// parseSlice parses a single slice in direct or extended form.
func parseSlice(name string, node *yaml.Node) (*ast.Slice, error) {
	// Empty slice (null value): placeholder
	if isNullNode(node) {
		return &ast.Slice{Name: name}, nil
	}

	switch node.Kind {
	case yaml.SequenceNode:
		elements, err := parseElementList(node)
		if err != nil {
			return nil, err
		}
		if len(elements) == 0 {
			return nil, fmt.Errorf("slice must have at least one element at line %d", node.Line)
		}
		return &ast.Slice{
			Name:     name,
			Elements: elements,
		}, nil

	case yaml.MappingNode:
		slice := &ast.Slice{
			Name:  name,
			Tests: make(map[string]*ast.Test),
		}

		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			switch keyNode.Value {
			case "steps":
				if isNullNode(valueNode) {
					slice.Elements = []*ast.Element{}
				} else {
					elements, err := parseElementList(valueNode)
					if err != nil {
						return nil, fmt.Errorf("steps: %w", err)
					}
					if len(elements) == 0 {
						return nil, fmt.Errorf("steps must have at least one element at line %d", valueNode.Line)
					}
					slice.Elements = elements
				}

			case "tests":
				tests, err := parseTests(valueNode)
				if err != nil {
					return nil, fmt.Errorf("tests: %w", err)
				}
				slice.Tests = tests

			default:
				return nil, fmt.Errorf("unknown slice key %q at line %d", keyNode.Value, keyNode.Line)
			}
		}

		if slice.Elements == nil {
			return nil, fmt.Errorf("extended slice must have 'steps' at line %d", node.Line)
		}

		return slice, nil

	default:
		return nil, fmt.Errorf("slice must be a sequence or mapping at line %d", node.Line)
	}
}

// parseTests parses tests attached to a slice.
func parseTests(node *yaml.Node) (map[string]*ast.Test, error) {
	tests := make(map[string]*ast.Test)

	if isNullNode(node) {
		return tests, nil
	}

	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("tests must be a mapping at line %d", node.Line)
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		testName := keyNode.Value
		test, err := parseTest(testName, valueNode)
		if err != nil {
			return nil, fmt.Errorf("test %q: %w", testName, err)
		}

		tests[testName] = test
	}

	return tests, nil
}

// parseTest parses a single test definition.
func parseTest(name string, node *yaml.Node) (*ast.Test, error) {
	// A test MAY be empty (null node).
	if isNullNode(node) {
		return &ast.Test{Name: name}, nil
	}

	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("test must be a mapping at line %d", node.Line)
	}

	test := &ast.Test{Name: name}

	allowedGiven := map[ast.ElementType]bool{ast.ElementEvent: true, ast.ElementView: true}
	allowedWhen := map[ast.ElementType]bool{ast.ElementCommand: true}
	allowedThen := map[ast.ElementType]bool{ast.ElementEvent: true, ast.ElementView: true, ast.ElementException: true}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "given":
			test.HasGiven = true
			elems, err := parseTestSection(keyNode.Value, valueNode, allowedGiven)
			if err != nil {
				return nil, err
			}
			test.Given = elems

		case "when":
			test.HasWhen = true
			elems, err := parseTestSection(keyNode.Value, valueNode, allowedWhen)
			if err != nil {
				return nil, err
			}
			test.When = elems

		case "then":
			test.HasThen = true
			elems, err := parseTestSection(keyNode.Value, valueNode, allowedThen)
			if err != nil {
				return nil, err
			}
			test.Then = elems

		default:
			return nil, fmt.Errorf("unknown test key %q at line %d", keyNode.Value, keyNode.Line)
		}
	}

	return test, nil
}

// parseTestSection parses a given/when/then section, validating element types.
func parseTestSection(section string, node *yaml.Node, allowed map[ast.ElementType]bool) ([]*ast.Element, error) {
	if isNullNode(node) {
		return nil, nil
	}
	elements, err := parseElementList(node)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", section, err)
	}
	for _, elem := range elements {
		if !allowed[elem.Type] {
			return nil, fmt.Errorf("%s: %s not allowed at line %d", section, elem.Type, elem.Line)
		}
	}
	return elements, nil
}

// parseElementList parses a sequence of elements.
func parseElementList(node *yaml.Node) ([]*ast.Element, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected sequence at line %d", node.Line)
	}

	var elements []*ast.Element
	for _, itemNode := range node.Content {
		elem, err := parseElement(itemNode)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
	}

	return elements, nil
}

// parseElement parses a single element.
func parseElement(node *yaml.Node) (*ast.Element, error) {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("element must be a mapping at line %d", node.Line)
	}

	elem := &ast.Element{
		Line:   node.Line,
		Column: node.Column,
	}

	var foundType bool
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value

		if key == "props" {
			props, err := parseProps(valueNode)
			if err != nil {
				return nil, fmt.Errorf("props at line %d: %w", valueNode.Line, err)
			}
			elem.Props = props
			continue
		}

		// Check if it's an element type prefix
		if elemType, ok := elementPrefixes[key]; ok {
			if foundType {
				return nil, fmt.Errorf("element has multiple type keys at line %d", node.Line)
			}
			foundType = true
			elem.Type = elemType
			elem.Name = strings.TrimSpace(valueNode.Value)
			if elem.Name == "" {
				return nil, fmt.Errorf("element %s has no name at line %d", elemType, keyNode.Line)
			}
			if strings.HasSuffix(elem.Name, "/") {
				return nil, fmt.Errorf("element name must not end with '/' at line %d", keyNode.Line)
			}
			elem.ParseSwimlane()
			elem.Swimlane = strings.TrimSpace(elem.Swimlane)
			elem.Name = strings.TrimSpace(elem.Name)
			if elem.Swimlane != "" && elem.Name == "" {
				return nil, fmt.Errorf("element %s has empty name after swimlane at line %d", elemType, keyNode.Line)
			}
		} else {
			return nil, fmt.Errorf("unknown key %q at line %d", key, keyNode.Line)
		}
	}

	if !foundType {
		return nil, fmt.Errorf("element missing type at line %d", node.Line)
	}

	return elem, nil
}

// parseProps parses the props field, preserving source order.
func parseProps(node *yaml.Node) ([]ast.PropEntry, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("props must be a mapping at line %d", node.Line)
	}
	props := make([]ast.PropEntry, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		var val interface{}
		if err := valNode.Decode(&val); err != nil {
			return nil, err
		}
		props = append(props, ast.PropEntry{Key: keyNode.Value, Value: val})
	}
	return props, nil
}
