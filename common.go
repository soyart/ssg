package ssg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type (
	// OutputFile is the main output struct for ssg-go.
	//
	// Its values are not supposed to be changed by other packages,
	// and thus the only ways other packages can work with OutputFile
	// is via the constructor [Output] and the type's getter methods.
	OutputFile struct {
		target     string
		originator string
		data       []byte
		perm       fs.FileMode
	}

	Set map[string]struct{}

	// perDir tracks files under directory in a trie-like fashion.
	// Used to choose headers and footers.
	perDir[T any] struct {
		defaultValue T
		values       map[string]T
	}

	header struct {
		*bytes.Buffer
		titleFrom TitleFrom
	}

	headers struct {
		perDir[header]
	}

	footers struct {
		perDir[*bytes.Buffer]
	}
)

func FileIs(f os.FileInfo, mode fs.FileMode) bool {
	return f.Mode()&mode != 0
}

func ChangeExt(path, old, new string) string {
	path = strings.TrimSuffix(path, old)
	return path + new
}

func (s Set) Insert(v string) bool {
	_, ok := s[v]
	s[v] = struct{}{}
	return ok
}

func (s Set) Contains(items ...string) bool {
	for _, v := range items {
		_, ok := s[v]
		if !ok {
			return false
		}
	}
	return true
}

func Fprint(w io.Writer, data ...interface{}) {
	_, err := fmt.Fprint(w, data...)
	if err != nil {
		panic(err)
	}
}

func Fprintf(w io.Writer, format string, data ...interface{}) {
	_, err := fmt.Fprintf(w, format, data...)
	if err != nil {
		panic(err)
	}
}

func Fprintln(w io.Writer, data ...interface{}) {
	_, err := fmt.Fprintln(w, data...)
	if err != nil {
		panic(err)
	}
}

// For debugging
func ReadFile(path string) ([]byte, error) {
	// fmt.Println(">>> reading file", path)
	return os.ReadFile(path)
}

func Output(target string, originator string, data []byte, perm fs.FileMode) OutputFile {
	return OutputFile{
		target:     target,
		originator: originator,
		data:       data,
		perm:       perm,
	}
}

func (o *OutputFile) Target() string {
	return o.target
}

func (o *OutputFile) Originator() string {
	return o.originator
}

func (o *OutputFile) Data() []byte {
	return o.data
}

func (o *OutputFile) Perm() fs.FileMode {
	if o.perm == fs.FileMode(0) {
		return fs.ModePerm
	}
	return o.perm
}

// WriteOutSlice blocks and writes concurrently from writes to their output locations.
func WriteOutSlice(writes []OutputFile, concurrent int) error {
	if concurrent == 0 {
		concurrent = 1
	}

	wg := new(sync.WaitGroup)
	errs := make(chan errorWrite)
	guard := make(chan struct{}, concurrent)

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
				errs <- errorWrite{
					err:        err,
					target:     w.target,
					originator: w.originator,
				}
				return
			}
			err = os.WriteFile(w.target, w.data, w.Perm())
			if err != nil {
				errs <- errorWrite{
					err:        err,
					target:     w.target,
					originator: w.originator,
				}
				return
			}

			Fprintln(os.Stdout, w.target)
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

func newHeaders(defaultHeader string) headers {
	return headers{
		perDir: newPerDir(header{
			Buffer:    bytes.NewBufferString(defaultHeader),
			titleFrom: TitleFromH1,
		}),
	}
}

func newFooters(defaultFooter string) footers {
	return footers{
		perDir: newPerDir(bytes.NewBufferString(defaultFooter)),
	}
}

func newPerDir[T any](defaultValue T) perDir[T] {
	return perDir[T]{
		defaultValue: defaultValue,
		values:       make(map[string]T),
	}
}

func (p *perDir[T]) add(path string, v T) error {
	_, ok := p.values[path]
	if ok {
		return fmt.Errorf("found duplicate path '%s'", path)
	}

	p.values[path] = v
	return nil
}

func (p *perDir[T]) choose(path string) T {
	return choose(path, p.defaultValue, p.values)
}

// choose chooses which map value should be used for the given path.
func choose[T any](path string, valueDefault T, m map[string]T) T {
	chosen, ok := m[path]
	if ok {
		return chosen
	}
	parts := strings.Split(path, "/")
	chosen, max := valueDefault, 0

outer:
	for prefix, stored := range m {
		prefixes := strings.Split(prefix, "/")
		for i := range parts {
			if i >= len(prefixes) {
				break
			}
			if parts[i] != prefixes[i] {
				continue outer
			}
		}

		l := len(prefix)
		if max > l {
			continue
		}

		chosen, max = stored, l
	}

	return chosen
}

type errorWrite struct {
	err        error
	target     string
	originator string
}

func (e errorWrite) Error() string {
	return fmt.Errorf("WriteError(target='%s',originator='%s'): %w", e.target, e.originator, e.err).Error()
}
