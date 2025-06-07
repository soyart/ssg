package ssg

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	ignore "github.com/sabhiram/go-gitignore"
)

const (
	MarkerHeader = "_header.html"
	MarkerFooter = "_footer.html"
	SsgIgnore    = ".ssgignore"

	WritersEnvKey      = "SSG_WRITERS"
	WritersDefault int = 20

	HtmlFlags     = html.CommonFlags
	SsgExtensions = parser.CommonExtensions |
		parser.Mmark |
		parser.AutoHeadingIDs

	HeaderDefault = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-h1}}</title>
</head>
<body>
`
	FooterDefault = `</body>
</html>
`
)

var (
	// ErrBreakPipelines causes Ssg to break from pipeline iteration
	// and use the pipeline's output
	ErrBreakPipelines = errors.New("ssg_break_pipeline")

	// ErrSkipCore causes Ssg to break from pipeline iteration
	// and skip core processor, continuing to the new input file.
	ErrSkipCore = errors.New("ssg_skip_core")
)

type Ssg struct {
	Src   string
	Dst   string
	Title string
	Url   string

	options options

	outputs    Outputs
	ssgignores func(path string) (ignore bool)
	headers    headers
	footers    footers
	preferred  Set      // Used to prefer html and ignore md files with identical names, as with the original ssg
	files      []string // Input files read (not ignored)
	cache      []OutputFile
}

func (s *Ssg) Options() Options { return s.options }

// Build returns the ssg outputs built from src without writing the outputs.
func Build(src, dst, title, url string, opts ...Option) ([]string, []OutputFile, error) {
	s := New(src, dst, title, url)
	return s.
		With(opts...).
		With(Caching()).
		buildV2(nil)
}

// Generate builds and writes to outputs.
// It creates a one-off [Ssg] that's used to generate a site right away.
func Generate(src, dst, title, url string, opts ...Option) error {
	s := New(src, dst, title, url)
	return s.
		With(opts...).
		Generate()
}

// New returns a default [Ssg] with no options.
func New(src, dst, title, url string) Ssg {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	ignores, err := prepare(src, dst)
	if err != nil {
		panic(err)
	}

	return Ssg{
		Src:        src,
		Dst:        dst,
		Title:      title,
		Url:        url,
		ssgignores: ignores.Ignore,
		preferred:  make(Set),
		headers:    newHeaders(HeaderDefault),
		footers:    newFooters(FooterDefault),
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

// With applies opts to s sequentially
func (s *Ssg) With(opts ...Option) *Ssg {
	for i := range opts {
		opts[i](s)
	}
	return s
}

// Generate builds from s.Src and writes the outputs to s.Dst
func (s *Ssg) Generate() error {
	return generate(s)
}

// AddOutputs adds outputs to cache (if enabled)
// and sends the outputs to output stream to concurrent writers.
// **It does not write the outputs**.
func (s *Ssg) AddOutputs(outputs ...OutputFile) {
	if s.options.caching {
		s.cache = append(s.cache, outputs...)
	}
	s.outputs.AddOutputs(outputs...)
}

func (s *Ssg) buildV2(o Outputs) ([]string, []OutputFile, error) {
	defer func() {
		s.outputs = nil
	}()
	s.outputs = o

	err := filepath.WalkDir(s.Src, s.walkBuildV2)
	if err != nil {
		return nil, nil, err
	}
	return s.files, s.cache, nil
}

func (s *Ssg) walkBuildV2(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
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
		MarkerFooter,
		SsgIgnore:

		return nil
	}

	data, err := ReadFile(path)
	if err != nil {
		return err
	}

	// Remember input files for .files
	//
	// Original ssg does not include _header.html
	// and _footer.html in .files
	s.files = append(s.files, path)

	skipCore := false
	for i, p := range s.options.pipelines {
		path, data, d, err = p(path, data, d)
		if err == nil {
			continue
		}
		if errors.Is(err, ErrSkipCore) {
			skipCore = true
			break
		}
		if errors.Is(err, ErrBreakPipelines) {
			break
		}
		return fmt.Errorf("[pipeline %d] error: %w", i, err)
	}

	if skipCore {
		return nil
	}

	output, err := s.core(path, data, d)
	if err != nil {
		return fmt.Errorf("core error: %w", err)
	}
	s.outputs.AddOutputs(output)
	return nil
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
			data, err := ReadFile(pathChild)
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
			data, err := ReadFile(pathChild)
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

// core does 2 things:
// - If path extension is not .md, then the current file will
// simply be copied to outputs.
// - If path has .md extension, it converts Markdown to HTML
// and adds a new output with .html extension
func (s *Ssg) core(path string, data []byte, d fs.DirEntry) (OutputFile, error) {
	info, err := d.Info()
	if err != nil {
		return OutputFile{}, err
	}
	for i, hook := range s.options.hooks {
		data, err = hook(path, data)
		if err != nil {
			return OutputFile{}, fmt.Errorf("hooks[%d]: error when building %s: %w", i, path, err)
		}
	}

	ext := filepath.Ext(path)
	if ext != ".md" {
		target, err := mirrorPath(s.Src, s.Dst, path)
		if err != nil {
			return OutputFile{}, err
		}
		return Output(
			target,
			path,
			data,
			info.Mode().Perm(),
		), nil
	}

	// Make way for existing (preferred) html file with matching base name
	if s.preferred.Contains(
		ChangeExt(path, ".md", ".html"),
	) {
		return OutputFile{}, nil
	}

	target, err := mirrorPath(s.Src, s.Dst, path)
	if err != nil {
		return OutputFile{}, err
	}

	target = ChangeExt(target, ".md", ".html")
	header := s.headers.choose(path)
	footer := s.footers.choose(path)

	// Copy, leave the underlying data in header unchanged
	headerText := make([]byte, header.Len())
	_ = copy(headerText, header.Bytes())

	switch header.titleFrom {
	case TitleFromH1:
		headerText = AddTitleFromH1([]byte(s.Title), headerText, data)

	case TitleFromTag:
		headerText, data = AddTitleFromTag([]byte(s.Title), headerText, data)
	}

	// HTML output buffer
	buf := bytes.NewBuffer(headerText)
	buf.Write(ToHtml(data))
	buf.Write(footer.Bytes())

	for i, h := range s.options.hookGenerate {
		b, err := h(buf.Bytes())
		if err != nil {
			return OutputFile{}, fmt.Errorf("hooksGenerate[%d] error when building %s: %w", i, path, err)
		}
		buf = bytes.NewBuffer(b)
	}

	return Output(
		target,
		path,
		buf.Bytes(),
		info.Mode().Perm(),
	), nil
}

func (s *Ssg) Ignore(path string) bool {
	return s.ssgignores(path)
}

func (s *Ssg) pront(l int) {
	Fprintf(os.Stdout, "[ssg-go] wrote %d file(s) to %s\n", l, s.Dst)
}

func prepare(src, dst string) (*gitIgnorer, error) {
	if src == "" {
		return nil, fmt.Errorf("empty src")
	}
	if dst == "" {
		return nil, fmt.Errorf("empty dst")
	}
	if src == dst {
		return nil, fmt.Errorf("src is identical to dst: '%s'", src)
	}

	ssgignore := filepath.Join(src, SsgIgnore)
	return ParseSsgIgnore(ssgignore)
}

func ParseSsgIgnore(path string) (*gitIgnorer, error) {
	ignores, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse ssgignore at %s: %w", path, err)
	}

	return &gitIgnorer{GitIgnore: ignores}, nil
}

type gitIgnorer struct {
	*ignore.GitIgnore
}

func (i *gitIgnorer) Ignore(path string) bool {
	if i == nil {
		return false
	}
	if i.GitIgnore == nil {
		return false
	}
	return i.MatchesPath(path)
}

// TODO: refactor
func shouldIgnore(ignoreFn func(path string) (ignored bool), path, base string, d fs.DirEntry) (bool, error) {
	isDot := strings.HasPrefix(base, ".")
	isDir := d.IsDir()

	switch {
	case isDot && isDir:
		return true, fs.SkipDir

	// Ignore hidden files and dir
	case isDot, isDir:
		return true, nil

	case ignoreFn(path):
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

// mirrorPath mirrors the target HTML file path under src to under dist
//
// i.e. if src="foo/src" and dst="foo/dist",
// and path="foo/src/bar/baz.md"  newExt=".html",
// then the return value will be foo/dist/bar/baz.html
func mirrorPath(
	src string,
	dst string,
	path string,
) (
	string,
	error,
) {
	path, err := filepath.Rel(src, path)
	if err != nil {
		return "", err
	}

	return filepath.Join(dst, path), nil
}
