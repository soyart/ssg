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
	"github.com/sabhiram/go-gitignore"
)

const (
	MarkerHeader  = "_header.html"
	MarkerFooter  = "_footer.html"
	HtmlFlags     = html.CommonFlags
	SsgExtensions = parser.CommonExtensions |
		parser.Mmark |
		parser.AutoHeadingIDs

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

	parallelWritesEnvKey      = "SSG_PARALLEL_WRITES"
	parallelWritesDefault int = 20
)

type Ssg struct {
	Src   string
	Dst   string
	Title string
	Url   string

	options

	ssgignores ignorer
	headers    headers
	footers    footers
	preferred  Set // Used to prefer html and ignore md files with identical names, as with the original ssg
	dist       []OutputFile
}

// Generate creates a one-off [Ssg] that's used to generate a site right away.
func Generate(src, dst, title, url string, opts ...Option) error {
	s := New(src, dst, title, url)
	return s.
		With(opts...).
		Generate()
}

// New returns a default, vanilla [Ssg].
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
		preferred:  make(Set),
		headers:    newHeaders(headerDefault),
		footers:    newFooters(footerDefault),
	}
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
	sm := bytes.NewBufferString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset
xmlns:xsi="https://www.w3.org/2001/XMLSchema-instance"
xsi:schemaLocation="https://www.sitemaps.org/schemas/sitemap/0.9
https://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"
xmlns="https://www.sitemaps.org/schemas/sitemap/0.9">
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

	sm.WriteString("</urlset>\n")
	return sm.String(), nil
}

func (s *Ssg) AddOutputs(outputs ...OutputFile) {
	s.dist = append(s.dist, outputs...)
}

func (s *Ssg) With(opts ...Option) *Ssg {
	for i := range opts {
		opts[i](s)
	}
	return s
}

func (s *Ssg) Generate() error {
	// Reset
	s.dist = nil

	stat, err := os.Stat(s.Src)
	if err != nil {
		return err
	}
	dist, err := s.build()
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
	err = os.WriteFile(s.Dst+"/sitemap.xml", []byte(sitemap), 0644)
	if err != nil {
		return err
	}

	s.pront(len(dist) + 1)
	return nil
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

	err = WriteOut(s.dist, s.parallelWrites)
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
	err = os.WriteFile(target, files.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", target, err)
	}

	return nil
}

func (s *Ssg) buildV2() ([]OutputFile, error) {
	err := filepath.WalkDir(s.Src, s.walkBuildV2)
	if err != nil {
		return nil, err
	}

	return s.dist, nil
}

func (s *Ssg) walkBuildV2(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		fmt.Println("collecting", path)
		return s.collect(path)
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
		MarkerHeader,
		MarkerFooter:
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if s.impl != nil {
		return s.impl(path, data, d)
	}

	return s.implDefault(path, data, d)
}

func (s *Ssg) collect(path string) error {
	children, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for i := range children {
		child := children[i]
		base := child.Name()
		pathChild := filepath.Join(path, base)

		switch base {
		case MarkerHeader:
			data, err := os.ReadFile(pathChild)
			if err != nil {
				return err
			}
			err = s.headers.add(path, header{
				Buffer:    bytes.NewBuffer(data),
				titleFrom: GetTitleFrom(data),
			})
			if err != nil {
				return err
			}

			continue

		case MarkerFooter:
			data, err := os.ReadFile(pathChild)
			if err != nil {
				return err
			}
			err = s.footers.add(path, bytes.NewBuffer(data))
			if err != nil {
				return err
			}

			continue
		}

		ext := filepath.Ext(base)
		if ext != ".html" {
			continue
		}

		if s.preferred.Insert(pathChild) {
			return fmt.Errorf("duplicate html file %s", path)
		}
	}

	return nil
}

// build walks the src directory, and converts Markdown into HTML,
// returning the results as []write.
//
// build also caches the result in s for [WriteOut] later.
func (s *Ssg) build() ([]OutputFile, error) {
	err := filepath.WalkDir(s.Src, s.walkScan)
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(s.Src, s.walkBuild)
	if err != nil {
		return nil, err
	}

	return s.dist, nil
}

