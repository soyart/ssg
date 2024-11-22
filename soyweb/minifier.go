package soyweb

import (
	"bytes"
	"path/filepath"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/json"
)

const mediaTypeHtml = "text/html"
const mediaTypeCss = "style/css"
const mediaTypeJson = "application/json"

var m = minify.New()

func init() {
	m.AddFunc(mediaTypeHtml, html.Minify)
	m.AddFunc(mediaTypeCss, css.Minify)
	m.AddFunc(mediaTypeJson, json.Minify)
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

func MinifyJson(jsonDoc []byte) ([]byte, error) {
	minified := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeJson, minified, bytes.NewBuffer(jsonDoc))
	if err != nil {
		return nil, err
	}

	return minified.Bytes(), nil
}

func Minify(path string, data []byte) ([]byte, error) {
	switch filepath.Ext(path) {
	case ".html":
		return MinifyHtml(data)
	case ".css":
		return MinifyCss(data)
	case ".json":
		return MinifyJson(data)
	}

	return data, nil
}
