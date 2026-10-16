// Package ast holds the data model of an Emlang document, as built by the
// parser and read by the linter, the formatter and the diagram generator.
//
// A file may hold several YAML documents separated by ---. Document keeps
// one SubDoc per YAML document, in source order, and also merges the
// slices of all of them in Slices. Slices, tests, elements and props keep
// their source order, through the *Order fields and ordered slices, because
// the diagram and the formatter show them as written.
//
// Document also keeps the raw source. The formatter rewrites that source
// rather than the model, and the diagram generator derives its HTML ids
// from it.
//
// ElementTypeFromKey maps every YAML key of an element (t, trg, trigger,
// and so on) to its type, so the parser and the formatter accept the same
// aliases.
package ast
