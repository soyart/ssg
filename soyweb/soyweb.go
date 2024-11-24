package soyweb

import (
	"github.com/soyart/ssg"
)

type (
	MinifyFlags struct {
		MinifyHtml    bool `arg:"--min-html" help:"Minify HTML outputs"`
		MinifyHtmlAll bool `arg:"--min-html-all" help:"Minify all HTML outputs"`
		MinifyCss     bool `arg:"--min-css" help:"Minify CSS files"`
		MinifyJson    bool `arg:"--min-json" help:"Minify JSON files"`
	}

	NoMinifyFlags struct {
		NoMinifyHtml    bool `arg:"--no-min-html,env:NO_MIN_HTML" help:"Do not minify HTML outputs"`
		NoMinifyHtmlAll bool `arg:"--no-min-html-all,env:NO_MIN_HTML_ALL" help:"Do not minify all HTML outputs"`
		NoMinifyCss     bool `arg:"--no-min-css,env:NO_MIN_CSS" help:"Do not minify CSS files"`
		NoMinifyJson    bool `arg:"--no-min-json,env:NO_MIN_JSON" help:"Do not minify JSON files"`
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

	if f.MinifyHtmlAll {
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

func negate(yes MinifyFlags, no NoMinifyFlags) MinifyFlags {
	if no.NoMinifyHtml {
		yes.MinifyHtml = false
	}
	if no.NoMinifyHtmlAll {
		yes.MinifyHtmlAll = false
	}
	if no.NoMinifyCss {
		yes.MinifyCss = false
	}
	if no.NoMinifyJson {
		yes.MinifyJson = false
	}

	return yes
}
