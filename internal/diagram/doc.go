// Package diagram renders an Emlang document as an HTML fragment.
//
// Each YAML document becomes a CSS grid. Every element of a slice takes one
// column. Rows hold, from top to bottom, the slice names, the triggers (one
// row per swimlane), the commands and views, the events and exceptions (one
// row per swimlane), then the tests, each under its slice.
//
// A column is never narrower than the min-content width of its items. With
// plain auto columns, a diagram wider than the page squeezes each item down
// to its min-width, and the props it holds overflow the item.
//
// The HTML and CSS live in the templates directory, embedded in the
// binary, and are filled with html/template. The CSS uses custom
// properties, which Generator.CSSOverrides can set.
// Deprecated property names are translated to their current names before
// rendering, so a config written for an earlier v1 release keeps working.
//
// The comments above a document, a slice or a test are rendered from
// GitHub Flavored Markdown with goldmark. goldmark runs in its default safe mode: it drops
// raw HTML and empties dangerous link URLs. This is what makes it correct
// to insert its output as template.HTML, so the mode must stay safe.
//
// The HTML id of each document is derived from a hash of the source, so
// several diagrams on one page don't share ids or styles.
package diagram
