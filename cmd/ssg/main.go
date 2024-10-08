package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/soyart/ssg"
)

const (
	header = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
</head>
<body>`

	footer = `
</body>
</html>`
)

func main() {
	site := site{
		src:    "artnoi.com/src",
		dist:   "artnoi.com/dist",
		header: header,
		footer: footer,
	}

	err := site.gen()
	if err != nil {
		panic(err)
	}
}

type site struct {
	exitError error
	src       string
	dist      string
	header    string
	footer    string
}

func (s *site) gen() error {
	err := filepath.WalkDir(s.src, s.walk)
	if err != nil {
		err = s.exitError
	}

	return err
}

func (s *site) walk(path string, d fs.DirEntry, e error) error {
	if d.IsDir() && strings.HasPrefix(path, ".git") {
		return fs.SkipDir
	}

	if e != nil {
		if d == nil { // The dir is not a directory
			return nil
		}

		return e
	}

	if d.IsDir() {
		return nil
	}

	switch filepath.Base(path) {
	case "_header.html":
		fmt.Println("found header!! xxxxxxxxxxxxx")
		data, err := os.ReadFile(path)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		s.header = string(data)

	case "_footer.html":
		fmt.Println("found footer!! xxxxxxxxxxxxx")
		data, err := os.ReadFile(path)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		s.footer = string(data)
	}

	if filepath.Ext(path) != ".md" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		s.exitError = err
		return filepath.SkipAll
	}

	body := ssg.ToHtml(data)
	html := s.header + string(body) + s.footer

	fmt.Println()
	fmt.Println(string(html))
	fmt.Println()

	return nil
}
