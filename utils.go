package ssg

import (
	"bytes"
	"fmt"
	"strings"
)

type (
	from int

	// perDir tracks files under directory in a trie-like fashion.
	perDir[T any] struct {
		d      T
		values map[string]T
	}

	headers struct {
		perDir[header]
	}

	footers struct {
		perDir[*bytes.Buffer]
	}

	header struct {
		*bytes.Buffer
		titleFrom from
	}

	setStr map[string]struct{}
)

const (
	fromNone = 0
	fromH1   = 1 << iota
	fromTag
)

func (h *headers) choose(path string) header {
	return choose(path, h.d, h.values)
}

func (f *footers) choose(path string) *bytes.Buffer {
	return choose(path, f.d, f.values)
}

// choose chooses which map value should be used for the given path.
func choose[T any](path string, valueDefault T, m map[string]T) T {
	chosen, ok := m[path]
	if ok {
		return chosen
	}

	chosen, max := valueDefault, 0
	for prefix, v := range m {
		l := len(prefix)
		if max > l {
			continue
		}
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		chosen, max = v, l
	}

	return chosen

}

func (p *perDir[T]) add(path string, v T) error {
	_, ok := p.values[path]
	if ok {
		return fmt.Errorf("found duplicate path '%s'", path)
	}

	p.values[path] = v
	return nil
}

func (s setStr) insert(v string) bool {
	_, ok := s[v]
	s[v] = struct{}{}

	return ok
}

func (s setStr) contains(v string) bool {
	_, ok := s[v]
	return ok
}
