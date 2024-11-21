package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/soyart/ssg"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

func main() {
	const mediaType = "text/html"

	header, err := os.ReadFile("../testdata/johndoe.com/src/_header.html")
	if err != nil {
		panic(err)
	}
	footer, err := os.ReadFile("../testdata/johndoe.com/src/_footer.html")
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(header)

	_, err = buf.Write(ssg.ToHtml([]byte("Hello, world!")))
	if err != nil {
		panic(err)
	}

	_, err = buf.Write(footer)
	if err != nil {
		panic(err)
	}

	m := minify.New()
	m.AddFunc(mediaType, html.Minify)

	minified := bytes.NewBuffer(nil)
	err = m.Minify(mediaType, minified, buf)
	if err != nil {
		panic(err)
	}

	fmt.Println(minified.String())
}
