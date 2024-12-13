package ssg

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Code here is deprecated but kept as handy reference

// deprecated
func (s *Ssg) build() ([]OutputFile, error) {
	// build walks the src directory, and converts Markdown into HTML,
	// returning the results as []write.
	//
	// build also caches the result in s for [WriteOut] later.

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

// deprecated
func (s *Ssg) walkScan(path string, d fs.DirEntry, err error) error {
	// walkScan scans the source directory for header and footer files,
	// and anything required to build a page.

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

// deprecated
func (s *Ssg) walkBuild(path string, d fs.DirEntry, err error) error {
	// walkBuild finds and converts Markdown files to HTML,
	// and assembles it with header and footer.

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
