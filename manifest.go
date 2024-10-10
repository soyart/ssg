package ssg

import (
	"encoding/json"
	"fmt"
	"os"
)

type Manifest struct {
	Sites map[string]Site
}

type Site struct {
	Name      string            `json:"name"`
	Url       string            `json:"url"`
	Src       string            `json:"src"`
	Dst       string            `json:"dst"`
	Links     map[string]string `json:"links"`
	SsgIgnore []string          `json:"ssgignore"`
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
	for _, dst := range s.Links {
		if dups.insert(dst) {
			return fmt.Errorf("duplicate link destination '%s'", dst)
		}
	}

	for src, dst := range s.Links {
		statSrc, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to stat link src '%s': %w", src, err)
		}

		statDst, err := os.Stat(dst)
		if os.IsNotExist(err) {
			err = os.MkdirAll(dst, os.ModePerm)
		}

		if err != nil {
			return fmt.Errorf("fail to prepare dst '%s': %w", dst, err)
		}

		_, _ = statSrc, statDst
	}

	return nil
}

func (m *Manifest) HeaderFile() string {
	return ""
}
