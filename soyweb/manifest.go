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

	"github.com/soyart/ssg/ssg-go"
)

var loglevel = new(slog.LevelVar)

type Stage int

const (
	StageCollect Stage = -1

	// These stages can be skipped
	StageCleanUp Stage = 1 << iota
	StageCopy
	StageBuild

	StageAll Stage = StageCollect | StageCleanUp | StageCopy | StageBuild
)

type Manifest map[string]Site

type Site struct {
	ssg ssg.Ssg `json:"-"`

	CleanUp           bool                   `json:"-"` // Remove files in Copies before copying them
	Copies            map[string]CopyTargets `json:"-"`
	GenerateIndex     bool                   `json:"-"`
	GenerateIndexMode IndexGeneratorMode     `json:"-"`
	Replaces          Replaces               `json:"-"`
}

type CopyTarget struct {
	Target string `json:"-"`
	Force  bool   `json:"-"`
}

type CopyTargets []CopyTarget

type ReplaceTarget struct {
	Text  string `json:"-"`
	Count uint   `json:"-"` // 0 replaces all, 1 replaces once, 2 replaces twice, and so on
}

type Replaces map[string]ReplaceTarget

func (s *Site) Src() string { return s.ssg.Src }
func (s *Site) Dst() string { return s.ssg.Dst }

func (s *Site) UnmarshalJSON(b []byte) error {
	var site struct {
		Src   string `json:"src"`
		Dst   string `json:"dst"`
		Title string `json:"title"`
		Url   string `json:"url"`

		Copies            map[string]CopyTargets `json:"copies"`
		CleanUp           bool                   `json:"cleanup"`
		GenerateIndex     bool                   `json:"generate-index"`
		GenerateIndexMode IndexGeneratorMode     `json:"generate-index-mode"`
		Replaces          Replaces               `json:"replaces"`
	}

	err := json.Unmarshal(b, &site)
	if err != nil {
		return err
	}

	*s = Site{
		Copies:            site.Copies,
		Replaces:          site.Replaces,
		CleanUp:           site.CleanUp,
		GenerateIndex:     site.GenerateIndex,
		GenerateIndexMode: site.GenerateIndexMode,
		ssg: ssg.New(
			site.Src,
			site.Dst,
			site.Title,
			site.Url,
		),
	}
	return nil
}

func (c *CopyTargets) UnmarshalJSON(b []byte) error {
	var data any
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	result, err := parseCopyTarget(data)
	if err != nil {
		return err
	}

	*c = result
	return nil
}

func (r *Replaces) UnmarshalJSON(b []byte) error {
	var data map[string]any
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	results := make(Replaces)
	for k, v := range data {
		result, err := decodeReplace(v)
		if err != nil {
			return err
		}
		results[k] = result
	}

	*r = results
	return nil
}

func decodeReplace(data any) (ReplaceTarget, error) {
	switch data := data.(type) {
	case string:
		return ReplaceTarget{Text: data}, nil

	case map[string]any:
		textRaw, ok := data["text"]
		if !ok {
			return ReplaceTarget{}, errors.New("missing field 'text'")
		}
		text, ok := textRaw.(string)
		if !ok {
			return ReplaceTarget{}, fmt.Errorf("unexpected type for field 'text': %s'", reflect.TypeOf(textRaw).String())
		}
		countRaw, ok := data["count"]
		if !ok {
			return ReplaceTarget{}, errors.New("missing field 'text'")
		}
		countFloat, ok := countRaw.(float64)
		if !ok {
			return ReplaceTarget{}, fmt.Errorf("unexpected type for field 'count': '%s'", reflect.TypeOf(countRaw).String())
		}
		if countFloat < 0 {
			return ReplaceTarget{}, fmt.Errorf("bad replace count %f", countFloat)
		}
		return ReplaceTarget{
			Text:  text,
			Count: uint(countFloat),
		}, nil
	}

	return ReplaceTarget{}, fmt.Errorf("bad entry data shape of type %s: '%+v'", reflect.TypeOf(data).String(), data)
}

