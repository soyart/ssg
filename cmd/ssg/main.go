package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/soyart/ssg"
)

const (
	header = `
<!DOCTYPE html>
<html lang="en">
	<head>
	  <meta charset="UTF-8">
	</head>
	<body>
`

	footer = `
	</body>
</html>
`
)

func main() {
	errs := make(chan error)
	writes := make(chan write)

	src := "johndoe.com/src"
	dst := "johndoe.com/dist"

	site := site{
		src:    src,
		dist:   dst,
		writes: writes,
		errs:   errs,

		headers: perDir{
			defaultValue: bytes.NewBufferString(header),
			values:       map[string]*bytes.Buffer{},
		},

		footers: perDir{
			defaultValue: bytes.NewBufferString(footer),
			values:       map[string]*bytes.Buffer{},
		},
	}

	err := site.gen()
	if err != nil {
		panic(err)
	}
}

type site struct {
	headers   perDir
	footers   perDir
	writes    chan write
	errs      chan error
	exitError error

	src  string
	dist string
}

type write struct {
	target string
	data   []byte
}

type perDir struct {
	defaultValue *bytes.Buffer
	values       map[string]*bytes.Buffer
}

func (s *site) gen() error {
	_, err := os.Stat(s.dist)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.dist, os.ModePerm)
	}

	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		writeOut(s.writes, s.errs)
	}(&wg)

	err = filepath.WalkDir(s.src, s.walk)
	if err != nil {
		err = s.exitError
	}

	close(s.writes)
	close(s.errs)

	wg.Wait()

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

	dir := filepath.Dir(path)
	switch filepath.Base(path) {
	case "_header.html":
		h, err := os.Open(path)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		header := bytes.NewBuffer(nil)
		_, err = header.ReadFrom(h)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		s.headers.values[dir] = header

	case "_footer.html":
		f, err := os.Open(path)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		footer := bytes.NewBuffer(nil)
		_, err = footer.ReadFrom(f)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		s.footers.values[dir] = footer
	}

	if filepath.Ext(path) != ".md" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		s.exitError = err
		return filepath.SkipAll
	}

	target, err := s.targetPath(path)
	if err != nil {
		s.exitError = err
		return filepath.SkipAll
	}

	body := ssg.ToHtml(data)

	header := s.headers.defaultValue
	footer := s.footers.defaultValue

	max := 0
	for k, h := range s.headers.values {
		if !strings.HasPrefix(path, k) {
			continue
		}

		if len(k) < max {
			continue
		}

		max = len(k)
		header = h
	}

	max = 0
	for k, h := range s.footers.values {
		if !strings.HasPrefix(path, k) {
			continue
		}

		if len(k) < max {
			continue
		}

		max = len(k)
		footer = h
	}

	w := write{
		target: target, // TODO: cut path and create dir
		data:   []byte(header.String() + string(body) + footer.String()),
	}

	s.writes <- w

	return nil
}

func (s *site) targetPath(p string) (string, error) {
	p = strings.TrimSuffix(p, ".md")
	p += ".html"

	p, err := filepath.Rel(s.src, p)
	if err != nil {
		return "", err
	}

	return filepath.Join(s.dist, p), nil
}

func writeOut(writes <-chan write, _ chan<- error) {
	wg := sync.WaitGroup{}

	for {
		w, ok := <-writes
		if !ok {
			break
		}

		wg.Add(1)
		go func(w *write) {
			defer func() {
				wg.Done()
				fmt.Println(w.target)
			}()

			d := filepath.Dir(w.target)
			err := os.MkdirAll(d, os.ModePerm)
			if err != nil {
				panic(err)
			}

			f, err := os.OpenFile(w.target, os.O_CREATE|os.O_WRONLY, os.ModePerm)
			if err != nil {
				panic(err)
			}

			defer f.Close()

			_, err = f.Write(w.data)
			if err != nil {
				panic(err)
			}
		}(&w)
	}

	wg.Wait()
}
