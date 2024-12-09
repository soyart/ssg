package soyweb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/soyart/ssg"
)

var loglevel = new(slog.LevelVar)

type Stage int

const (
	StageCollect Stage = -1

	// These stages can be skipped
	StageCleanUp Stage = 1 << iota
	StageCopy
	StageBuild

	StagesAll = StageCollect | StageCleanUp | StageCopy | StageBuild
)

type Manifest map[string]Site

type Site struct {
	ssg          ssg.Ssg                `json:"-"`
	Copies       map[string]WriteTarget `json:"-"`
	CleanUp      bool                   `json:"-"`
	GenerateBlog bool                   `json:"-"`
}

type WriteTarget struct {
	Target string `json:"target"`
	Force  bool   `json:"force"`
}

type manifestError struct {
	err   error
	key   string
	msg   string
	stage Stage
}

func (s *Stage) Skip(targets ...Stage) {
	copied := *s
	for i := range targets {
		copied &^= targets[i]
	}

	*s = copied
}

func (s *Stage) Ok(targets ...Stage) bool {
	copied := *s
	for i := range targets {
		if copied&targets[i] == 0 {
			return false
		}
	}

	return true
}

// ApplyManifest loops through all sites and apply manifest stages
// described in do. It applies opts to each site's [Ssg] before
// the call to [Ssg.Generate].
func ApplyManifest(m Manifest, stages Stage, opts ...ssg.Option) error {
	slog.Info("stages",
		StageCleanUp.String(), stages.Ok(StageCleanUp),
		StageCopy.String(), stages.Ok(StageCopy),
		StageBuild.String(), stages.Ok(StageBuild),
	)

	targets, err := collect(m)
	if err != nil {
		return err
	}
	if stages.Ok(StageCleanUp) {
		err = cleanup(m, targets)
		if err != nil {
			return err
		}
	}

	// Copy
	old := slog.Default()
	for key, site := range m {
		if !stages.Ok(StageCopy) {
			old.Info("skipping stage copy")
			break
		}

		slog.SetDefault(old.
			WithGroup("copy").
			With("key", key, "url", site.ssg.Url),
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
		if !stages.Ok(StageBuild) {
			old.Info("skipping stage build")
			break
		}

		old.
			WithGroup("build").
			With(
				"key", key,
				"url", site.ssg.Url,
			).
			Info("building site")

		s := &site.ssg
		if site.GenerateBlog {
			gen := IndexGenerator(s.Src, s.ImplDefault())
			opts = append(opts, ssg.WithImpl(gen))
		}

		s.With(opts...)
		if err := s.Generate(); err != nil {
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

func ApplyFromManifest(path string, do Stage, opts ...ssg.Option) error {
	logger := newLogger().With("manifest", path)
	slog.SetDefault(logger)
	slog.Info("parsing manifest")

	m, err := NewManifest(path)
	if err != nil {
		logger.Error("failed to parse manifest", "error", err)
		return err
	}

	return ApplyManifest(m, do, opts...)
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
	var site struct {
		Copies map[string]interface{} `json:"copies"`

		Src          string `json:"src"`
		Dst          string `json:"dst"`
		Title        string `json:"title"`
		Url          string `json:"url"`
		CleanUp      bool   `json:"cleanup"`
		GenerateBlog bool   `json:"generate_blog"`
	}

	err := json.Unmarshal(b, &site)
	if err != nil {
		return err
	}

	copies := make(map[string]WriteTarget)
	err = decodeTargetsForce(site.Copies, copies)
	if err != nil {
		return err
	}

	*s = Site{
		Copies:       copies,
		CleanUp:      site.CleanUp,
		GenerateBlog: site.GenerateBlog,
		ssg: ssg.NewWithOptions(
			site.Src,
			site.Dst,
			site.Title,
			site.Url,
		),
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

func collect(m Manifest) (map[string]ssg.Set, error) {
	// Collect and detect duplicate write dups
	dups := make(ssg.Set)
	targets := make(map[string]ssg.Set)
	for key, site := range m {
		if targets[key] == nil {
			targets[key] = make(ssg.Set)
		}

		logger := slog.Default().WithGroup("collect").With("key", key, "url", site.ssg.Url)
		for src, dst := range site.Copies {
			if !dups.Insert(dst.Target) {
				if targets[key].Insert(dst.Target) {
					panic("unexpected duplicate")
				}

				continue
			}

			logger.Error("duplicate write target", "src", src, "target", dst.Target)
			return nil, manifestError{
				err:   fmt.Errorf("duplicate target '%s'", dst.Target),
				key:   key,
				msg:   "duplicate write target",
				stage: StageCollect,
			}
		}
	}

	return targets, nil
}

func cleanup(m Manifest, targets map[string]ssg.Set) error {
	// Cleanup
	for key, site := range m {
		if !site.CleanUp {
			continue
		}

		logger := slog.Default().WithGroup("cleanup").With("key", key, "url", site.ssg.Url)
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

		w := WriteTarget{Target: target}

		forceRaw, ok := data["force"]
		if !ok {
			return w, nil
		}
		force, ok := forceRaw.(bool)
		if !ok {
			return WriteTarget{}, fmt.Errorf("invalid data type for field 'target', expecting bool, got '%s'", reflect.TypeOf(forceRaw).String())
		}

		w.Force = force
		return w, nil
	}

	return WriteTarget{}, fmt.Errorf("bad entry data shape: '%v'", entry)
}

func (s *Site) Copy() error {
	logger := slog.Default()

	dirs := make(ssg.Set)
	perms := make(map[string]fs.FileMode)
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
		if ssg.FileIs(ssrc, os.ModeSymlink) {
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
			dirs.Insert(cpSrc)
			perms[cpSrc] = ssrc.Mode().Perm()
		}
		if sdst != nil && sdst.IsDir() {
			dirs.Insert(cpDst.Target)
			perms[cpDst.Target] = sdst.Mode().Perm()
		}
	}

	for cpSrc, cpDst := range s.Copies {
		logger := logger.With("phase", "copy", "cpSrc", cpSrc, "cpDst", cpDst)

		err := copyFiles(dirs, cpSrc, cpDst, perms)
		if err != nil {
			logger.Error("failed to copy file")
			return fmt.Errorf("failed to copy directory '%s'->'%s': %w", cpSrc, cpDst.Target, err)
		}
	}

	return nil
}

func cp(src string, dst WriteTarget, perm fs.FileMode) error {
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

	if perm.Perm() == 0 {
		stat, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("error stating cp src file: %w", err)
		}
		perm = stat.Mode().Perm()
	}
	err = os.WriteFile(dst.Target, b, perm)
	if err != nil {
		return fmt.Errorf("error writing to dst: %w", err)
	}

	return nil
}

func cpRecurse(src string, dst WriteTarget) error {
	dstRoot := dst.Target
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dstRoot, rel)
		slog.Debug("cpRecurse", "src", src, "base", filepath.Base(path), "dst", dst, "target", target)
		out := WriteTarget{
			Target: target,
			Force:  dst.Force,
		}

		return cp(path, out, info.Mode().Perm())
	})
	if err != nil {
		return fmt.Errorf("walkDir failed for src '%s', dst '%s': %w", src, dst.Target, err)
	}

	return nil
}

func copyFiles(dirs ssg.Set, src string, dst WriteTarget, permsCache map[string]fs.FileMode) error {
	switch {
	// Copy dir to dir, with target not yet existing
	case dirs.ContainsAll(src) && !dirs.ContainsAll(dst.Target):
		err := os.MkdirAll(dst.Target, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to prepare dst directory: %w", err)
		}

		fallthrough

	// Copy dir to dir, with target dir existing
	case dirs.ContainsAll(src, dst.Target):
		return cpRecurse(src, dst)

	// Copy file to dir, i.e. cp foo.json ./some-dir/
	// which will just writes out to ./some-dir/foo.json
	case dirs.ContainsAll(dst.Target):
		base := filepath.Base(src)
		dst.Target = filepath.Join(dst.Target, base)
	}

	return cp(src, dst, permsCache[src])
}
