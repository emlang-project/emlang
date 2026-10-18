package diagram

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// markdown renders comments with the GitHub Flavored Markdown extensions
// (tables, strikethrough, autolinks, task lists). It runs in goldmark's
// default safe mode: raw HTML is dropped and dangerous link URLs are
// emptied, which makes its output safe to insert as template.HTML.
var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(
		parser.WithASTTransformers(util.Prioritized(newTabLinks{}, 0)),
	),
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

// newTabLinks makes every link open in a new tab.
type newTabLinks struct{}

func (newTabLinks) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.Link, *ast.AutoLink:
			n.SetAttributeString("target", []byte("_blank"))
			n.SetAttributeString("rel", []byte("noopener noreferrer"))
		}
		return ast.WalkContinue, nil
	})
}

// renderComment converts a Markdown comment to HTML.
func renderComment(src string) (template.HTML, error) {
	if src == "" {
		return "", nil
	}
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
