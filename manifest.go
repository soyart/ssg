package ssg

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Manifest map[string]Site

type Site struct {
	Copies   map[string]string `json:"copies"`
	Links    map[string]string `json:"links"`
	Replaces map[string]string `json:"replaces"`

	logger *slog.Logger `json:"-"`

	Ssg `json:""`
}

type stage int

const (
	stageLink = iota << 1
)

type soywebError struct {
	err   error
	msg   string
	stage stage
}

func (s stage) String() string {
	switch s {
	case stageLink:
		return "stage-link"
	}

	return "bad stage"
}

func Main() {
	manifestPath := "./manifest.json"
	loglevel.Set(slog.LevelDebug)

	logger := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: false,
			Level:     loglevel,
		})).
		With("manifest", manifestPath)

	slog.SetDefault(logger)
	slog.Info("starting")

	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		logger.Error("failed to parse manifest")
		panic(err)
	}

	for key, site := range manifest {
		loggerSite := logger.With("key", key, "url", site.Url)
		site.logger = loggerSite

		err := site.Link()
		if err != nil {
			loggerSite.Error("failed to link", "error", err)
		}
	}
}

var loglevel = new(slog.LevelVar)

func ParseManifest(filename string) (Manifest, error) {
	slog.Info("parsing manifest")

	b, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("failed to read manifest file")
		return Manifest{}, fmt.Errorf("failed to read manifest from file '%s': %w", filename, err)
	}

	var m Manifest
	err = json.Unmarshal(b, &m)
	if err != nil {
		slog.Error("failed to unmarshal json")
		return Manifest{}, fmt.Errorf("failed to parse JSON from file '%s': %w", filename, err)
	}

	return m, nil
}

func (s *Site) Link() error {
	logger := s.logger.WithGroup("link")

	dups := make(setStr)
	for lsrc, ldst := range s.Links {
		logger.With("src", lsrc, "dst", ldst, "phase", "scan")

		if len(lsrc) == 0 {
			return fmt.Errorf("found empty link src")
		}
		if len(ldst) == 0 {
			return fmt.Errorf("found empty link dst")
		}

		if dups.insert(ldst) {
			return fmt.Errorf("duplicate link destination '%s'", ldst)
		}

		ssrc, err := os.Stat(lsrc)
		if err != nil {
			logger.Error("failed to stat src", "error", err)
			return fmt.Errorf("[link] failed to stat src '%s': %w", lsrc, err)
		}

		if ssrc.Mode()&os.ModeSymlink != 0 {
			logger.Error("file is not synlink", "mode", ssrc.Mode())
			return fmt.Errorf("[link] expecting normal file, found link src symlink '%s'", lsrc)
		}

		if ssrc.IsDir() {
			logger.Error("file is dir")
			return fmt.Errorf("[link] expecting normal file, found link src directory '%s'", lsrc)
		}

		sdst, err := os.Stat(ldst)
		if err == nil {
			if sdst.IsDir() {
				logger.Debug("dst is dir")
				return fmt.Errorf("[link] dst is dir")
			}
		}

		if err != nil {
			if !os.IsNotExist(err) {
				logger.Error("failed to stat dst", "error", err)
				return fmt.Errorf("failed to stat link dst '%s': %w", ldst, err)
			}

			err = os.MkdirAll(filepath.Dir(ldst), os.ModePerm)
			if err != nil {
				return fmt.Errorf("fail to prepare dst '%s': %w", ldst, err)
			}
		}
	}

	for lsrc, ldst := range s.Links {
		logger := logger.With("src", lsrc, "dst", lsrc, "phase", "symlink")
		err := os.Symlink(lsrc, ldst)
		if err != nil {
			logger.Error("fail to do symlink")
			return fmt.Errorf("[link] fail to link '%s'->'%s': %w", lsrc, ldst, err)
		}

		logger.Info("ok")
	}

	return nil
}
