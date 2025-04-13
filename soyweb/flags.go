package soyweb

import (
	"io/fs"
	"sort"

	"github.com/soyart/ssg/ssg-go"
)

type (
	NoMinifyFlags struct {
		NoMinifyHtmlGenerate bool `arg:"--no-min-html,env:NO_MIN_HTML" help:"Do not minify converted HTML outputs"`
		NoMinifyHtmlCopy     bool `arg:"--no-min-html-copy,env:NO_MIN_HTML_COPY" help:"Do not minify all copied HTML"`
		NoMinifyCss          bool `arg:"--no-min-css,env:NO_MIN_CSS" help:"Do not minify CSS files"`
		NoMinifyJs           bool `arg:"--no-min-js,env:NO_MIN_JS" help:"Do not minify Javascript files"`
		NoMinifyJson         bool `arg:"--no-min-json,env:NO_MIN_JSON" help:"Do not minify JSON files"`
	}
)

func (f NoMinifyFlags) Flags() FlagsV2 {
	return FlagsV2{
		MinifyHtmlGenerate: !f.NoMinifyHtmlGenerate,
		MinifyHtmlCopy:     !f.NoMinifyHtmlCopy,
		MinifyCss:          !f.NoMinifyCss,
		MinifyJs:           !f.NoMinifyJs,
		MinifyJson:         !f.NoMinifyJson,
	}
}

func NewIndexGenerator(m IndexGeneratorMode) func(*ssg.Ssg) ssg.Pipeline {
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
