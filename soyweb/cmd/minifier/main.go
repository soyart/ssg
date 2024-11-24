package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg/soyweb"
)

type cli struct {
	Target string `arg:"positional"`
	soyweb.NoMinifyFlags
}

// @TODO: cli option for walking dir
func main() {
	c := cli{}
	arg.MustParse(&c)

	ext := filepath.Ext(c.Target)
	if ext == "" {
		panic("unknown media type")
	}

	switch ext {
	case ".html":
		if c.NoMinifyHtml || c.NoMinifyHtmlAll {
			return
		}
	case ".css":
		if c.NoMinifyCss {
			return
		}
	case ".json":
		if c.NoMinifyJson {
			return
		}
	}

	minified, err := soyweb.MinifyFile(c.Target)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", minified)
}
