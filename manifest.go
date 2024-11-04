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
	Copies map[string]WriteTarget `json:"-"`
	Ssg
	CleanUp bool `json:"cleanup"`
}

type WriteTarget struct {
	Target string `json:"target"`
	Force  bool   `json:"force"`
}

type Stage int

type manifestError struct {
	err   error
	key   string
	msg   string
	stage Stage
}

const (
	StageCollect Stage = -1
	// These stages can be skipped
	StageCleanUp Stage = 1 << iota
	StageCopy
	StageBuild

	StagesAll = StageCollect | StageCleanUp | StageCopy | StageBuild
)

var loglevel = new(slog.LevelVar)

func ApplyManifest(path string, do Stage) error {
	logger := newLogger().With("manifest", path)
	slog.SetDefault(logger)
	slog.Info("parsing manifest")

	m, err := NewManifest(path)
	if err != nil {
		logger.Error("failed to parse manifest", "error", err)
		return err
	}

	return Apply(m, do)
}

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

func (s manifestError) Error() string {
	if s.err == nil {
		return fmt.Sprintf("[%s %s] %s", s.stage, s.key, s.msg)
	}

	return fmt.Errorf("[%s %s] %s: %w", s.stage, s.key, s.msg, s.err).Error()
}

func (s manifestError) Unwrap() error {
	return s.err
}

func (s *Site) UnmarshalJSON(b []byte) error {
	var tmp struct {
		Copies map[string]interface{} `json:"copies"`
		Ssg
		CleanUp bool `json:"cleanup"`
	}

	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	copies := make(map[string]WriteTarget)
	err = decodeTargetsForce(tmp.Copies, copies)
	if err != nil {
		return err
	}

	*s = Site{
		CleanUp: tmp.CleanUp,
		Copies:  copies,
		Ssg:     New(tmp.Src, tmp.Dst, tmp.Title, tmp.Url),
	}

	return nil
}

func (t WriteTarget) String() string {
	if t.Force {
		return fmt.Sprintf("%s (force)", t.Target)
	}

	return t.Target
}

func (s Stage) String() string {
	switch s {
	case StageCollect:
		return "collect"

	case StageCleanUp:
		return "cleanup"

	case StageCopy:
		return "copy"

	case StageBuild:
		return "build"
	}

	return "BAD_STAGE"
}

func newLogger() *slog.Logger {
	loglevel.Set(slog.LevelDebug)
	logger := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: true,
			Level:     loglevel,
		}))

	return logger
}

func collect(m Manifest) (map[string]setStr, error) {
	// Collect and detect duplicate write dups
	dups := make(setStr)
	targets := make(map[string]setStr)
	for key, site := range m {
		if targets[key] == nil {
			targets[key] = make(setStr)
		}

		logger := slog.Default().WithGroup("collect").With("key", key, "url", site.Url)
		for src, dst := range site.Copies {
			if !dups.insert(dst.Target) {
				if targets[key].insert(dst.Target) {
					panic("unexpected duplicate")
				}

				continue
			}

			logger.Error("duplicate write target", "src", src, "target", dst.Target)
			return nil, manifestError{
				err:   nil,
				key:   key,
				msg:   "duplicate write target",
				stage: StageCollect,
			}
		}
	}

	return targets, nil
}

func cleanup(m Manifest, targets map[string]setStr) error {
	// Cleanup
	for key, site := range m {
		if !site.CleanUp {
			continue
		}

		logger := slog.Default().WithGroup("cleanup").With("key", key, "url", site.Url)
		siteTargets := targets[key]
		for target := range siteTargets {
			logger.Info("cleaning up", "target", target)
			err := os.RemoveAll(target)
			if err != nil && os.IsNotExist(err) {
				continue
			}
			if err == nil {
				continue
			}

			return manifestError{
				err:   err,
				key:   key,
				msg:   "failed to cleanup",
				stage: StageCleanUp,
			}
		}
	}

	return nil
}

