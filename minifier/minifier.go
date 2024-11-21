package minifier

import (
	"bytes"
	"path/filepath"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
)

const mediaTypeHtml = "text/html"
const mediaTypeCss = "style/css"

var m = minify.New()

func init() {
	m.AddFunc(mediaTypeHtml, html.Minify)
	m.AddFunc(mediaTypeCss, css.Minify)
}

func SsgPipelineMinifyCss(path string, cssDoc []byte) ([]byte, error) {
	if filepath.Ext(path) != ".css" {
		return cssDoc, nil
	}

	minified := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeCss, minified, bytes.NewBuffer(cssDoc))
	if err != nil {
		return nil, err
	}

	return minified.Bytes(), nil
}

// Minify the whole HTML output
func SsgHookMinifyHtml(htmlDoc []byte) ([]byte, error) {
	minified := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeHtml, minified, bytes.NewBuffer(htmlDoc))
	if err != nil {
		return nil, err
	}

	return minified.Bytes(), nil
}
