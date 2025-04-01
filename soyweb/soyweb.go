package soyweb

import (
	"errors"
	"log/slog"

	"github.com/soyart/ssg/ssg-go"
)

type IndexGeneratorMode string

const (
	MarkerIndex string = "_index.soyweb"

	IndexGeneratorModeDefault IndexGeneratorMode = ""
	IndexGeneratorModeReverse IndexGeneratorMode = "reverse"
	IndexGeneratorModeModTime IndexGeneratorMode = "modtime"
)

var ErrNotSupported = errors.New("unsupported web format")

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

type builder struct {
	Site
	FlagsV2
}

func NewManifestBuilder(s Site, f FlagsV2) *builder {
	b := &builder{Site: s, FlagsV2: f}
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

func (b *builder) Caching() bool { panic("unexpected call to Caching()") }
func (b *builder) Writers() int  { panic("unexpected call to Writers()") }

func (b FlagsV2) Stage() Stage {
	s := StageAll
	if b.NoCleanup {
		s.Skip(StageCleanUp)
	}
	if b.NoCopy {
		s.Skip(StageCopy)
	}
	if b.NoBuild {
		s.Skip(StageBuild)
	}
	return s
}

func (b *builder) Hooks() []ssg.Hook {
	if b.NoBuild {
		return nil
	}

	minifiers := make(map[string]MinifyFn)
	if b.MinifyHtmlCopy {
		minifiers[".html"] = MinifyHtml
	}
	if b.MinifyCss {
		minifiers[".css"] = MinifyCss
	}
	if b.MinifyJs {
		minifiers[".js"] = MinifyJs
	}
	if b.MinifyJson {
		minifiers[".json"] = MinifyJson
	}

	hookMinifies := HookMinify(minifiers)
	hookReplacer := HookReplacer(b.Replaces)

	if hookMinifies == nil && hookReplacer == nil {
		return nil
	}

	// Minify only
	if b.NoReplace {
		if hookMinifies == nil {
			return nil
		}
		return []ssg.Hook{
			hookMinifies,
		}
	}

	// Replace and minify
	if hookMinifies == nil {
		return []ssg.Hook{
			hookReplacer,
		}
	}
	return []ssg.Hook{
		hookReplacer,
		hookMinifies,
	}
}

func (b *builder) HooksGenerate() []ssg.HookGenerate {
	if b.MinifyHtmlGenerate {
		return []ssg.HookGenerate{
			MinifyHtml,
		}
	}
	return nil
}

func (b *builder) Pipelines() []any {
	if b.NoGenerateIndex || !b.GenerateIndex {
		return nil
	}
	return []any{
		NewIndexGenerator(b.GenerateIndexMode)(&b.ssg),
	}
}

func ApplyManifestV2(m Manifest, f FlagsV2, do Stage) error {
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
		b := NewManifestBuilder(site, f)

		log.
			With(
				"key", key,
				"url", site.ssg.Url,
				// @TODO: Len logs below will be removed
				"len_hooks", len(b.Hooks()),
				"len_hooks_generate", len(b.HooksGenerate()),
				"len_pipelines", len(b.Pipelines()),
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
