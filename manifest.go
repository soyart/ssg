package ssg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
)

type Manifest map[string]Site

type Site struct {
	logger *slog.Logger           `json:"-"`
	Links  map[string]WriteTarget `json:"-"`
	Copies map[string]WriteTarget `json:"-"`
	Ssg
	CleanUp bool `json:"cleanup"`
}

type WriteTarget struct {
	Target string `json:"target"`
	Force  bool   `json:"force"`
}

type stage int

type soywebError struct {
	err   error
	msg   string
	stage stage
}

func (s *Site) UnmarshalJSON(b []byte) error {
	var tmp struct {
		Links  map[string]interface{} `json:"links"`
		Copies map[string]interface{} `json:"copies"`
		Ssg
		CleanUp bool `json:"cleanup"`
	}

	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	links := make(map[string]WriteTarget)
	err = decodeTargetsForce(tmp.Links, links)
	if err != nil {
		return err
	}

	copies := make(map[string]WriteTarget)
	err = decodeTargetsForce(tmp.Copies, copies)
	if err != nil {
		return err
	}

	*s = Site{
		Links:   links,
		CleanUp: tmp.CleanUp,
		Copies:  copies,
		Ssg:     tmp.Ssg,
	}

	return nil
}

func (t WriteTarget) String() string {
	if t.Force {
		return fmt.Sprintf("%s (force)", t.Target)
	}

	return t.Target
}

func (s stage) String() string {
	switch s {
	case stageLink:
		return "stage-link"
	}

	return "bad stage"
}

const (
	stageCopy stage = iota << 1
	stageLink
)

func Build(manifestPath string) error {
	loglevel.Set(slog.LevelDebug)

	logger := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: false,
			Level:     loglevel,
		})).
		With("manifest", manifestPath)

	slog.SetDefault(logger)
	slog.Info("parsing manifest")

	m, err := NewManifest(manifestPath)
	if err != nil {
		logger.Error("failed to parse manifest", "error", err)
		return err
	}

	// Collect and detect duplicate write dups
	dups := make(setStr)
	targets := make(map[string]setStr)
	for key, site := range m {
		if targets[key] == nil {
			targets[key] = make(setStr)
		}

		logger := logger.WithGroup("collect").With("key", key, "url", site.Url)
		for src, dst := range site.Copies {
			if !dups.insert(dst.Target) {
				if targets[key].insert(dst.Target) {
					panic("unexpected duplicate")
				}

				continue
			}

			logger.Error("duplicate write target", "src", src, "target", dst.Target)
			return fmt.Errorf("duplicate write target '%s'", dst.Target)
		}

		for src, dst := range site.Links {
			if !dups.insert(dst.Target) {
				if targets[key].insert(dst.Target) {
					panic("unexpected duplicate")
				}

				continue
			}

			logger.Error("duplicate write target", "src", src, "target", dst.Target)
			return fmt.Errorf("duplicate write target '%s'", dst.Target)
		}
	}

	// Cleanup
	for key, site := range m {
		if !site.CleanUp {
			continue
		}

		logger := logger.WithGroup("cleanup").With("key", key, "url", site.Url)
		siteTargets := targets[key]
		for target := range siteTargets {
			logger.Info("cleaning up", "target", target)

			err := os.Remove(target)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("[cleanup] failed to remove '%s': %w", target, err)
			}
		}
	}

	// Copy
	for key, site := range m {
		logger := logger.With("key", key, "url", site.Url)
		site.logger = logger

		err := site.Copy()
		if err != nil {
			return err
		}
	}

	// Link
	for key, site := range m {
		logger := logger.With("key", key, "url", site.Url)
		site.logger = logger

		err := site.Link()
		if err != nil {
			return err
		}
	}

	return nil
}

var loglevel = new(slog.LevelVar)

func NewManifest(filename string) (Manifest, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("failed to read manifest file")
		return Manifest{}, fmt.Errorf("failed to read manifest from file '%s': %w", filename, err)
	}

	m := Manifest{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return Manifest{}, nil
	}

	return m, nil
}
func decodeTargetsForce(m map[string]interface{}, target map[string]WriteTarget) error {
	for k, entry := range m {
		link, err := decodeTargetForce(entry)
		if err != nil {
			return err
		}

		target[k] = link
	}

	return nil
}

func decodeTargetForce(entry interface{}) (WriteTarget, error) {
	switch data := entry.(type) {
	case string:
		return WriteTarget{Target: data}, nil

	case map[string]interface{}:
		targetRaw, ok := data["target"]
		if !ok {
			return WriteTarget{}, errors.New("missing key 'target'")
		}
		target, ok := targetRaw.(string)
		if !ok {
			return WriteTarget{}, fmt.Errorf("invalid data type for field 'target', expecting string, got '%s'", reflect.TypeOf(targetRaw).String())
		}

		l := WriteTarget{Target: target}

		forceRaw, ok := data["force"]
		if !ok {
			return l, nil
		}
		force, ok := forceRaw.(bool)
		if !ok {
			return WriteTarget{}, fmt.Errorf("invalid data type for field 'target', expecting string, got '%s'", reflect.TypeOf(forceRaw).String())
		}

		l.Force = force
		return l, nil
	}

	return WriteTarget{}, fmt.Errorf("bad entry data shape: '%v'", entry)
}

