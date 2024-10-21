package ssg

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
)

type Manifest map[string]Site

type Site struct {
	LinksJson  map[string]interface{} `json:"links"`
	Links      map[string]TargetForce `json:"-"`
	CopiesJson map[string]interface{} `json:"copies"`
	Copies     map[string]TargetForce `json:"-"`

	logger *slog.Logger `json:"-"`

	Ssg
}

type TargetForce struct {
	Target string `json:"target"`
	Force  bool   `json:"force"`
}

func (t TargetForce) String() string {
	if t.Force {
		return fmt.Sprintf("%s (force)", t.Target)
	}

	return t.Target
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
		logger.Error("failed to parse manifest", "error", err)
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

	for k, site := range m {
		links := make(map[string]TargetForce)
		err := decodeTargetsForce(site.LinksJson, links)
		if err != nil {
			return Manifest{}, err
		}

		copies := make(map[string]TargetForce)
		err = decodeTargetsForce(site.CopiesJson, copies)
		if err != nil {
			return Manifest{}, err
		}

		site.Links, site.Copies = links, copies

		m[k] = site
	}

	return m, nil
}

func decodeTargetsForce(m map[string]interface{}, target map[string]TargetForce) error {
	for k, entry := range m {
		link, err := decodeTargetForce(entry)
		if err != nil {
			return err
		}

		target[k] = link
	}

	return nil
}

func decodeTargetForce(entry interface{}) (TargetForce, error) {
	switch data := entry.(type) {
	case string:
		return TargetForce{Target: data}, nil

	case map[string]interface{}:
		targetRaw, ok := data["target"]
		if !ok {
			return TargetForce{}, errors.New("missing key 'target'")
		}
		target, ok := targetRaw.(string)
		if !ok {
			return TargetForce{}, fmt.Errorf("invalid data type for field 'target', expecting string, got '%s'", reflect.TypeOf(targetRaw).String())
		}

		l := TargetForce{Target: target}

		forceRaw, ok := data["force"]
		if !ok {
			return l, nil
		}
		force, ok := forceRaw.(bool)
		if !ok {
			return TargetForce{}, fmt.Errorf("invalid data type for field 'target', expecting string, got '%s'", reflect.TypeOf(forceRaw).String())
		}

		l.Force = force
		return l, nil
	}

	return TargetForce{}, fmt.Errorf("bad entry data shape: '%v'", entry)
}

func (s *Site) Link() error {
	logger := s.logger.WithGroup("link")

	dups := make(setStr)
	for lsrc, ldst := range s.Links {
		logger.With("src", lsrc, "dst", ldst, "phase", "scan")

		if len(lsrc) == 0 {
			return fmt.Errorf("found empty link src")
		}
		if len(ldst.Target) == 0 {
			return fmt.Errorf("found empty link dst")
		}

		if dups.insert(ldst.Target) {
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

		sdst, err := os.Stat(ldst.Target)
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

			err = os.MkdirAll(filepath.Dir(ldst.Target), os.ModePerm)
			if err != nil {
				return fmt.Errorf("fail to prepare dst '%s': %w", ldst, err)
			}
		}
	}

	for lsrc, ldst := range s.Links {
		logger := logger.With("src", lsrc, "dst", lsrc, "phase", "symlink")

		err := os.Symlink(lsrc, ldst.Target)
		if err != nil {
			if ldst.Force && os.IsExist(err) {
				err = os.Remove(ldst.Target)
				if err == nil {
					err = os.Symlink(lsrc, ldst.Target)
				}
				if err == nil {
					continue
				}
			}

			logger.Error("fail to make symlink")
			return fmt.Errorf("[link] fail to link: %w", err)
		}

		logger.Info("ok")
	}

	return nil
}