// walkScan scans the source directory for header and footer files,
// and anything required to build a page.
func (s *Ssg) walkScan(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
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
	case MarkerHeader:
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		err = s.headers.add(filepath.Dir(path), header{
			Buffer:    bytes.NewBuffer(data),
			titleFrom: GetTitleFrom(data),
		})
		if err != nil {
			return err
		}

		return nil

	case MarkerFooter:
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

// walkBuild finds and converts Markdown files to HTML,
// and assembles it with header and footer.
func (s *Ssg) walkBuild(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
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
		MarkerHeader,
		MarkerFooter:
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if s.impl != nil {
		return s.impl(path, data, d)
	}

	return s.implDefault(path, data, d)
}

func (s *Ssg) implDefault(path string, data []byte, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}

	if s.hookAll != nil {
		data, err = s.hookAll(path, data)
		if err != nil {
			return fmt.Errorf("hook error when building %s: %w", path, err)
		}
	}

	ext := filepath.Ext(path)
	if ext != ".md" {
		target, err := MirrorPath(s.Src, s.Dst, path, ext)
		if err != nil {
			return err
		}
		s.AddOutputs(Output(
			target,
			data,
			info.Mode().Perm(),
		))

		return nil
	}

	html := strings.TrimSuffix(path, ".md")
	html += ".html"
	if s.preferred.ContainsAll(html) {
		return nil
	}

	target, err := MirrorPath(s.Src, s.Dst, path, ".html")
	if err != nil {
		return err
	}

	header := s.headers.choose(path)
	footer := s.footers.choose(path)

	// Copy header as string,
	// so the underlying bytes.Buffer is unchanged and ready for the next file
	headerText := []byte(header.String()) //nolint:gosimple
	switch header.titleFrom {
	case TitleFromH1:
		headerText = AddTitleFromH1([]byte(s.Title), headerText, data)

	case TitleFromTag:
		headerText, data = AddTitleFromTag([]byte(s.Title), headerText, data)
	}

	out := bytes.NewBuffer(headerText)
	out.Write(ToHtml(data))
	out.Write(footer.Bytes())

	if s.hookGenerate != nil {
		b, err := s.hookGenerate(out.Bytes())
		if err != nil {
			return fmt.Errorf("hook error when building %s: %w", path, err)
		}

		out = bytes.NewBuffer(b)
	}

	s.AddOutputs(Output(
		target,
		out.Bytes(),
		info.Mode().Perm(),
	))

	return nil
}

func (s *Ssg) ImplDefault() Impl {
	return s.implDefault
}

func (s *Ssg) pront(l int) {
	fmt.Fprintf(os.Stdout, "[ssg-go] wrote %d file(s) to %s\n", l, s.Dst)
}

type ignorer interface {
	ignore(path string) bool
}

type ignorerGitignore struct {
	*ignore.GitIgnore
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

// MirrorPath mirrors the target HTML file path under src to under dist
//
// i.e. if src="foo/src" and dst="foo/dist",
// and path="foo/src/bar/baz.md"  newExt=".html",
// then the return value will be foo/dist/bar/baz.html
func MirrorPath(
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

type OutputFile struct {
	target string
	data   []byte
	perm   fs.FileMode
}

func Output(target string, data []byte, perm fs.FileMode) OutputFile {
	return OutputFile{
		target: target,
		data:   data,
		perm:   perm,
	}
}

func (o *OutputFile) modeOutput() fs.FileMode {
	if o.perm == fs.FileMode(0) {
		return fs.ModePerm
	}

	return o.perm
}

type writeError struct {
	err    error
	target string
}

func (w writeError) Error() string {
	return fmt.Errorf("WriteError(%s): %w", w.target, w.err).Error()
}

// WriteOut blocks and writes concurrently to output locations.
func WriteOut(writes []OutputFile, parallelWrites int) error {
	if parallelWrites == 0 {
		parallelWrites = parallelWritesDefault
	}

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
			err = os.WriteFile(w.target, w.data, w.modeOutput())
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
