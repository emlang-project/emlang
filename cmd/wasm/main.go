//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/emlang-project/emlang/internal/diagram"
	"github.com/emlang-project/emlang/internal/formatter"
	"github.com/emlang-project/emlang/internal/linter"
	"github.com/emlang-project/emlang/internal/parser"
)

func render(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "missing source argument"}
	}

	src := args[0].String()

	doc, err := parser.Parse(strings.NewReader(src))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	gen := diagram.New()

	// Optional CSS overrides from second argument (JS object)
	if len(args) >= 2 && args[1].Type() == js.TypeObject {
		css := make(map[string]string)
		keys := js.Global().Get("Object").Call("keys", args[1])
		for i := 0; i < keys.Length(); i++ {
			k := keys.Index(i).String()
			css[k] = args[1].Get(k).String()
		}
		gen.CSSOverrides = css
	}

	html, err := gen.Generate(doc)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	lint := linter.New()
	issues := lint.Lint(doc)
	var lintItems []interface{}
	for _, issue := range issues {
		lintItems = append(lintItems, map[string]interface{}{
			"rule":     issue.Rule,
			"message":  issue.Message,
			"line":     issue.Line,
			"column":   issue.Column,
			"severity": issue.Severity.String(),
		})
	}

	return map[string]interface{}{"html": string(html), "lint": lintItems}
}

func format(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "missing source argument"}
	}

	src := args[0].String()

	doc, err := parser.Parse(strings.NewReader(src))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	keyStyle := "long"
	if len(args) >= 2 && args[1].Type() == js.TypeString {
		keyStyle = args[1].String()
	}

	out := formatter.Format(doc, formatter.Options{KeyStyle: keyStyle})

	return map[string]interface{}{"yaml": string(out)}
}

func main() {
	js.Global().Set("emlangRender", js.FuncOf(render))
	js.Global().Set("emlangFormat", js.FuncOf(format))

	// Signal ready
	if cb := js.Global().Get("onEmlangReady"); cb.Truthy() {
		cb.Invoke()
	}

	select {}
}
