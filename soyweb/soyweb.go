package soyweb

import "github.com/soyart/ssg"

type Flags struct {
	MinifyFlags
}

func SsgOptions(f Flags) []ssg.Option {
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
