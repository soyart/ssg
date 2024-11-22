package main

import (
	"bytes"
	"fmt"
	"os"
	"syscall"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

const mediaType = "text/html"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stdout, "expecting 1 argument")
		syscall.Exit(1)
	}

	filename := os.Args[1]
	doc, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to read input file '%s': %v\n", filename, err)
		syscall.Exit(2)
	}

	m := minify.New()
	m.AddFunc(mediaType, html.Minify)

	minified := bytes.NewBuffer(nil)
	err = m.Minify(mediaType, minified, bytes.NewBuffer(doc))
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", minified.Bytes())
}
