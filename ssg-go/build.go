package ssg

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

func build(s *Ssg, o Outputs) ([]string, []OutputFile, error) {
	s.result = buildOutput{
		cacheOutput: s.options.caching,
		writer:      o,
	}
	err := filepath.WalkDir(s.Src, s.walk)
	if err != nil {
		return nil, nil, err
	}
	return s.result.files, s.result.cache, nil
}

func (s *Ssg) walk(path string, d fs.DirEntry, err error) error {
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
	s.result.files = append(s.result.files, path)

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
	s.result.Add(output)
	return nil
}
