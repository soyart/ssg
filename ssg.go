package ssg

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const (
	MarkdownExtensions = parser.CommonExtensions // | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	HtmlFlags          = html.CommonFlags        // | html.HrefTargetBlank

	HeaderDefault = `
<!DOCTYPE html>
<html lang="en">
	<head>
	  <meta charset="UTF-8">
	</head>
	<body>
`

	FooterDefault = `
	</body>
</html>
`
)

func New(src, dst string) Ssg {
	return Ssg{
		src:   src,
		dist:  dst,
		htmls: make(setStr),

		headers: perDir{
			valueDefault: bytes.NewBufferString(HeaderDefault),
			values:       make(map[string]*bytes.Buffer),
		},

		footers: perDir{
			valueDefault: bytes.NewBufferString(FooterDefault),
			values:       make(map[string]*bytes.Buffer),
		},
	}
}

func ToHtml(md []byte) []byte {
	node := markdown.Parse(md, parser.NewWithExtensions(MarkdownExtensions))
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: HtmlFlags,
	})

	return markdown.Render(node, renderer)
}

type Ssg struct {
	headers   perDir
	footers   perDir
	htmls     setStr // Used to ignore md files with identical names, as per the original ssg
	walkError error
	src       string
	dist      string
	writes    []write
}

type write struct {
	target string
	data   []byte
}

type perDir struct {
	valueDefault *bytes.Buffer
	values       map[string]*bytes.Buffer
}

type writeError struct {
	err    error
	target string
}

func (w writeError) Error() string {
	return fmt.Errorf("WriteError(%s): %w", w.target, w.err).Error()
}

// Walk walks the src directory, and converts Markdown into HTML,
// which gets stored in s.writes.
func (s *Ssg) Walk() error {
	err := filepath.WalkDir(s.src, s.walk)
	if err == nil {
		err = s.walkError
	}

	return err
}

// WriteOut concurrently writes out s.writes to their target locations
func (s *Ssg) WriteOut() error {
	if len(s.writes) == 0 {
		return nil
	}

	_, err := os.Stat(s.dist)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.dist, os.ModePerm)
	}

	if err != nil {
		return err
	}

	var wErrors []error // write errors
	errChan := make(chan error)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for err := range errChan {
			wErrors = append(wErrors, err)
		}
	}(wg)

	writeOut(s.writes, errChan)

	wg.Wait()
	if len(wErrors) != 0 {
		return fmt.Errorf("error writing out: %w", errors.Join(wErrors...))
	}

	return nil
}

func (s *Ssg) Generate(baseUrl string) error {
	err := s.Walk()
	if err != nil {
		return err
	}

	pront := func(l int) {
		fmt.Printf("[ssg-go] wrote %d file(s) to %s\n", l, s.dist)
	}

	err = s.WriteOut()
	if err != nil {
		return err
	}

	if baseUrl == "" {
		pront(len(s.writes))
		return nil
	}

	sitemap, err := s.Sitemap(baseUrl, time.Now())
	if err != nil {
		return err
	}

	err = os.WriteFile(s.dist+"/sitemap.xml", []byte(sitemap), os.ModePerm)
	if err != nil {
		return err
	}

	pront(len(s.writes) + 1)

	return nil
}

func (s *Ssg) walk(path string, d fs.DirEntry, e error) error {
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
		data, err := os.ReadFile(path)
		if err != nil {
			s.walkError = err
			return fs.SkipAll
		}

		dir := filepath.Dir(path)
		s.headers.values[dir] = bytes.NewBuffer(data)

		return nil

	case "_footer.html":
		data, err := os.ReadFile(path)
		if err != nil {
			s.walkError = err
			return fs.SkipAll
		}

		dir := filepath.Dir(path)
		s.footers.values[dir] = bytes.NewBuffer(data)

		return nil
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		// Do nothing here

	case ".md":
		html := strings.TrimSuffix(path, ".md")
		html += ".html"

		// Check if there's a competing HTML file
		if s.htmls.contains(html) {
			return nil
		}

	default:
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		s.walkError = err
		return filepath.SkipAll
	}

	target, err := s.mirrorPathDist(path, ext)
	if err != nil {
		s.walkError = err
		return filepath.SkipAll
	}

	// Copy HTML files as they are
	if ext == ".html" {
		s.writes = append(s.writes, write{
			target: target,
			data:   data,
		})

		if s.htmls.insert(path) {
			return fmt.Errorf("duplicate html file %s", path)
		}

		return nil
	}

	body := ToHtml(data)
	header := s.headers.valueDefault
	footer := s.footers.valueDefault

	max := 0
	for p, h := range s.headers.values {
		if !strings.HasPrefix(path, p) {
			continue
		}

		if len(p) < max {
			continue
		}

		header, max = h, len(p)
	}

	max = 0
	for p, f := range s.footers.values {
		if !strings.HasPrefix(path, p) {
			continue
		}

		if len(p) < max {
			continue
		}

		footer, max = f, len(p)
	}

	s.writes = append(s.writes, write{
		target: target,
		data:   []byte(header.String() + string(body) + footer.String()),
	})

	return nil
}

// mirrorPathDist mirrors the target HTML file path under s.src to under s.dist
//
// i.e. if s.src="foo/src" and s.dist="foo/dist",
// and p="foo/src/bar/baz.md" ext=".md",
// then the return value will be foo/dist/bar/baz.html
func (s *Ssg) mirrorPathDist(p string, ext string) (string, error) {
	p = strings.TrimSuffix(p, ext)
	p += ".html"

	p, err := filepath.Rel(s.src, p)
	if err != nil {
		return "", err
	}

	return filepath.Join(s.dist, p), nil
}

func writeOut(writes []write, errs chan<- error) {
	wg := sync.WaitGroup{}
	guard := make(chan struct{}, 20)

	for i := range writes {
		wg.Add(1)
		guard <- struct{}{}

		go func(w *write, wg *sync.WaitGroup) {
			defer func() {
				wg.Done()
				fmt.Println(w.target)
			}()

			<-guard

			d := filepath.Dir(w.target)
			err := os.MkdirAll(d, os.ModePerm)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}

				return
			}

			f, err := os.OpenFile(w.target, os.O_CREATE|os.O_WRONLY, os.ModePerm)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}

				return
			}

			defer f.Close()

			_, err = f.Write(w.data)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}

				return
			}
		}(&writes[i], &wg)
	}

	wg.Wait()

	close(errs)
}

func (s *Ssg) Sitemap(baseUrl string, date time.Time) (string, error) {
	dateStr := date.Format(time.DateOnly)

	sm := new(strings.Builder)
	sm.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9
http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"
xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)

	for i := range s.writes {
		w := &s.writes[i]

		target, err := filepath.Rel(s.dist, w.target)
		if err != nil {
			return sm.String(), err
		}

		sm.WriteString("<url><loc><")
		sm.WriteString(baseUrl + "/")

		/* There're 2 possibilities for this
		1. First is when the HTML is some/path/index.html
		<url><loc>https://example.com/some/path</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>

		2. Then there is when the HTML is some/path/foo.html
		<url><loc>https://example.com/some/path/page.html</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>
		*/

		switch filepath.Base(target) {
		case "index.html":
			sm.WriteString(filepath.Dir(target) + "/")

		default:
			sm.WriteString(target)
		}

		sm.WriteString("><lastmod>")
		sm.WriteString(dateStr)
		sm.WriteString("</lastmod><priority>1.0</priority></url>\n")
	}

	sm.WriteString("</urlset>")

	return sm.String(), nil
}
