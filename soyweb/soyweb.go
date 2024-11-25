package soyweb

import (
	"errors"

	"github.com/soyart/ssg"
)

var (
	ErrNotSupported = errors.New("unsupported web format")
)

type (
	MinifyFlags struct {
		MinifyHtml     bool `arg:"--min-html" help:"Minify converted HTML outputs"`
		MinifyHtmlCopy bool `arg:"--min-html-all" help:"Minify all copied HTML"`
		MinifyCss      bool `arg:"--min-css" help:"Minify CSS files"`
		MinifyJson     bool `arg:"--min-json" help:"Minify JSON files"`
	}

	NoMinifyFlags struct {
		NoMinifyHtml     bool `arg:"--no-min-html,env:NO_MIN_HTML" help:"Do not minify converted HTML outputs"`
		NoMinifyHtmlCopy bool `arg:"--no-min-html-copy,env:NO_MIN_HTML_COPY" help:"Do not minify all copied HTML"`
		NoMinifyCss      bool `arg:"--no-min-css,env:NO_MIN_CSS" help:"Do not minify CSS files"`
		NoMinifyJson     bool `arg:"--no-min-json,env:NO_MIN_JSON" help:"Do not minify JSON files"`
	}

	Flags struct {
		MinifyFlags
		NoMinifyFlags
	}
)

func SsgOptions(f Flags) []ssg.Option {
	f.MinifyFlags = negate(f.MinifyFlags, f.NoMinifyFlags)

	minifiers := make(map[string]MinifyFn)
	opts := []ssg.Option{}

	if f.MinifyHtmlCopy {
		minifiers[".html"] = MinifyHtml
	}
	if f.MinifyCss {
		minifiers[".css"] = MinifyCss
	}
	if f.MinifyJson {
		minifiers[".json"] = MinifyJson
	}

	pipeline := pipelineMinify(minifiers)
	if pipeline != nil {
		opts = append(opts, ssg.Pipeline(pipeline))
	}
	if f.MinifyHtml {
		opts = append(opts, ssg.Hook(MinifyHtml))
	}

	return opts
}

func (m MinifyFlags) Skip(ext string) bool {
	switch ext {
	case ".html":
		if m.MinifyHtml {
			return false
		}
		if m.MinifyHtmlCopy {
			return false
		}

	case ".css":
		if m.MinifyCss {
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
		if n.NoMinifyHtml {
			return true
		}
		if n.NoMinifyHtmlCopy {
			return true
		}

	case ".css":
		if n.NoMinifyCss {
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
	if no.NoMinifyHtml {
		yes.MinifyHtml = false
	}
	if no.NoMinifyHtmlCopy {
		yes.MinifyHtmlCopy = false
	}
	if no.NoMinifyCss {
		yes.MinifyCss = false
	}
	if no.NoMinifyJson {
		yes.MinifyJson = false
	}

	return yes
}
