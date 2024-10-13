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
	MarkdownExtensions = parser.CommonExtensions |
		parser.Mmark |
		parser.AutoHeadingIDs

	HtmlFlags = html.CommonFlags

	headerDefault = `
<!DOCTYPE html>
<html lang="en">
	<head>
	  <meta charset="UTF-8">
	</head>
	<body>
`

	footerDefault = `
	</body>
</html>
`
)

func New(baseUrl, src, dst string) Ssg {
	return Ssg{
		baseUrl:   baseUrl,
		src:       src,
		dst:       dst,
		preferred: make(setStr),

		headers: perDir{
			valueDefault: bytes.NewBufferString(headerDefault),
			values:       make(map[string]*bytes.Buffer),
		},

		footers: perDir{
			valueDefault: bytes.NewBufferString(footerDefault),
			values:       make(map[string]*bytes.Buffer),
		},
	}
}

func ToHtml(md []byte) []byte {
	root := markdown.Parse(md, parser.NewWithExtensions(MarkdownExtensions))
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: HtmlFlags,
	})

	return markdown.Render(root, renderer)
}

type Ssg struct {
	baseUrl   string
	headers   perDir
	footers   perDir
	preferred setStr // Used to prefer html and ignore md files with identical names, as with the original ssg
	walkError error
	src       string
	dst       string
	dist      []write
}

type write struct {
	target string
	data   []byte
}

type writeError struct {
	err    error
	target string
}

func (w writeError) Error() string {
	return fmt.Errorf("WriteError(%s): %w", w.target, w.err).Error()
}

func (s *Ssg) pront(l int) {
	fmt.Printf("[ssg-go] wrote %d file(s) to %s\n", l, s.dst)
}

func (s *Ssg) Generate() error {
	stat, err := os.Stat(s.src)
	if err != nil {
		return err
	}

	dist, err := s.Build()
	if err != nil {
		return err
	}

	err = s.WriteOut()
	if err != nil {
		return err
	}

	if s.baseUrl == "" {
		s.pront(len(dist))
		return nil
	}

	sitemap, err := Sitemap(s.dst, s.baseUrl, stat.ModTime(), s.dist)
	if err != nil {
		return err
	}

	err = os.WriteFile(s.dst+"/sitemap.xml", []byte(sitemap), os.ModePerm)
	if err != nil {
		return err
	}

	s.pront(len(dist) + 1)

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

		_, err = s.Build()
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

		err := s.WriteOut()
		if err != nil {
			return fmt.Errorf("error writing out to %s: %w", s.dst, err)
		}

		if s.baseUrl == "" {
			s.pront(len(s.dist))
			return nil
		}

		sitemap, err := Sitemap(s.dst, s.baseUrl, stat.ModTime(), s.dist)
		if err != nil {
			return err
		}

		err = os.WriteFile(s.dst+"/sitemap.xml", []byte(sitemap), os.ModePerm)
		if err != nil {
			return err
		}

		s.pront(len(s.dist) + 1)
	}

	return nil
}

// Build walks the src directory, and converts Markdown into HTML,
// returning the results as []write.
//
// Build also caches the result in s for [WriteOut] later.
func (s *Ssg) Build() ([]write, error) {
	err := filepath.WalkDir(s.src, s.scan)
	if err == nil {
		err = s.walkError
	}
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(s.src, s.build)
	if err == nil {
		err = s.walkError
	}
	if err != nil {
		return nil, err
	}

	return s.dist, nil
}

