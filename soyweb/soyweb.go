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
