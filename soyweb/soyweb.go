package soyweb

import (
	"errors"
	"log/slog"
	"os"

	"github.com/soyart/ssg/ssg-go"
)

type IndexGeneratorMode string

const (
	MarkerIndex string = "_index.soyweb"

	IndexGeneratorModeDefault IndexGeneratorMode = ""
	IndexGeneratorModeReverse IndexGeneratorMode = "reverse"
	IndexGeneratorModeModTime IndexGeneratorMode = "modtime"
)

var ErrWebFormatNotSupported = errors.New("unsupported web format")

// FlagsV2 represents CLI arguments that could modify soyweb behavior, such as skipping stages
// and minifying content of certain file extensions.
type FlagsV2 struct {
	NoCleanup       bool `arg:"--no-cleanup" help:"Skip cleanup stage"`
	NoCopy          bool `arg:"--no-copy" help:"Skip scopy stage"`
	NoBuild         bool `arg:"--no-build" help:"Skip build stage"`
	NoReplace       bool `arg:"--no-replace" help:"Do not do text replacements defined in manifest"`
	NoGenerateIndex bool `arg:"--no-gen-index" help:"Do not generate indexes on _index.soyweb"`

	MinifyHtmlGenerate bool `arg:"--min-html" help:"Minify converted HTML outputs"`
	MinifyHtmlCopy     bool `arg:"--min-html-copy" help:"Minify all copied HTML"`
	MinifyCss          bool `arg:"--min-css" help:"Minify CSS files"`
	MinifyJs           bool `arg:"--min-js" help:"Minify Javascript files"`
	MinifyJson         bool `arg:"--min-json" help:"Minify JSON files"`
}

func (f FlagsV2) Stage() Stage {
	s := StageAll
	if f.NoCleanup {
		s.Skip(StageCleanUp)
	}
	if f.NoCopy {
		s.Skip(StageCopy)
	}
	if f.NoBuild {
		s.Skip(StageBuild)
	}
	return s
}

func (f FlagsV2) Hooks() []ssg.Hook {
	return filterNilHooks(
		f.hookMinify(),
	)
}

func (f FlagsV2) hookMinify() ssg.Hook {
	m := make(map[string]MinifyFn)
	if f.MinifyHtmlCopy {
		m[".html"] = MinifyHtml
	}
	if f.MinifyCss {
		m[".css"] = MinifyCss
	}
	if f.MinifyJs {
		m[".js"] = MinifyJs
	}
	if f.MinifyJson {
		m[".json"] = MinifyJson
	}
	return HookMinify(m)
}

// builder controls `soyweb build` behavior
// by ordering hooks and pipeline
type builder struct {
	Site
	flags FlagsV2
}

func newManifestBuilder(s Site, f FlagsV2) *builder {
	b := &builder{Site: s, flags: f}
	b.initialize()
	return b
}

func (b *builder) initialize() {
	b.ssg.With(
		ssg.WithHooks(b.Hooks()...),
		ssg.WithHooksGenerate(b.HooksGenerate()...),
		ssg.WithPipelines(b.Pipelines()...),
	)
}

func (b *builder) Hooks() []ssg.Hook {
	if b.flags.NoBuild {
		return nil
	}
	if b.flags.NoReplace {
		return filterNilHooks(
			b.flags.hookMinify(),
		)
	}
	return filterNilHooks(
		HookReplacer(b.Replaces),
		b.flags.hookMinify(),
	)
}

func (b *builder) HooksGenerate() []ssg.HookGenerate {
	if b.flags.MinifyHtmlGenerate {
		return []ssg.HookGenerate{
			MinifyHtml,
		}
	}
	return nil
}

func (b *builder) Pipelines() []any {
	if b.flags.NoGenerateIndex || !b.GenerateIndex {
		return nil
	}
	return []any{
		NewIndexGenerator(b.GenerateIndexMode)(&b.ssg),
	}
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

func ApplyManifestV2(m Manifest, f FlagsV2, do Stage) error {
	slog.SetDefault(newLogger())
	slog.Info("stages",
		StageCleanUp.String(), do.Ok(StageCleanUp),
		StageCopy.String(), do.Ok(StageCopy),
		StageBuild.String(), do.Ok(StageBuild),
	)

	targets, err := collect(m)
	if err != nil {
		return err
	}
	if do.Ok(StageCleanUp) {
		err = cleanup(m, targets)
		if err != nil {
			return err
		}
	}

	// Copy
	old := slog.Default()
	for key, site := range m {
		slog.SetDefault(old.
			WithGroup("copy").
			With(
				"key", key,
				"url", site.ssg.Url,
			),
		)
		if !do.Ok(StageCopy) {
			slog.Info("skipping stage copy")
			break
		}

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
		log := old.
			WithGroup("build").
			With(
				"key", key,
				"url", site.ssg.Url,
			)

		if !do.Ok(StageBuild) {
			log.Info("skipping stage build")
			break
		}

		slog.SetDefault(log)
		b := newManifestBuilder(site, f)

		log.
			With(
				"key", key,
				"url", site.ssg.Url,
				// @TODO: Len logs below will be removed
				// "len_hooks", len(b.Hooks()),
				// "len_hooks_generate", len(b.HooksGenerate()),
				// "len_pipelines", len(b.Pipelines()),
			).
			Info("building site")

		if err := b.ssg.Generate(); err != nil {
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

func filterNilHooks(slice ...ssg.Hook) []ssg.Hook {
	var result []ssg.Hook
	for i := range slice {
		elem := slice[i]
		if elem == nil {
			continue
		}
		result = append(result, elem)
	}
	return result
}
