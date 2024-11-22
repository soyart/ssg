package soyweb

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/json"
)

const (
	mediaTypeHtml = "text/html"
	mediaTypeCss  = "style/css"
	mediaTypeJson = "application/json"
)

var m = minify.New()

func init() {
	m.AddFunc(mediaTypeHtml, html.Minify)
	m.AddFunc(mediaTypeCss, css.Minify)
	m.AddFunc(mediaTypeJson, json.Minify)
}

func MinifyHtml(htmlDoc []byte) ([]byte, error) {
	min := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeHtml, min, bytes.NewBuffer(htmlDoc))
	if err != nil {
		return nil, err
	}

	return min.Bytes(), nil
}

func MinifyCss(cssDoc []byte) ([]byte, error) {
	min := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeCss, min, bytes.NewBuffer(cssDoc))
	if err != nil {
		return nil, err
	}

	return min.Bytes(), nil
}

func MinifyJson(jsonDoc []byte) ([]byte, error) {
	min := bytes.NewBuffer(nil)
	err := m.Minify(mediaTypeJson, min, bytes.NewBuffer(jsonDoc))
	if err != nil {
		return nil, err
	}

	return min.Bytes(), nil
}

func MinifyAll(path string, data []byte) ([]byte, error) {
	fn, err := mapFn(filepath.Ext(path))
	if err != nil {
		return data, nil
	}

	out, err := fn(data)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func MinifyFile(path string) ([]byte, error) {
	fn, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer fn.Close()

	mediaType, err := mapType(filepath.Ext(path))
	if err != nil {
		return nil, err
	}

	min := bytes.NewBuffer(nil)
	err = m.Minify(mediaType, min, fn)
	if err != nil {
		return nil, err
	}

	return min.Bytes(), nil
}

func mapType(ext string) (string, error) {
	switch ext {
	case ".html":
		return mediaTypeHtml, nil
	case ".css":
		return mediaTypeCss, nil
	case ".json":
		return mediaTypeJson, nil
	}

	return "", fmt.Errorf("unknown media extension '%s'", ext)
}

func mapFn(ext string) (func([]byte) ([]byte, error), error) {
	switch ext {
	case ".html":
		return MinifyHtml, nil
	case ".css":
		return MinifyCss, nil
	case ".json":
		return MinifyJson, nil
	}

	return nil, fmt.Errorf("unknown media extension '%s'", ext)
}
