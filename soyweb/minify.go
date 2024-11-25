package soyweb

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"

	"github.com/soyart/ssg"
)

const (
	mediaTypeHtml = "text/html"
	mediaTypeCss  = "style/css"
	mediaTypeJs   = "text/javascript"
	mediaTypeJson = "application/json"
)

type (
	MinifyFn func(data []byte) ([]byte, error)
)

var m = minify.New()

func init() {
	m.AddFunc(mediaTypeHtml, html.Minify)
	m.AddFunc(mediaTypeCss, css.Minify)
	m.AddFunc(mediaTypeJs, js.Minify)
	m.AddFunc(mediaTypeJson, json.Minify)
}

func MinifyHtml(original []byte) ([]byte, error) {
	return minifyFormat(original, mediaTypeHtml)
}

func MinifyCss(original []byte) ([]byte, error) {
	return minifyFormat(original, mediaTypeCss)
}

func MinifyJs(original []byte) ([]byte, error) {
	return minifyFormat(original, mediaTypeJs)
}

func MinifyJson(original []byte) ([]byte, error) {
	return minifyFormat(original, mediaTypeJson)
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
	original, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer original.Close()

	mediaType, err := ExtToMediaType(filepath.Ext(path))
	if err != nil {
		return nil, err
	}

	min := bytes.NewBuffer(nil)
	err = m.Minify(mediaType, min, original)
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
	case ".js":
		return mediaTypeJs, nil
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
	case ".js":
		return MinifyJs, nil
	case ".json":
		return MinifyJson, nil
	}

	return nil, fmt.Errorf("'%s': %w", ext, ErrNotSupported)
}

func minifyFormat(original []byte, format string) ([]byte, error) {
	min := bytes.NewBuffer(nil)
	err := m.Minify(format, min, bytes.NewBuffer(original))
	if err != nil {
		return nil, err
	}

	return min.Bytes(), nil
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
