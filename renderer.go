package ssg

import (
	"fmt"

	"bytes"
	"io"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const title string = ":title "

type Title struct {
	ast.Leaf
	text string
}

// Custom Renderer implements markdown.Renderer interface
type Renderer struct {
	htmlRenderer *html.Renderer
	elems        []string
}

func NewRenderer(base *html.Renderer, elems ...string) *Renderer {
	return &Renderer{
		htmlRenderer: base,
		elems:        elems,
	}
}

func (r *Renderer) RenderHeader(w io.Writer, ast ast.Node) {
	// TODO: implement custom renderer
	r.htmlRenderer.RenderHeader(w, ast)
}

func (r *Renderer) RenderFooter(w io.Writer, ast ast.Node) {
	r.htmlRenderer.RenderFooter(w, ast)
}

func (r *Renderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	return r.htmlRenderer.RenderNode(w, node, entering)
}

func parseTitle(data []byte) (ast.Node, []byte, int) {
	if !bytes.HasPrefix(data, []byte(title)) {
		return nil, nil, 0
	}

	fmt.Printf("Found a title!\n\n")
	i := len(title)

	end := bytes.Index(data[i:], []byte("\n"))
	if end < 0 {
		return nil, data, 0
	}

	end = end + i
	title := string(data[i:end])
	res := &Title{
		text: title,
	}

	return res, nil, end
}

func parserHook(data []byte) (ast.Node, []byte, int) {
	if node, d, n := parseTitle(data); node != nil {
		return node, d, n
	}

	return nil, nil, 0
}

func newMarkdownParser() *parser.Parser {
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	p.Opts.ParserHook = parserHook
	return p
}

func titleRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if title, ok := node.(*Title); ok {
		if entering {
			io.WriteString(w, "<title>")
			io.WriteString(w, title.text)
			io.WriteString(w, "</title>\n")
		}
		return ast.GoToNext, true
	}

	return ast.GoToNext, false
}

func newTitleRender() *html.Renderer {
	opts := html.RendererOptions{
		Flags:          html.CommonFlags,
		RenderNodeHook: titleRenderHook,
	}

	return html.NewRenderer(opts)
}

var mds = `:title This is my title

# Some h1

Some para1

## Some h2

Some para2

Rest of the document.`

func ParserHookExample() {
	md := []byte(mds)

	p := newMarkdownParser()
	doc := p.Parse([]byte(md))

	renderer := newTitleRender()
	html := markdown.Render(doc, renderer)

	fmt.Printf("--- Markdown:\n%s\n\n--- HTML:\n%s\n", md, html)
}
