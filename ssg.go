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
	SsgExtensions = parser.CommonExtensions |
		parser.Mmark |
		parser.AutoHeadingIDs

	HtmlFlags = html.CommonFlags

	headerDefault = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-h1}}</title>
</head>
	<body>
`

	footerDefault = `
	</body>
</html>
`

	keyTitleH1         = "# "      // The first h1 tag is used as document header title
	keyTitleFromTag    = ":title " // The first line starting with :title will be parsed as document header title
	targetFromH1       = "{{from-h1}}"
	targetFromTag      = "{{from-tag}}"
	placeholderFromH1  = "<title>" + targetFromH1 + "</title>"
	placeholderFromTag = "<title>" + targetFromTag + "</title>"
)

func ToHtml(md []byte) []byte {
	root := markdown.Parse(md, parser.NewWithExtensions(SsgExtensions))
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: HtmlFlags,
	})

	return markdown.Render(root, renderer)
}

func Sitemap(
	dst string,
	url string,
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
		sm.WriteString(url)
		sm.WriteRune('/')

		/* There're 2 possibilities for this
		1. First is when the HTML is some/path/index.html
		<url><loc>https://example.com/some/path/</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>

		2. Then there is when the HTML is some/path/page.html
		<url><loc>https://example.com/some/path/page.html</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>
		*/

		switch filepath.Base(target) {
		case "index.html":
			sm.WriteString(filepath.Dir(target))
			sm.WriteRune('/')

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

type (
	Ssg struct {
		src   string
		dst   string
		title string
		url   string

		headers   headers
		footers   footers
		preferred setStr // Used to prefer html and ignore md files with identical names, as with the original ssg
		dist      []write
	}

	write struct {
		target string
		data   []byte
	}

	writeError struct {
		err    error
		target string
	}
)

func New(src, dst, title, url string) Ssg {
	return Ssg{
		src:   src,
		dst:   dst,
		title: title,
		url:   url,

		preferred: make(setStr),

		headers: headers{
			perDir: perDir[header]{
				d:      header{},
				values: make(map[string]header),
			},
		},

		footers: footers{
			perDir: perDir[*bytes.Buffer]{
				d:      bytes.NewBufferString(footerDefault),
				values: make(map[string]*bytes.Buffer),
			},
		},
	}
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
			return fmt.Errorf("ssg-go bug: unexpected missing stat for directory %s (url='%s')", s.src, s.url)
		}

		err := s.WriteOut()
		if err != nil {
			return fmt.Errorf("error writing out to %s: %w", s.dst, err)
		}

		if s.url == "" {
			s.pront(len(s.dist))
			return nil
		}

		sitemap, err := Sitemap(s.dst, s.url, stat.ModTime(), s.dist)
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

	if s.url == "" {
		s.pront(len(dist))
		return nil
	}

	sitemap, err := Sitemap(s.dst, s.url, stat.ModTime(), s.dist)
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

// Build walks the src directory, and converts Markdown into HTML,
// returning the results as []write.
//
// Build also caches the result in s for [WriteOut] later.
func (s *Ssg) Build() ([]write, error) {
	err := filepath.WalkDir(s.src, s.scan)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(s.src, s.build)
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

func shouldIgnore(base string, d fs.DirEntry) (bool, error) {
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
	ignore, err := shouldIgnore(base, d)
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
			return err
		}

		var from from
		switch {
		case bytes.Contains(data, []byte(placeholderFromH1)):
			from = fromH1

		case bytes.Contains(data, []byte(placeholderFromTag)):
			from = fromTag
		}

		err = s.headers.add(filepath.Dir(path), header{
			Buffer:    bytes.NewBuffer(data),
			titleFrom: from,
		})
		if err != nil {
			return err
		}

		return nil

	case "_footer.html":
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		err = s.footers.add(filepath.Dir(path), bytes.NewBuffer(data))
		if err != nil {
			return err
		}

		return nil
	}

	ext := filepath.Ext(base)
	if ext == ".html" {
		if s.preferred.insert(path) {
			err = fmt.Errorf("duplicate html file %s", path)
			return err
		}
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
	ignore, err := shouldIgnore(base, d)
	if err != nil {
		return err
	}
	if ignore {
		return nil
	}

	switch base {
	case
		"_header.html",
		"_footer.html":

		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
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
		return err
	}

	header := s.headers.choose(path)
	footer := s.footers.choose(path)

	// Copy header as string,
	// so the underlying bytes.Buffer is unchanged and ready for the next file
	headerText := header.String()
	switch header.titleFrom {
	case fromH1:
		headerText = titleFromH1(s.title, headerText, data)

	case fromTag:
		headerText, data = titleFromTag(s.title, headerText, data)
	}

	out := bytes.NewBufferString(headerText)
	out.Write(ToHtml(data))
	out.Write(footer.Bytes())

	s.dist = append(s.dist, write{
		target: target,
		data:   out.Bytes(),
	})

	return nil
}

func (s *Ssg) pront(l int) {
	fmt.Printf("[ssg-go] wrote %d file(s) to %s\n", l, s.dst)
}

func (w writeError) Error() string {
	return fmt.Errorf("WriteError(%s): %w", w.target, w.err).Error()
}

// TODO: Refactor
//
// titleFromH1 finds the first h1 in markdown and uses the h1 title
// to write to <title> tag in header.
func titleFromH1(d string, header string, markdown []byte) string {
	start := bytes.Index(markdown, []byte{'#', ' '})
	if start == -1 {
		header = strings.Replace(header, "{{from-h1}}", d, 1)
		return header
	}

	end := bytes.Index(markdown[start:], []byte{'\n', '\n'})
	if end == -1 {
		end = bytes.Index(markdown[start:], []byte{'\n'})
	}

	if end == -1 {
		header = strings.Replace(header, "{{from-h1}}", d, 1)
		return header
	}

	title := markdown[start+len(keyTitleH1) : start+end]
	header = strings.Replace(header, "{{from-h1}}", string(title), 1)

	return header
}

// titleFromTag finds title in markdown and then write it to <title> tag in header.
// It also deletes the tag line from markdown.
func titleFromTag(
	d string,
	header string,
	markdown []byte,
) (
	string,
	[]byte,
) {
	start := bytes.Index(markdown, []byte(keyTitleFromTag))
	if start == -1 {
		header = strings.Replace(header, targetFromTag, d, 1)
		return header, markdown
	}

	end := bytes.Index(markdown[start:], []byte{'\n', '\n'})

	title := markdown[start+len(keyTitleFromTag) : start+end]
	line := markdown[start : start+end+1]

	header = strings.Replace(header, targetFromTag, string(title), 1)
	markdown = bytes.Replace(markdown, line, nil, 1) // TODO: fix diff in test

	return header, markdown
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
