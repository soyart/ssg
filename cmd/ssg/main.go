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
<body>`

	footer = `
</body>
</html>`
)

func main() {
	errs := make(chan error)
	writes := make(chan write)

	site := site{
		src:    "artnoi.com/src",
		dist:   "artnoi.com/dist",
		header: bytes.NewBufferString(header),
		footer: bytes.NewBufferString(footer),
		writes: writes,
		errs:   errs,
	}

	err := site.gen()
	if err != nil {
		panic(err)
	}
}

type site struct {
	header *bytes.Buffer
	footer *bytes.Buffer

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

	switch filepath.Base(path) {
	case "_header.html":
		h, err := os.Open(path)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		s.header.Reset()
		_, err = s.header.ReadFrom(h)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

	case "_footer.html":
		f, err := os.Open(path)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}

		s.footer.Reset()
		_, err = s.footer.ReadFrom(f)
		if err != nil {
			s.exitError = err
			return fs.SkipAll
		}
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
	w := write{
		target: target, // TODO: cut path and create dir
		data:   []byte(s.header.String() + string(body) + s.footer.String()),
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
	for {
		w, ok := <-writes
		if !ok {
			return
		}

		fmt.Println("Writing out:", w.target)

		go func() {
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
		}()
	}
}
