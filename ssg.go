package ssg

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sabhiram/go-gitignore"
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

	parallelWritesEnvKey      = "SSG_PARALLEL_WRITES"
	parallelWritesDefault int = 20
)

type Ssg struct {
	Src   string
	Dst   string
	Title string
	Url   string

	ssgignores     ignorer
	headers        headers
	footers        footers
	preferred      Set // Used to prefer html and ignore md files with identical names, as with the original ssg
	dist           []OutputFile
	parallelWrites int

	pipeline PipelineFn // Applied to all unignored files
	hook     HookFn     // Applied to converted files
}

// PipelineFn takes in a path and reads file data,
// returning modified output to be written at destination
type PipelineFn func(path string, data []byte) (output []byte, err error)

// HookFn takes in converted HTML bytes and returns modified HTML output
// (e.g. minified) to be written at destination
type HookFn func(htmlDoc []byte) (output []byte, err error)

type Option func(*Ssg)

type OutputFile struct {
	target string
	data   []byte
}

type writeError struct {
	err    error
	target string
}

type ignorerGitignore struct {
	*ignore.GitIgnore
}

type ignorer interface {
	ignore(path string) bool
}

func New(src, dst, title, url string) Ssg {
	ignores, err := prepare(src, dst)
	if err != nil {
		panic(err)
	}

	return Ssg{
		Src:            src,
		Dst:            dst,
		Title:          title,
		Url:            url,
		ssgignores:     ignores,
		preferred:      make(Set),
		headers:        newHeaders(headerDefault),
		footers:        newFooters(footerDefault),
		parallelWrites: parallelWritesDefault,
	}
}

func NewWithOptions(src, dst, title, url string, opts ...Option) Ssg {
	s := New(src, dst, title, url)
	s.With(opts...)

	return s
}

// ToHtml converts md (Markdown) into HTML document
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

		fmt.Fprintf(sm, "<url><loc>%s/", url)

		/* There're 2 possibilities for this
		1. First is when the HTML is some/path/index.html
		<url><loc>https://example.com/some/path/</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>

		2. Then there is when the HTML is some/path/page.html
		<url><loc>https://example.com/some/path/page.html</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>
		*/

		base := filepath.Base(target)
		switch base {
		case "index.html":
			d := filepath.Dir(target)
			if d != "." {
				fmt.Fprintf(sm, "%s/", d)
			}

		default:
			sm.WriteString(target)
		}

		fmt.Fprintf(sm, "><lastmod>%s</lastmod><priority>1.0</priority></url>\n", dateStr)
	}

	sm.WriteString("</urlset>")
	return sm.String(), nil
}

func prepare(src, dst string) (*ignorerGitignore, error) {
	if src == "" {
		return nil, fmt.Errorf("empty src")
	}
	if dst == "" {
		return nil, fmt.Errorf("empty dst")
	}
	if src == dst {
		return nil, fmt.Errorf("src is identical to dst: '%s'", src)
	}

	ssgignore := filepath.Join(src, ".ssgignore")
	ignores, err := ignore.CompileIgnoreFile(ssgignore)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse ssgignore at %s: %w", ssgignore, err)
	}

	return &ignorerGitignore{GitIgnore: ignores}, nil
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

func (s *Ssg) With(opts ...Option) *Ssg {
	for i := range opts {
		opts[i](s)
	}

	return s
}

func ParallelWritesEnv() Option {
	return func(s *Ssg) {
		writesEnv := os.Getenv(parallelWritesEnvKey)
		writes, err := strconv.ParseUint(writesEnv, 10, 32)
		if err == nil && writes != 0 {
			s.parallelWrites = int(writes)
		}
	}
}

// Pipeline will make [Ssg] call f(path, fileContent)
// on every unignored files.
func Pipeline(f func(string, []byte) ([]byte, error)) Option {
	return func(s *Ssg) {
		s.pipeline = f
	}
}

