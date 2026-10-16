// Package linter reports issues in a parsed Emlang document.
//
// The parser rejects what the specification forbids. The linter checks a
// valid document for patterns that are likely mistakes, such as a command
// with no event after it, and reports each one as an Issue with a rule
// name, a position and a severity. Rules listed in IgnoreRules are not
// reported, which is how the lint.ignore setting of .emlang.yaml applies.
package linter
