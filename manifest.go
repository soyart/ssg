package ssg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Manifest struct {
	Sites map[string]Site
}

type Site struct {
	Copies   map[string]string `json:"copies"`
	Links    map[string]string `json:"links"`
	Replaces map[string]string `json:"replaces"`

	Name string `json:"name"`
	Url  string `json:"url"`
	Src  string `json:"src"`
	Dst  string `json:"dst"`

	// Files to ignore with glob
	Ignores []string `json:"ssgignore"`
}

func ParseManifest(filename string) (Manifest, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to read manifest from file '%s': %w", filename, err)
	}

	var m Manifest
	err = json.Unmarshal(b, &m)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to parse JSON from file '%s': %w", filename, err)
	}

	return m, nil
}

func (s *Site) Link() error {
	dups := make(setStr)
	for src, dst := range s.Links {
		if len(src) == 0 {
			return fmt.Errorf("found empty link src")
		}
		if len(dst) == 0 {
			return fmt.Errorf("found empty link dst")
		}
		if dups.insert(dst) {
			return fmt.Errorf("duplicate link destination '%s'", dst)
		}
	}

	for src, dst := range s.Links {
		ssrc, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to stat link src '%s': %w", src, err)
		}

		if ssrc.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("expecting normal file, found link src symlink '%s'", src)
		}

		if ssrc.IsDir() {
			return fmt.Errorf("expecting normal file, found link src directory '%s'", src)
		}

		sdst, err := os.Stat(dst)
		if err == nil {
			if sdst.IsDir() {
			}
		}

		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat link dst '%s': %w", dst, err)
			}

			err = os.MkdirAll(filepath.Dir(dst), os.ModePerm)
			if err != nil {
				return fmt.Errorf("fail to prepare dst '%s': %w", dst, err)
			}
		}

		if err != nil {
			return fmt.Errorf("fail to prepare dst '%s': %w", dst, err)
		}

		err = os.Symlink(src, dst)
		if err != nil {
			return fmt.Errorf("fail to link '%s'->'%s'", src, dst)
		}
	}

	return nil
}