// Hook assigns f to be called on full output of files
// that will be converted by ssg from Markdown to HTML.
func Hook(f func([]byte) ([]byte, error)) Option {
	return func(s *Ssg) {
		s.hook = f
	}
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

	err = writeOut(s.dist, s.parallelWrites)
	if err != nil {
		return err
	}

	files := bytes.NewBuffer(nil)
	for i := range s.dist {
		f := &s.dist[i]
		path, err := filepath.Rel(s.Dst, f.target)
		if err != nil {
			return err
		}
		// Replace Markdown extension
		if filepath.Ext(path) == ".html" {
			path = strings.TrimSuffix(path, ".html")
			path += ".md"
		}

		fmt.Fprintf(files, "./%s\n", path)
	}

	target := filepath.Join(s.Dst, ".files")
	err = os.WriteFile(target, files.Bytes(), os.ModePerm)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", target, err)
	}

	return nil
}

func shouldIgnore(ignores ignorer, path, base string, d fs.DirEntry) (bool, error) {
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

	case ignores.ignore(path):
		return true, nil
	}

	// Ignore symlink
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	if FileIs(stat, os.ModeSymlink) {
		return true, nil
	}

	return false, nil
}

func (i *ignorerGitignore) ignore(path string) bool {
	if i == nil {
		return false
	}
	if i.GitIgnore == nil {
		return false
	}

	return i.MatchesPath(path)
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
	placeholderH1 := []byte(placeholderFromH1)
	placeholderTag := []byte(placeholderFromTag)
	switch base {
	case "_header.html":
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var from from
		switch {
		case bytes.Contains(data, placeholderH1):
			from = fromH1
		case bytes.Contains(data, placeholderTag):
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
	if s.preferred.Insert(path) {
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

	if s.pipeline != nil {
		data, err = s.pipeline(path, data)
		if err != nil {
			return fmt.Errorf("hook error when building %s: %w", path, err)
		}
	}

	ext := filepath.Ext(base)

	switch ext {
	// Check if there's a competing HTML file
	case ".md":
		html := strings.TrimSuffix(path, ".md")
		html += ".html"
		if s.preferred.ContainsAll(html) {
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
	headerText := []byte(header.String()) //nolint:gosimple
	switch header.titleFrom {
	case fromH1:
		headerText = titleFromH1([]byte(s.Title), headerText, data)

	case fromTag:
		headerText, data = titleFromTag([]byte(s.Title), headerText, data)
	}

	out := bytes.NewBuffer(headerText)
	out.Write(ToHtml(data))
	out.Write(footer.Bytes())

	if s.hook != nil {
		b, err := s.hook(out.Bytes())
		if err != nil {
			return fmt.Errorf("hook error when building %s: %w", path, err)
		}

		out = bytes.NewBuffer(b)
	}

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
	k := []byte(keyTitleFromH1)
	t := []byte(targetFromH1)
	s := bufio.NewScanner(bytes.NewBuffer(markdown))

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
	k := []byte(keyTitleFromTag)
	t := []byte(targetFromTag)
	s := bufio.NewScanner(bytes.NewBuffer(markdown))

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
func writeOut(writes []OutputFile, parallelWrites int) error {
	wg := new(sync.WaitGroup)
	errs := make(chan writeError)
	guard := make(chan struct{}, parallelWrites)

	for i := range writes {
		guard <- struct{}{}
		wg.Add(1)

		go func(w *OutputFile, wg *sync.WaitGroup) {
			defer func() {
				<-guard
				wg.Done()
			}()

			err := os.MkdirAll(filepath.Dir(w.target), os.ModePerm)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}
				return
			}
			err = os.WriteFile(w.target, w.data, os.ModePerm)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}
				return
			}

			fmt.Fprintln(os.Stdout, w.target)

		}(&writes[i], wg)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	var wErrs []error
	for err := range errs { // Blocks here until errs is closed
		wErrs = append(wErrs, err)
	}
	if len(wErrs) > 0 {
		return errors.Join(wErrs...)
	}

	return nil
}
