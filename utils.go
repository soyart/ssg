package ssg

import (
	"bytes"
	"fmt"
	"strings"
)

type (
	// perDir tracks files under directory in a trie-like fashion.
	perDir struct {
		valueDefault *bytes.Buffer
		values       map[string]*bytes.Buffer
	}

	setStr map[string]struct{}
)

// choose chooses which byte buffer should be used for the given path.
func (p *perDir) choose(path string) *bytes.Buffer {
	buf := p.values[path]
	if buf != nil {
		return buf
	}

	buf, max := p.valueDefault, 0
	for prefix, v := range p.values {
		l := len(prefix)
		if max > l {
			continue
		}
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		buf, max = v, l
	}

	return buf
}

func (p *perDir) add(path string, buf *bytes.Buffer) error {
	_, ok := p.values[path]
	if ok {
		return fmt.Errorf("found duplicate path '%s'", path)
	}

	p.values[path] = buf
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
