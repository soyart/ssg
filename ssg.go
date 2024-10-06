package ssg

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const (
	MarkdownExtensions = parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	HtmlFlags          = html.CommonFlags | html.HrefTargetBlank
)

func ToHtml(md []byte) []byte {
	node := markdown.Parse(md, parser.NewWithExtensions(MarkdownExtensions))
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: HtmlFlags,
	})

	return markdown.Render(node, renderer)
}
