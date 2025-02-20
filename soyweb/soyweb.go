package soyweb

import (
	"errors"
	"io/fs"
	"sort"

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
		GenerateIndex     bool               `arg:"--gen-index" default:"true" help:"Generate index on _index.soyweb"`
		GenerateIndexMode IndexGeneratorMode `arg:"--gen-index-mode" help:"Index generation mode"`
	}
)

func SsgOptions(f Flags) []ssg.Option {
	opts := []ssg.Option{}

	pipes := []interface{}{}
	if f.GenerateIndex {
		pipeGenIndex := GetIndexGenerator(f.GenerateIndexMode)
		pipes = append(pipes, pipeGenIndex)
	}

	minifiers := make(map[string]MinifyFn)
	f.MinifyFlags = negate(f.MinifyFlags, f.NoMinifyFlags)

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

	hook := hookMinify(minifiers)
	if hook != nil {
		opts = append(opts, ssg.WithHook(hook))
	}
	if f.MinifyHtmlGenerate {
		opts = append(opts, ssg.WithHookGenerate(MinifyHtml))
	}

	return append(opts, ssg.WithPipelines(pipes...))
}

func GetIndexGenerator(m IndexGeneratorMode) func(*ssg.Ssg) ssg.Pipeline {
	switch m {
	case
		IndexGeneratorModeReverse,
		"rev",
		"r":
		return IndexGeneratorReverse

	case
		IndexGeneratorModeModTime,
		"updated_at",
		"u":
		return IndexGeneratorModTime
	}

	return IndexGenerator
}

// IndexGenerator returns an [ssg.Pipeline] that would look for
// marker file "_index.soyweb" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl
func IndexGenerator(s *ssg.Ssg) ssg.Pipeline {
	return IndexGeneratorTemplate(
		nil,
		generateIndex,
	)(s)
}

// IndexGeneratorReverse returns an index generator whose index list
// is populated reversed, i.e. descending alphanumerical sort
func IndexGeneratorReverse(s *ssg.Ssg) ssg.Pipeline {
	return IndexGeneratorTemplate(
		func(entries []fs.FileInfo) []fs.FileInfo {
			reverseInPlace(entries)
			return entries
		},
		generateIndex,
	)(s)
}

// IndexGeneratorModTime returns an index generator that sort index entries
// by ModTime returned by fs.FileInfo
func IndexGeneratorModTime(s *ssg.Ssg) ssg.Pipeline {
	sortByModTime := func(entries []fs.FileInfo) func(i int, j int) bool {
		return func(i, j int) bool {
			infoI, infoJ := entries[i], entries[j]
			cmp := infoI.ModTime().Compare(infoJ.ModTime())
			if cmp == 0 {
				return infoI.Name() < infoJ.Name()
			}
			return cmp == -1
		}
	}

	return IndexGeneratorTemplate(
		func(entries []fs.FileInfo) []fs.FileInfo {
			sort.Slice(entries, sortByModTime(entries))
			return entries
		},
		generateIndex,
	)(s)
}

func reverseInPlace(arr []fs.FileInfo) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
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
