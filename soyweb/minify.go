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

	"github.com/soyart/ssg/ssg-go"
)

type (
	MinifyFn func(data []byte) ([]byte, error)
)

const (
	MediaTypeHtml = "text/html"
	MediaTypeCss  = "style/css"
	MediaTypeJs   = "text/javascript"
	MediaTypeJson = "application/json"
)

var m = minify.New()

func init() {
	m.Add(MediaTypeHtml, &html.Minifier{
		// Default values are shown to be more conspicuous
		KeepComments:            false,
		KeepConditionalComments: false,
		KeepSpecialComments:     false,
		KeepDefaultAttrVals:     false,
		KeepDocumentTags:        true,
		KeepEndTags:             true,
		KeepQuotes:              false,
		KeepWhitespace:          false,
		TemplateDelims:          [2]string{},
	})
	m.AddFunc(MediaTypeCss, css.Minify)
	m.AddFunc(MediaTypeJs, js.Minify)
	m.AddFunc(MediaTypeJson, json.Minify)
}

func MinifyHtml(og []byte) ([]byte, error) { return minifyMedia(og, MediaTypeHtml) }
func MinifyCss(og []byte) ([]byte, error)  { return minifyMedia(og, MediaTypeCss) }
func MinifyJs(og []byte) ([]byte, error)   { return minifyMedia(og, MediaTypeJs) }
func MinifyJson(og []byte) ([]byte, error) { return minifyMedia(og, MediaTypeJson) }

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
		return MediaTypeHtml, nil
	case ".css":
		return MediaTypeCss, nil
	case ".js":
		return MediaTypeJs, nil
	case ".json":
		return MediaTypeJson, nil
	}

	return "", fmt.Errorf("'%s': %w", ext, ErrWebFormatNotSupported)
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

	return nil, fmt.Errorf("'%s': %w", ext, ErrWebFormatNotSupported)
}

func minifyMedia(original []byte, mediaType string) ([]byte, error) {
	min := bytes.NewBuffer(nil)
	err := m.Minify(mediaType, min, bytes.NewBuffer(original))
	if err != nil {
		return nil, err
	}
	return min.Bytes(), nil
}

func HookMinifyDefault(mediaTypes ssg.Set) ssg.Hook {
	m := make(map[string]MinifyFn)
	if mediaTypes.Contains(MediaTypeHtml) {
		m[".html"] = MinifyHtml
	}
	if mediaTypes.Contains(MediaTypeJs) {
		m[".js"] = MinifyJs
	}
	if mediaTypes.Contains(MediaTypeJson) {
		m[".json"] = MinifyJson
	}
	if mediaTypes.Contains(MediaTypeCss) {
		m["css"] = MinifyCss
	}
	return HookMinify(m)
}

func HookMinify(m map[string]MinifyFn) ssg.Hook {
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
