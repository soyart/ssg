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

func New(baseUrl, src, dst string) Ssg {
	return Ssg{
		baseUrl: baseUrl,
		src:     src,
		dist:    dst,
		htmls:   make(setStr),

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
	baseUrl   string
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

func (s *Ssg) pront(l int) {
	fmt.Printf("[ssg-go] wrote %d file(s) to %s\n", l, s.dist)
}

func (s *Ssg) Generate() error {
	stat, err := os.Stat(s.src)
	if err != nil {
		return err
	}

	err = s.cache()
	if err != nil {
		return err
	}

	err = s.writeOut()
	if err != nil {
		return err
	}

	if s.baseUrl == "" {
		s.pront(len(s.writes))
		return nil
	}

	sitemap, err := s.Sitemap(s.baseUrl, stat.ModTime())
	if err != nil {
		return err
	}

	err = os.WriteFile(s.dist+"/sitemap.xml", []byte(sitemap), os.ModePerm)
	if err != nil {
		return err
	}

	s.pront(len(s.writes) + 1)

	return nil
}

func Generate(sites ...Ssg) error {
	stats := make(map[string]fs.FileInfo)

	for i := range sites {
		s := &sites[i]
		stat, err := os.Stat(s.src)
		if err != nil {
			return err
		}

		stats[s.src] = stat

		err = s.cache()
		if err != nil {
			return fmt.Errorf("error walking in %s: %w", s.src, err)
		}
	}

	for i := range sites {
		s := &sites[i]
		stat, ok := stats[s.src]
		if !ok {
			return fmt.Errorf("ssg-go bug: unexpected missing stat for directory %s (baseUrl='%s')", s.src, s.baseUrl)
		}

		err := s.writeOut()
		if err != nil {
			return fmt.Errorf("error writing out to %s: %w", s.dist, err)
		}

		if s.baseUrl == "" {
			s.pront(len(s.writes))
			return nil
		}

		sitemap, err := s.Sitemap(s.baseUrl, stat.ModTime())
		if err != nil {
			return err
		}

		err = os.WriteFile(s.dist+"/sitemap.xml", []byte(sitemap), os.ModePerm)
		if err != nil {
			return err
		}

		s.pront(len(s.writes) + 1)
	}

	return nil
}

// cache walks the src directory, and converts Markdown into HTML,
// which gets stored in-memory in s.writes.
func (s *Ssg) cache() error {
	err := filepath.WalkDir(s.src, s.walk)
	if err == nil {
		err = s.walkError
	}

	return err
}

// writeOut concurrently writes out s.writes to their target locations
func (s *Ssg) writeOut() error {
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

	wg, errs := new(sync.WaitGroup), make(chan error)
	wg.Add(1)
	go func() {
		err = collectErrors(errs, wg)
	}()

	writeOut(s.writes, errs)

	wg.Wait()
	return err
}

func collectErrors(ch <-chan error, wg *sync.WaitGroup) error {
	defer wg.Done()
	var errs []error
	for err := range ch {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (s *Ssg) walk(path string, d fs.DirEntry, e error) error {
	if e != nil {
		return e
	}

	base := filepath.Base(path)
	isDot := strings.HasPrefix(base, ".")

	switch {
	// Skip hidden folders
	case isDot && d.IsDir():
		return fs.SkipDir

	// Ignore hidden files and dir
	case isDot, d.IsDir():
		return nil
	}

	// Collect cascading headers
	switch base {
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

	data, err := os.ReadFile(path)
	if err != nil {
		s.walkError = err
		return filepath.SkipAll
	}

	ext := filepath.Ext(path)

	switch ext {
	// Check if there's a competing HTML file
	case ".md":
		html := strings.TrimSuffix(path, ".md")
		html += ".html"

		if s.htmls.contains(html) {
			return nil
		}

	// Remember the HTML file, so we can ignore the competing Markdown
	case ".html":
		if s.htmls.insert(path) {
			return fmt.Errorf("duplicate html file %s", path)
		}

		fallthrough

	// Copy files as they are
	default:
		target, err := s.mirrorPathDist(path, ext, ext)
		if err != nil {
			s.walkError = err
			return filepath.SkipAll
		}

		s.writes = append(s.writes, write{
			target: target,
			data:   data,
		})

		return nil
	}

	target, err := s.mirrorPathDist(path, ext, ".html")
	if err != nil {
		s.walkError = err
		return filepath.SkipAll
	}

	body := ToHtml(data)
	header := s.headers.valueDefault
	footer := s.footers.valueDefault

	max := 0
	for p, h := range s.headers.values {
		if max > len(p) {
			continue
		}
		if !strings.HasPrefix(path, p) {
			continue
		}

		header, max = h, len(p)
	}

	max = 0
	for p, f := range s.footers.values {
		if max > len(p) {
			continue
		}
		if !strings.HasPrefix(path, p) {
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
// and p="foo/src/bar/baz.md" ext=".md" newExt=".html",
// then the return value will be foo/dist/bar/baz.html
func (s *Ssg) mirrorPathDist(p, ext, newExt string) (string, error) {
	if ext != newExt {
		p = strings.TrimSuffix(p, ext)
		p += newExt
	}

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

		2. Then there is when the HTML is some/path/page.html
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
