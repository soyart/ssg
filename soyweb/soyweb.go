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