func (s *Site) Copy() error {
	logger := s.logger.WithGroup("copy")

	dirs := make(setStr)
	for cpSrc, cpDst := range s.Copies {
		logger := logger.With("phase", "scan", "cpSrc", cpSrc, "cpDst", cpDst)

		if len(cpSrc) == 0 {
			return fmt.Errorf("[copy] found empty copy src")
		}
		if len(cpDst.Target) == 0 {
			return fmt.Errorf("[copy] found empty copy dst")
		}

		ssrc, err := os.Stat(cpSrc)
		if err != nil {
			logger.Error("failed to stat copy src")
			return fmt.Errorf("[copy] failed to stat copy src: '%s'", cpSrc)
		}

		if fileIs(ssrc, os.ModeSymlink) {
			logger.Error("copy src is symlink")
			return fmt.Errorf("[copy] copy src is symlink: '%s'", cpSrc)
		}

		sdst, err := os.Stat(cpDst.Target)
		if err == nil {
			if sdst.IsDir() {
				logger.Debug("dst is dir")
				return fmt.Errorf("[copy] copy dst is dir")
			}
		}

		if err != nil {
			if !os.IsNotExist(err) {
				logger.Error("failed to stat copy dst", "error", err)
				return fmt.Errorf("[copy] failed to stat copy dst '%s': %w", cpDst.Target, err)
			}

			err = os.MkdirAll(filepath.Dir(cpDst.Target), os.ModePerm)
			if err != nil {
				return fmt.Errorf("[copy] fail to prepare copy dst '%s': %w", cpDst.Target, err)
			}
		}

		if sdst != nil && sdst.IsDir() {
			dirs.insert(cpSrc)
		}
	}

	for cpSrc, cpDst := range s.Copies {
		logger := logger.With("phase", "copy", "cpSrc", cpSrc, "cpDst", cpDst)

		if dirs.contains(cpSrc) {
			err := cpRecurse(cpSrc, cpDst)
			if err != nil {
				logger.Error("failed to copy directory")
				return fmt.Errorf("[copy] failed to copy directory '%s'", cpSrc)
			}

			continue
		}

		err := cp(cpSrc, cpDst)
		if err != nil {
			logger.Error("failed to copy file")
			return fmt.Errorf("[copy] failed to copy file '%s'", cpSrc)
		}
	}

	return nil
}

func (s *Site) Link() error {
	logger := s.logger.WithGroup("link")

	for lnSrc, lnDst := range s.Links {
		logger.With("lnSrc", lnSrc, "lnDst", lnDst, "phase", "scan")

		if len(lnSrc) == 0 {
			return fmt.Errorf("[link] found empty link src")
		}
		if len(lnDst.Target) == 0 {
			return fmt.Errorf("[link] found empty link dst")
		}

		ssrc, err := os.Stat(lnSrc)
		if err != nil {
			logger.Error("failed to stat link src", "error", err)
			return fmt.Errorf("[link] failed to stat src '%s': %w", lnSrc, err)
		}

		if fileIs(ssrc, os.ModeSymlink) {
			logger.Error("file is not synlink", "mode", ssrc.Mode())
			return fmt.Errorf("[link] expecting normal file, found link src symlink '%s'", lnSrc)
		}

		if ssrc.IsDir() {
			logger.Error("file is dir")
			return fmt.Errorf("[link] expecting normal file, found link src directory '%s'", lnSrc)
		}

		sdst, err := os.Stat(lnDst.Target)
		if err == nil {
			if sdst.IsDir() {
				logger.Debug("dst is dir")
				return fmt.Errorf("[link] dst is dir")
			}
		}

		if err != nil {
			if !os.IsNotExist(err) {
				logger.Error("failed to stat link dst", "error", err)
				return fmt.Errorf("failed to stat link dst '%s': %w", lnDst, err)
			}

			err = os.MkdirAll(filepath.Dir(lnDst.Target), os.ModePerm)
			if err != nil {
				return fmt.Errorf("fail to prepare link dst '%s': %w", lnDst, err)
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

func fileIs(f os.FileInfo, mode fs.FileMode) bool {
	return f.Mode()&mode != 0
}

func cp(src string, dst WriteTarget) error {
	if dst.Force {
		err := os.Remove(dst.Target)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst.Target, b, os.ModePerm)
}

func cpRecurse(src string, dst WriteTarget) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		base := filepath.Base(path)
		target := filepath.Join(dst.Target, base)
		dst.Target = target

		return cp(path, dst)
	})
}
