package soyweb

import (
	"github.com/soyart/ssg/ssg-go"
)

type (
	// FlagsV2 represents CLI arguments that could modify soyweb behavior, such as skipping stages
	// and minifying content of certain file extensions.
	FlagsV2 struct {
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

	NoMinifyFlags struct {
		NoMinifyHtmlGenerate bool `arg:"--no-min-html,env:NO_MIN_HTML" help:"Do not minify converted HTML outputs"`
		NoMinifyHtmlCopy     bool `arg:"--no-min-html-copy,env:NO_MIN_HTML_COPY" help:"Do not minify all copied HTML"`
		NoMinifyCss          bool `arg:"--no-min-css,env:NO_MIN_CSS" help:"Do not minify CSS files"`
		NoMinifyJs           bool `arg:"--no-min-js,env:NO_MIN_JS" help:"Do not minify Javascript files"`
		NoMinifyJson         bool `arg:"--no-min-json,env:NO_MIN_JSON" help:"Do not minify JSON files"`
	}
)

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

func (f NoMinifyFlags) Flags() FlagsV2 {
	return FlagsV2{
		MinifyHtmlGenerate: !f.NoMinifyHtmlGenerate,
		MinifyHtmlCopy:     !f.NoMinifyHtmlCopy,
		MinifyCss:          !f.NoMinifyCss,
		MinifyJs:           !f.NoMinifyJs,
		MinifyJson:         !f.NoMinifyJson,
	}
}

func (n NoMinifyFlags) Skip(ext string) bool {
	switch ext {
	case ExtHtml:
		if n.NoMinifyHtmlGenerate {
			return true
		}
		if n.NoMinifyHtmlCopy {
			return true
		}
	case ExtCss:
		if n.NoMinifyCss {
			return true
		}
	case ExtJs:
		if n.NoMinifyJs {
			return true
		}
	case ExtJson:
		if n.NoMinifyJson {
			return true
		}

		// Skip unknown file extension and media type
	default:
		return true
	}

	// Do not skip this extension
	return false
}
