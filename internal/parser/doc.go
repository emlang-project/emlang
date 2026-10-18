// Package parser reads Emlang YAML into the ast data model.
//
// Parse decodes each YAML document of the input into a yaml.v3 node tree,
// then walks it. Walking nodes rather than decoding into structs keeps the
// source order of keys and the line and column of each element, which
// error messages and lint issues report.
//
// The comment block above a document, a slice or a test is kept in the
// Comment field of its node, without the leading '#'. yaml.v3 attaches a
// document's comment to the slices key or, when a blank line follows it,
// to the head of the document node for the first document and to the foot
// of the previous document node for the next ones. All three are read. Line comments, foot comments
// and comments on elements are not kept.
//
// The parser enforces the structure of the specification: known keys
// only, one type key per element, the element types allowed in each test
// section, and non-empty names. It returns the first error it finds.
package parser
