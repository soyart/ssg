package soyweb

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyart/ssg"
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

type (
	MinifyFn func(data []byte) ([]byte, error)
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
	fn, err := ExtToFn(filepath.Ext(path))
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

	mediaType, err := ExtToMediaType(filepath.Ext(path))
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

func ExtToMediaType(ext string) (string, error) {
	switch ext {
	case ".html":
		return mediaTypeHtml, nil
	case ".css":
		return mediaTypeCss, nil
	case ".json":
		return mediaTypeJson, nil
	}

	return "", fmt.Errorf("'%s': %w", ext, ErrNotSupported)
}

func ExtToFn(ext string) (func([]byte) ([]byte, error), error) {
	switch ext {
	case ".html":
		return MinifyHtml, nil
	case ".css":
		return MinifyCss, nil
	case ".json":
		return MinifyJson, nil
	}

	return nil, fmt.Errorf("'%s': %w", ext, ErrNotSupported)
}

func pipelineMinify(m map[string]MinifyFn) ssg.PipelineFn {
	if len(m) == 0 {
		return nil
	}

	return func(path string, data []byte) ([]byte, error) {
		ext := filepath.Ext(path)
		f, ok := m[ext]
		if !ok {
			return data, nil
		}

		b, err := f(data)
		if err != nil {
			return nil, fmt.Errorf("error from minifier for '%s'", ext)
		}

		return b, nil
	}
}
