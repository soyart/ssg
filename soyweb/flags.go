package soyweb

import (
	"github.com/soyart/ssg/ssg-go"
)

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
