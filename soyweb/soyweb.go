package soyweb

import (
	"errors"

	"github.com/soyart/ssg"
)

const (
	MarkerIndex = "_index.soyweb"
)

var ErrNotSupported = errors.New("unsupported web format")

type (
	MinifyFlags struct {
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

	Flags struct {
		MinifyFlags
		NoMinifyFlags
		GenerateIndex bool `arg:"--gen-index" default:"true" help:"Generate index on _index.soyweb"`
	}
)

// IndexGenerator returns an option that will make ssg generates
// index.md/index.html for all unignored _index.soyweb marker files.
func IndexGenerator() ssg.Option {
	return func(s *ssg.Ssg) {
		generator := indexGenerator(s)
		ssg.WithPipeline(generator)(s)
	}
}

func SsgOptions(f Flags) []ssg.Option {
	f.MinifyFlags = negate(f.MinifyFlags, f.NoMinifyFlags)
	opts := []ssg.Option{}

	if f.GenerateIndex {
		opts = append(opts, IndexGenerator())
	}

	minifiers := make(map[string]MinifyFn)
	if f.MinifyHtmlCopy {
		minifiers[".html"] = MinifyHtml
	}
	if f.MinifyCss {
		minifiers[".css"] = MinifyCss
	}
	if f.MinifyJs {
		minifiers[".js"] = MinifyJs
	}
	if f.MinifyJson {
		minifiers[".json"] = MinifyJson
	}

	hook := pipelineMinify(minifiers)
	if hook != nil {
		opts = append(opts, ssg.WithHookAll(hook))
	}
	if f.MinifyHtmlGenerate {
		opts = append(opts, ssg.WithHookGenerate(MinifyHtml))
	}

	return opts
}

func (m MinifyFlags) Skip(ext string) bool {
	switch ext {
	case ".html":
		if m.MinifyHtmlGenerate {
			return false
		}
		if m.MinifyHtmlCopy {
			return false
		}
	case ".css":
		if m.MinifyCss {
			return false
		}
	case ".js":
		if m.MinifyJs {
			return false
		}
	case ".json":
		if m.MinifyJson {
			return false
		}

	default:
		return true
	}

	return true
}

func (n NoMinifyFlags) Skip(ext string) bool {
	switch ext {
	case ".html":
		if n.NoMinifyHtmlGenerate {
			return true
		}
		if n.NoMinifyHtmlCopy {
			return true
		}
	case ".css":
		if n.NoMinifyCss {
			return true
		}
	case ".js":
		if n.NoMinifyJs {
			return true
		}
	case ".json":
		if n.NoMinifyJson {
			return true
		}

	default:
		return true
	}

	return false
}

func negate(yes MinifyFlags, no NoMinifyFlags) MinifyFlags {
	if no.NoMinifyHtmlGenerate {
		yes.MinifyHtmlGenerate = false
	}
	if no.NoMinifyHtmlCopy {
		yes.MinifyHtmlCopy = false
	}
	if no.NoMinifyCss {
		yes.MinifyCss = false
	}
	if no.NoMinifyJs {
		yes.MinifyJs = false
	}
	if no.NoMinifyJson {
		yes.MinifyJson = false
	}

	return yes
}