// WriteOut concurrently writes out s.writes to their target locations.
// If targets is empty, WriteOut writes to s.dst
func (s *Ssg) WriteOut(targets ...string) error {
	if len(s.dist) == 0 {
		return fmt.Errorf("nothing to write")
	}

	if len(targets) == 0 {
		return s.WriteOut(s.dst)
	}

	for i := range targets {
		target := targets[i]

		_, err := os.Stat(target)
		if os.IsNotExist(err) {
			err = os.MkdirAll(s.dst, os.ModePerm)
		}
		if err != nil {
			return err
		}

		wg, errs := new(sync.WaitGroup), make(chan error)
		wg.Add(1)
		go func() {
			err = collectErrors(errs, wg)
		}()

		writeOut(s.dist, errs)

		wg.Wait()

		if err != nil {
			return err
		}
	}

	return nil
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

func isIgnored(base string, d fs.DirEntry) (bool, error) {
	isDot := strings.HasPrefix(base, ".")
	isDir := d.IsDir()

	switch {
	// Skip hidden folders
	case isDot && isDir:
		return true, fs.SkipDir

	// Ignore hidden files and dir
	case isDot, isDir:
		return true, nil
	}

	return false, nil
}

// scan scans the source directory for header and footer files,
// and anything required to build a page.
func (s *Ssg) scan(path string, d fs.DirEntry, e error) error {
	if e != nil {
		return e
	}

	base := filepath.Base(path)
	ignore, err := isIgnored(base, d)
	if err != nil {
		return err
	}
	if ignore {
		return nil
	}

	// Collect cascading headers and footers
	switch base {
	case "_header.html":
		data, err := os.ReadFile(path)
		if err != nil {
			s.walkError = err
			return err
		}

		s.headers.add(filepath.Dir(path), bytes.NewBuffer(data))

	case "_footer.html":
		data, err := os.ReadFile(path)
		if err != nil {
			s.walkError = err
			return err
		}

		s.footers.add(filepath.Dir(path), bytes.NewBuffer(data))
	}

	return nil
}

// build finds and converts Markdown files to HTML,
// and assembles it with header and footer.
func (s *Ssg) build(path string, d fs.DirEntry, e error) error {
	if e != nil {
		return e
	}

	base := filepath.Base(path)
	ignore, err := isIgnored(base, d)
	if err != nil {
		return err
	}
	if ignore {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		s.walkError = err
		return err
	}

	ext := filepath.Ext(base)

	switch ext {
	// Check if there's a competing HTML file
	case ".md":
		html := strings.TrimSuffix(path, ".md")
		html += ".html"

		if s.preferred.contains(html) {
			return nil
		}

	// Remember the HTML file, so we can ignore the competing Markdown
	case ".html":
		if s.preferred.insert(path) {
			return fmt.Errorf("duplicate html file %s", path)
		}

		fallthrough

	// Copy files as they are
	default:
		target, err := mirrorPath(s.src, s.dst, path, ext)
		if err != nil {
			return err
		}

		s.dist = append(s.dist, write{
			target: target,
			data:   data,
		})

		return nil
	}

	target, err := mirrorPath(s.src, s.dst, path, ".html")
	if err != nil {
		s.walkError = err
		return err
	}

	body := ToHtml(data)
	header := s.headers.choose(path)
	footer := s.footers.choose(path)

	s.dist = append(s.dist, write{
		target: target,
		data:   []byte(header.String() + string(body) + footer.String()),
	})

	return nil
}

// mirrorPath mirrors the target HTML file path under src to under dist
//
// i.e. if src="foo/src" and dst="foo/dist",
// and path="foo/src/bar/baz.md"  newExt=".html",
// then the return value will be foo/dist/bar/baz.html
func mirrorPath(
	src string,
	dst string,
	path string,
	newExt string, // File's new extension after mirrored
) (
	string,
	error,
) {
	ext := filepath.Ext(path)
	if ext != newExt {
		path = strings.TrimSuffix(path, ext)
		path += newExt
	}

	path, err := filepath.Rel(src, path)
	if err != nil {
		return "", err
	}

	return filepath.Join(dst, path), nil
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

func Sitemap(
	dst string,
	baseUrl string,
	date time.Time,
	writes []write,
) (
	string,
	error,
) {
	dateStr := date.Format(time.DateOnly)

	sm := new(strings.Builder)
	sm.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9
http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"
xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)

	for i := range writes {
		w := &writes[i]

		target, err := filepath.Rel(dst, w.target)
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
