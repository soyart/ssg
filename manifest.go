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
	Name  string            `json:"name"`
	Url   string            `json:"url"`
	Src   string            `json:"src"`
	Dst   string            `json:"dst"`
	Links map[string]string `json:"links"`

	logger Logger `json:"-"`
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

func NewSite(m Manifest, site string) Site {
	return Site{
		logger: SiteLogger(site, os.Stdout, os.Stdin),
	}
}

func (s *Site) Link() error {
	for src, dst := range s.Links {
		src = fmt.Sprintf("%s/%s", s.Src, src)
		dst = fmt.Sprintf("%s/%s", s.Src, dst)

		statSrc, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to stat source '%s': %w", src, err)
		}

		statDst, err := os.Stat(dst)
		if err != nil {
			return fmt.Errorf("failed to stat destination '%s': %w", src, err)
		}

		link(statSrc, statDst)
	}

	return nil
}

func Link(src, dst string) error {
	statSrc, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source '%s': %w", src, err)
	}

	statDst, err := os.Stat(dst)
	if err != nil {
		return fmt.Errorf("failed to stat destination '%s': %w", src, err)
	}

	return link(statSrc, statDst)
}

func link(src, dst os.FileInfo) error {
	if src.IsDir() {
		entries, err := os.ReadDir(src.Name())
		if err != nil {
			return fmt.Errorf("failed to read directory for recursive linking: %w", err)
		}

		for i := range entries {
			e := entries[i]
			stat, err := os.Stat(e.Name())
			if err != nil {
				return err
			}

			err = link(stat, dst)
			if err != nil {
				return err
			}
		}

		return nil
	}

	fmt.Println(src.Name(), dst.Name())
	return nil
}

func (m *Manifest) HeaderFile() string {
	return ""
}
