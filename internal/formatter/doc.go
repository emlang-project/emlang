// Package formatter rewrites an Emlang document in a canonical form.
//
// Format works on the yaml.v3 node tree of the source, not on the AST: it
// renames element keys, collapses a slice without tests to the direct
// form, orders keys, and trims names, then lets the yaml.v3 encoder write
// the tree back. Everything the AST doesn't model, such as comments,
// quoting, and the style of prop values, is written back as it was.
//
// The document must come from parser.Parse, so the formatter can rely on
// a valid structure and skip validation.
//
// When the steps key of a slice is dropped, its comments move to the
// first and last elements. A comment that ends a block is attached to the
// deepest last key, where yaml.v3 itself puts such comments when decoding:
// the encoder only writes them back correctly there.
package formatter