func parseCopyTarget(data any) ([]CopyTarget, error) {
	switch data := data.(type) {
	case string:
		return []CopyTarget{{Target: data}}, nil

	case []any:
		results := []CopyTarget{}
		for i := range data {
			result, err := parseCopyTarget(data[i])
			if err != nil {
				return nil, err
			}
			results = append(results, result...)
		}
		return results, nil

	case map[string]any:
		targetRaw, ok := data["target"]
		if !ok {
			return nil, errors.New("missing key 'target'")
		}
		target, ok := targetRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid data type for field 'target', expecting string, got '%s'", reflect.TypeOf(targetRaw).String())
		}

		w := CopyTarget{Target: target}

		forceRaw, ok := data["force"]
		if !ok {
			return []CopyTarget{w}, nil
		}
		force, ok := forceRaw.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid data type for field 'target', expecting bool, got '%s'", reflect.TypeOf(forceRaw).String())
		}

		w.Force = force
		return []CopyTarget{w}, nil
	}

	return nil, fmt.Errorf("bad entry data shape of type %s: '%+v'", reflect.TypeOf(data).String(), data)
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
		if len(site.Replaces) != 0 {
			hookReplacer := HookReplacer(site.Replaces)
			opts = append(opts, ssg.PrependHooks(hookReplacer))
		}
		// TODO: refactor init options for manifest at 1 place!
		if site.GenerateIndex {
			opts = append(opts, ssg.WithPipelines(NewIndexGenerator(IndexGeneratorModeDefault)))
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

func (t CopyTarget) String() string {
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
	return slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: true,
			Level:     loglevel,
		}),
	)
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
		for src, dests := range site.Copies {
			for i := range dests {
				dst := dests[i]
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
	}
	return targets, nil
}

func cleanup(m Manifest, targets map[string]ssg.Set) error {
	for key, site := range m {
		if !site.CleanUp {
			continue
		}
		logger := slog.Default().
			WithGroup("cleanup").
			With("key", key, "url", site.ssg.Url)

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

func (s *Site) Copy() error {
	logger := slog.Default()
	dirs := make(ssg.Set)
	perms := make(map[string]fs.FileMode)

	for cpSrc, cpDsts := range s.Copies {
		for i := range cpDsts {
			cpDst := &cpDsts[i]
			logger := logger.
				With("phase", "scan", "cpSrc", cpSrc, "cpDst", cpDst)

			if len(cpSrc) == 0 {
				return fmt.Errorf("found empty copy src")
			}
			if len(cpDst.Target) == 0 {
				return fmt.Errorf("found empty copy dst")
			}

			ssrc, err := os.Stat(cpSrc)
			if err != nil {
				logger.Error("failed to stat copy src")
				return fmt.Errorf("failed to stat copy src '%s': %w", cpSrc, err)
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
	}

	for cpSrc, cpDsts := range s.Copies {
		for _, cpDst := range cpDsts {
			logger := logger.With("phase", "copy", "cpSrc", cpSrc, "cpDst", cpDst)
			err := copyFiles(dirs, cpSrc, cpDst, perms)
			if err != nil {
				logger.Error("failed to copy file")
				return fmt.Errorf("failed to copy directory '%s'->'%s': %w", cpSrc, cpDst.Target, err)
			}
		}
	}

	return nil
}

func cp(src string, dst CopyTarget, perm fs.FileMode) error {
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

func cpRecurse(src string, dst CopyTarget) error {
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
		// slog.Debug("cpRecurse", "src", src, "base", filepath.Base(path), "dst", dst, "target", target)

		out := CopyTarget{
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

func copyFiles(dirs ssg.Set, src string, dst CopyTarget, permsCache map[string]fs.FileMode) error {
	switch {
	// Copy dir to dir, with target not yet existing
	case dirs.Contains(src) && !dirs.Contains(dst.Target):
		err := os.MkdirAll(dst.Target, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to prepare dst directory: %w", err)
		}

		fallthrough

	// Copy dir to dir, with target dir existing
	case dirs.Contains(src, dst.Target):
		return cpRecurse(src, dst)

	// Copy file to dir, i.e. cp foo.json ./some-dir/
	// which will just writes out to ./some-dir/foo.json
	case dirs.Contains(dst.Target):
		base := filepath.Base(src)
		dst.Target = filepath.Join(dst.Target, base)
	}

	return cp(src, dst, permsCache[src])
}