func Apply(m Manifest, do Stage) error {
	slog.Info("skip",
		StageCleanUp.String(), willDo(do, StageCleanUp),
		StageCopy.String(), willDo(do, StageCopy),
		StageBuild.String(), willDo(do, StageBuild),
	)

	targets, err := collect(m)
	if err != nil {
		return err
	}

	if willDo(do, StageCleanUp) {
		err = cleanup(m, targets)
		if err != nil {
			return err
		}
	}

	// Copy
	old := slog.Default()
	for key, site := range m {
		if !willDo(do, StageCopy) {
			old.Info("skipping stage copy")
			break
		}

		slog.SetDefault(old.
			WithGroup("copy").
			With("key", key, "url", site.Url),
		)

		if err := site.Copy(); err != nil {
			return manifestError{
				err:   err,
				key:   key,
				msg:   "failed to copy",
				stage: StageCopy,
			}
		}

		slog.SetDefault(old)
	}

	// Build
	for key, site := range m {
		if !willDo(do, StageBuild) {
			old.Info("skipping stage build")
			break
		}

		old.
			WithGroup("build").
			With("key", key, "url", site.Url).
			Info("building site")

		err := site.Ssg.Generate()
		if err != nil {
			return manifestError{
				err:   err,
				key:   key,
				msg:   "failed to build",
				stage: StageBuild,
			}
		}
	}

	return nil
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
	logger := slog.Default()

	dirs := make(setStr)
	for cpSrc, cpDst := range s.Copies {
		logger := logger.With("phase", "scan", "cpSrc", cpSrc, "cpDst", cpDst)

		if len(cpSrc) == 0 {
			return fmt.Errorf("found empty copy src")
		}
		if len(cpDst.Target) == 0 {
			return fmt.Errorf("found empty copy dst")
		}

		ssrc, err := os.Stat(cpSrc)
		if err != nil {
			logger.Error("failed to stat copy src")
			return fmt.Errorf("failed to stat copy src: '%s'", cpSrc)
		}

		if fileIs(ssrc, os.ModeSymlink) {
			logger.Error("copy src is symlink")
			return fmt.Errorf("copy src is symlink: '%s'", cpSrc)
		}

		sdst, err := os.Stat(cpDst.Target)
		if err != nil {
			if !os.IsNotExist(err) {
				logger.Error("failed to stat copy dst", "error", err)
				return fmt.Errorf("failed to stat copy dst '%s': %w", cpDst, err)
			}

			err = os.MkdirAll(filepath.Dir(cpDst.Target), os.ModePerm)
			if err != nil {
				return fmt.Errorf("fail to prepare copy dst '%s': %w", cpDst, err)
			}
		}

		if ssrc != nil && ssrc.IsDir() {
			dirs.insert(cpSrc)
		}
		if sdst != nil && sdst.IsDir() {
			dirs.insert(cpDst.Target)
		}
	}

	for cpSrc, cpDst := range s.Copies {
		logger := logger.With("phase", "copy", "cpSrc", cpSrc, "cpDst", cpDst)

		err := copyFiles(dirs, cpSrc, cpDst)
		if err != nil {
			logger.Error("failed to copy file")
			return fmt.Errorf("failed to copy directory '%s'->'%s': %w", cpSrc, cpDst.Target, err)
		}
	}

	return nil
}

func cp(src string, dst WriteTarget) error {
	if dst.Force {
		err := os.RemoveAll(dst.Target)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("error reading src: %w", err)
	}

	dir := filepath.Dir(dst.Target)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error preparing dst directory at '%s': %w", dir, err)
	}

	err = os.WriteFile(dst.Target, b, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error writing to dst: %w", err)
	}

	return nil
}

func cpRecurse(src string, dst WriteTarget) error {
	dstRoot := dst.Target
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dstRoot, rel)

		slog.Debug("cpRecurse", "src", src, "base", filepath.Base(path), "dst", dst, "target", target)
		return cp(path, WriteTarget{
			Target: target,
			Force:  dst.Force,
		})
	})
	if err != nil {
		return fmt.Errorf("walkDir failed for src '%s', dst '%s': %w", src, dst.Target, err)
	}

	return nil
}

func copyFiles(dirs setStr, src string, dst WriteTarget) error {
	switch {
	// Copy dir to dir, with target not yet existing
	case dirs.contains(src) && !dirs.contains(dst.Target):
		err := os.MkdirAll(dst.Target, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to prepare dst directory: %w", err)
		}

		fallthrough

	// Copy dir to dir, with target dir existing
	case dirs.contains(src, dst.Target):
		return cpRecurse(src, dst)

	// Copy file to dir, i.e. cp foo.json ./some-dir/
	// which will just writes out to ./some-dir/foo.json
	case dirs.contains(dst.Target):
		base := filepath.Base(src)
		dst.Target = filepath.Join(dst.Target, base)
	}

	return cp(src, dst)
}

func willDo(b Stage, mask Stage) bool {
	return b&mask != 0
}
