package ssg

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

type (
	Set map[string]struct{}

	// perDir tracks files under directory in a trie-like fashion.
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

func (s Set) Insert(v string) bool {
	_, ok := s[v]
	s[v] = struct{}{}

	return ok
}

func (s Set) ContainsAll(items ...string) bool {
	for _, v := range items {
		_, ok := s[v]
		if !ok {
			return false
		}
	}

	return true
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
