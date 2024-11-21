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

func MinifyHtml(htmlDoc []byte) ([]byte, error) {
	minified := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeHtml, minified, bytes.NewBuffer(htmlDoc))
	if err != nil {
		return nil, err
	}

	return minified.Bytes(), nil
}

func MinifyCss(cssDoc []byte) ([]byte, error) {
	minified := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeCss, minified, bytes.NewBuffer(cssDoc))
	if err != nil {
		return nil, err
	}

	return minified.Bytes(), nil
}

func SsgPipeline(path string, cssDoc []byte) ([]byte, error) {
	switch filepath.Ext(path) {
	case ".css":
		return MinifyCss(cssDoc)
	}

	return cssDoc, nil
}
