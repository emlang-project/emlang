// Package parser reads Emlang YAML into the ast data model.
//
// Parse decodes each YAML document of the input into a yaml.v3 node tree,
// then walks it. Walking nodes rather than decoding into structs keeps the
// source order of keys and the line and column of each element, which
// error messages and lint issues report.
//
// The parser enforces the structure of the specification: known keys
// only, one type key per element, the element types allowed in each test
// section, and non-empty names. It returns the first error it finds.
package parser
