package minifier

import (
	"bytes"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

const mediaType = "text/html"

var m = minify.New()

func init() {
	m.AddFunc(mediaType, html.Minify)
}

// Minify the whole HTML output
func SsgPipelineMinifyHtml(htmlDoc []byte) ([]byte, error) {
	minified := bytes.NewBuffer(nil)
	err := m.Minify(mediaType, minified, bytes.NewBuffer(htmlDoc))
	if err != nil {
		return nil, err
	}

	return minified.Bytes(), nil
}
