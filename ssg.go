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

func NewSsg(src, dst string) Ssg {
	return Ssg{
		src:  src,
		dist: dst,

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
	_, err := os.Stat(s.dist)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.dist, os.ModePerm)
	}

	if err != nil {
		return err
	}

	err = filepath.WalkDir(s.src, s.walk)
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

func (s *Ssg) Generate() error {
	err := s.Walk()
	if err != nil {
		return err
	}

	return s.WriteOut()
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

	dir := filepath.Dir(path)
	switch filepath.Base(path) {
	case "_header.html":
		h, err := os.Open(path)
		if err != nil {
			s.walkError = err
			return fs.SkipAll
		}

		header := bytes.NewBuffer(nil)
		_, err = header.ReadFrom(h)
		if err != nil {
			s.walkError = err
			return fs.SkipAll
		}

		s.headers.values[dir] = header

	case "_footer.html":
		f, err := os.Open(path)
		if err != nil {
			s.walkError = err
			return fs.SkipAll
		}

		footer := bytes.NewBuffer(nil)
		_, err = footer.ReadFrom(f)
		if err != nil {
			s.walkError = err
			return fs.SkipAll
		}

		s.footers.values[dir] = footer
	}

	if filepath.Ext(path) != ".md" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		s.walkError = err
		return filepath.SkipAll
	}

	target, err := s.targetPath(path)
	if err != nil {
		s.walkError = err
		return filepath.SkipAll
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

func (s *Ssg) targetPath(p string) (string, error) {
	p = strings.TrimSuffix(p, ".md")
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
