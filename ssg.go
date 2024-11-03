package ssg

import (
	"bufio"
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

	headerDefault = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-h1}}</title>
</head>
<body>
`

	footerDefault = `</body>
</html>
`

	keyTitleFromH1     = "# "      // The first h1 tag is used as document header title
	keyTitleFromTag    = ":title " // The first line starting with :title will be parsed as document header title
	targetFromH1       = "{{from-h1}}"
	targetFromTag      = "{{from-tag}}"
	placeholderFromH1  = "<title>" + targetFromH1 + "</title>"
	placeholderFromTag = "<title>" + targetFromTag + "</title>"
)

type (
	Ssg struct {
		Src   string `json:"src"`
		Dst   string `json:"dst"`
		Title string `json:"title"`
		Url   string `json:"url"`

		ssgignores setStr
		headers    headers
		footers    footers
		preferred  setStr // Used to prefer html and ignore md files with identical names, as with the original ssg
		dist       []OutputFile
	}

	OutputFile struct {
		target string
		data   []byte
	}

	writeError struct {
		err    error
		target string
	}
)

func New(src, dst, title, url string) Ssg {
	ignores, err := prepare(src, dst)
	if err != nil {
		panic(err)
	}

	return Ssg{
		Src:        src,
		Dst:        dst,
		Title:      title,
		Url:        url,
		ssgignores: ignores,
		preferred:  make(setStr),
		headers:    newHeaders(headerDefault),
		footers:    newFooters(footerDefault),
	}
}

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
	outputs []OutputFile,
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

	for i := range outputs {
		o := &outputs[i]

		target, err := filepath.Rel(dst, o.target)
		if err != nil {
			return sm.String(), err
		}

		opening := fmt.Sprintf("<url><loc>%s/", url)
		sm.WriteString(opening)

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

		closing := fmt.Sprintf("><lastmod>%s</lastmod><priority>1.0</priority></url>\n", dateStr)
		sm.WriteString(closing)
	}

	sm.WriteString("</urlset>")

	return sm.String(), nil
}

func prepare(src, dst string) (setStr, error) {
	if src == "" {
		return nil, fmt.Errorf("empty src")
	}

	if dst == "" {
		return nil, fmt.Errorf("empty dst")
	}

	if src == dst {
		return nil, fmt.Errorf("src is identical to dst: '%s'", src)
	}

	b, err := os.ReadFile(filepath.Join(src, ".ssgignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	ignores := make(setStr)

	s := bufio.NewScanner(bytes.NewBuffer(b))
	for s.Scan() {
		ignore := s.Text()
		ignore = filepath.Join(src, ignore)

		if ignores.contains(ignore) {
			return nil, fmt.Errorf("duplicate ssgignore entry: '%s'", ignore)
		}

		ignores.insert(ignore)
	}

	return ignores, nil
}

func Generate(sites ...Ssg) error {
	stats := make(map[string]fs.FileInfo)

	for i := range sites {
		s := &sites[i]
		stat, err := os.Stat(s.Src)
		if err != nil {
			return err
		}

		stats[s.Src] = stat

		_, err = s.Build()
		if err != nil {
			return fmt.Errorf("error walking in %s: %w", s.Src, err)
		}
	}

	for i := range sites {
		s := &sites[i]
		stat, ok := stats[s.Src]
		if !ok {
			return fmt.Errorf("ssg-go bug: unexpected missing stat for directory %s (url='%s')", s.Src, s.Url)
		}

		err := s.WriteOut()
		if err != nil {
			return fmt.Errorf("error writing out to %s: %w", s.Dst, err)
		}

		if s.Url == "" {
			s.pront(len(s.dist))
			return nil
		}

		sitemap, err := Sitemap(s.Dst, s.Url, stat.ModTime(), s.dist)
		if err != nil {
			return err
		}

		err = os.WriteFile(s.Dst+"/sitemap.xml", []byte(sitemap), os.ModePerm)
		if err != nil {
			return err
		}

		s.pront(len(s.dist) + 1)
	}

	return nil
}

func (s *Ssg) Generate() error {
	stat, err := os.Stat(s.Src)
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

	if s.Url == "" {
		s.pront(len(dist))
		return nil
	}

	sitemap, err := Sitemap(s.Dst, s.Url, stat.ModTime(), s.dist)
	if err != nil {
		return err
	}

	err = os.WriteFile(s.Dst+"/sitemap.xml", []byte(sitemap), os.ModePerm)
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
func (s *Ssg) Build() ([]OutputFile, error) {
	err := filepath.WalkDir(s.Src, s.scan)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(s.Src, s.build)
	if err != nil {
		return nil, err
	}

	return s.dist, nil
}

// WriteOut blocks and concurrently writes out s.writes
// to their target locations.
//
// If targets is empty, WriteOut writes to s.dst
func (s *Ssg) WriteOut() error {
	_, err := os.Stat(s.Dst)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.Dst, os.ModePerm)
	}
	if err != nil {
		return err
	}

	err = writeOut(s.dist)
	if err != nil {
		return err
	}

	files := new(strings.Builder)
	for i := range s.dist {
		f := &s.dist[i]
		path, err := filepath.Rel(s.Dst, f.target)
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".html" {
			path = strings.TrimSuffix(path, ".html")
			path += ".md"
		}

		files.WriteString("./")
		files.WriteString(path)
		files.WriteRune('\n')
	}

	err = os.WriteFile(filepath.Join(s.Dst, ".files"), []byte(files.String()), os.ModePerm)
	if err != nil {
		return fmt.Errorf("error writing .files: %w", err)
	}

	return nil
}

func shouldIgnore(ignores setStr, path, base string, d fs.DirEntry) (bool, error) {
	isDot := strings.HasPrefix(base, ".")
	isDir := d.IsDir()

	switch {
	case base == ".ssgignore":
		return true, nil

	case isDot && isDir:
		return true, fs.SkipDir

	// Ignore hidden files and dir
	case isDot, isDir:
		return true, nil

	case ignores.contains(path):
		return true, nil
	}

	for ignored := range ignores {
		if strings.HasPrefix(path, ignored) {
			return true, nil
		}
	}

	// Ignore symlink
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return false, err
	}
	if fileIs(stat, os.ModeSymlink) {
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
	ignore, err := shouldIgnore(s.ssgignores, path, base, d)
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

	if filepath.Ext(base) != ".html" {
		return nil
	}

	if s.preferred.insert(path) {
		return fmt.Errorf("duplicate html file %s", path)
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
	ignore, err := shouldIgnore(s.ssgignores, path, base, d)
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
		target, err := mirrorPath(s.Src, s.Dst, path, ext)
		if err != nil {
			return err
		}

		s.dist = append(s.dist, OutputFile{
			target: target,
			data:   data,
		})

		return nil
	}

	target, err := mirrorPath(s.Src, s.Dst, path, ".html")
	if err != nil {
		return err
	}

	header := s.headers.choose(path)
	footer := s.footers.choose(path)

	// Copy header as string,
	// so the underlying bytes.Buffer is unchanged and ready for the next file
	headerText := []byte(header.String())
	switch header.titleFrom {
	case fromH1:
		headerText = titleFromH1([]byte(s.Title), headerText, data)

	case fromTag:
		headerText, data = titleFromTag([]byte(s.Title), headerText, data)
	}

	out := bytes.NewBuffer(headerText)
	out.Write(ToHtml(data))
	out.Write(footer.Bytes())

	s.dist = append(s.dist, OutputFile{
		target: target,
		data:   out.Bytes(),
	})

	return nil
}

func (s *Ssg) pront(l int) {
	fmt.Fprintf(os.Stdout, "[ssg-go] wrote %d file(s) to %s\n", l, s.Dst)
}

func (w writeError) Error() string {
	return fmt.Errorf("WriteError(%s): %w", w.target, w.err).Error()
}

// titleFromH1 finds the first h1 in markdown and uses the h1 title
// to write to <title> tag in header.
func titleFromH1(d []byte, header []byte, markdown []byte) []byte {
	s := bufio.NewScanner(bytes.NewBuffer(markdown))
	k := []byte(keyTitleFromH1)
	t := []byte(targetFromH1)

	for s.Scan() {
		line := s.Bytes()
		if !bytes.HasPrefix(line, k) {
			continue
		}
		parts := bytes.Split(line, k)
		if len(parts) != 2 {
			continue
		}

		title := parts[1]
		header = bytes.Replace(header, t, title, 1)
		return header
	}

	header = bytes.Replace(header, t, d, 1)
	return header
}

// titleFromTag finds title in markdown and then write it to <title> tag in header.
// It also deletes the tag line from markdown.
func titleFromTag(
	d []byte,
	header []byte,
	markdown []byte,
) (
	[]byte,
	[]byte,
) {
	s := bufio.NewScanner(bytes.NewBuffer(markdown))
	k := []byte(keyTitleFromTag)
	t := []byte(targetFromTag)
	for s.Scan() {
		line := s.Bytes()
		if !bytes.HasPrefix(line, k) {
			continue
		}
		parts := bytes.Split(line, k)
		if len(parts) != 2 {
			continue
		}

		line = trimRightWhitespace(line)
		title := parts[1]

		header = bytes.Replace(header, t, title, 1)
		markdown = bytes.Replace(markdown, append(line, []byte{'\n', '\n'}...), nil, 1)

		return header, markdown
	}

	// Remove target and use default header string
	header = bytes.Replace(header, t, []byte(d), 1)
	return header, markdown
}

func trimRightWhitespace(b []byte) []byte {
	return bytes.TrimRightFunc(b, func(r rune) bool {
		switch r {
		case ' ', '\t':
			return true
		}

		return false
	})
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

// writeOut blocks and writes concurrently to output locations.
func writeOut(writes []OutputFile) error {
	wgErrs := new(sync.WaitGroup)
	errs := make(chan writeError)

	var err error
	wgErrs.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		var wErrs []error

		for err := range errs {
			wErrs = append(wErrs, err)
		}

		err = errors.Join(wErrs...)
	}(wgErrs)

	wgWrites := new(sync.WaitGroup)
	guard := make(chan struct{}, 20)
	for i := range writes {
		wgWrites.Add(1)
		guard <- struct{}{}

		go func(w *OutputFile, wg *sync.WaitGroup) {
			var err error

			defer func() {
				wg.Done()
				fmt.Fprintf(os.Stdout, "%s\n", w.target)
			}()

			<-guard

			d := filepath.Dir(w.target)
			err = os.MkdirAll(d, os.ModePerm)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}

				return
			}

			f, err := os.OpenFile(w.target, os.O_RDWR|os.O_CREATE, os.ModePerm)
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
		}(&writes[i], wgWrites)
	}

	wgWrites.Wait()
	close(errs)

	wgErrs.Wait()
	return err
}
