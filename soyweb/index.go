package soyweb

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/soyart/ssg/ssg-go"
)

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
		generatorDefault,
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
		generatorDefault,
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
		generatorDefault,
	)(s)
}

func reverseInPlace(arr []fs.FileInfo) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
}

// IndexGeneratorTemplate allows us to build an index generator pipeline from 2 components:
// 1. fnEntries - a function that intercepts entries and returns the actual entries to be used.
// This is useful when you want to implement some kind of entry filter, or just want to inspect entries
// before handling them over to the generators.
//
// 2. fnGenIndex - a function that is called for each marker _index.soyweb.
func IndexGeneratorTemplate(
	fnEntries func(entries []fs.FileInfo) []fs.FileInfo,
	fnGenIndex func(
		ssgSrc string,
		ignore func(path string) bool,
		parent string,
		siblings []fs.FileInfo,
		template []byte,
	) (
		string,
		error,
	),
) func(*ssg.Ssg) ssg.Pipeline {
	return func(s *ssg.Ssg) ssg.Pipeline {
		return func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error) {
			switch {
			case
				d.IsDir(),
				filepath.Base(path) != MarkerIndex:
				return path, data, d, nil

			case s.Ignore(path):
				panic("unexpected ignored file for index-generator: " + path)
			}

			parent := filepath.Dir(path)
			ssg.Fprintf(os.Stdout, "found index-generator marker: marker=\"%s\", parent=\"%s\"\n", path, parent)

			entries, err := os.ReadDir(parent)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to read marker dir '%s': %w", path, err)
			}

			infos := make([]fs.FileInfo, len(entries))
			for i := range entries {
				entry := entries[i]
				info, err := entry.Info()
				if err != nil {
					return "", nil, nil, fmt.Errorf("failed to stat entry '%s' in path '%s': %w", entry.Name(), path, err)
				}

				infos[i] = info
			}

			if fnEntries != nil {
				infos = fnEntries(infos)
			}

			template, err := ssg.ReadFile(path)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to read marker '%s': %w", path, err)
			}
			index, err := fnGenIndex(s.Src, s.Ignore, parent, infos, template)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
			}

			return filepath.Join(parent, "index.md"), []byte(index), d, nil
		}
	}
}

// generatorDefault is a default index generator.
//
// It generates 1 index.md for each _index.soyweb.
// The default generator does accept a template, and will append its generated content
// to the template. The generated content will be lines of text,
// each a markdown simple link to a sibling of the marker.
//
// Each sibling of the marker will be given 1 "link" line in markdown,
// each line composing of 2 components: a link title and the actual link path,
// looking something like this: `[link-title](/actual/link)`.
//
// generatorDefault ensures that all links have titles, and will automatically
// select link titles based on these 2 steps:
//
// 1. From the entry filename or directory name.
// For example, if there're 2 entries ./entry1.md and ./entry-2/index.md,
// then generatorDefault will first assign "entry1" as link title for ./entry1.md,
// while "entry-2" is used for ./entry-2/index.md.
//
// 2. From the entry's 1st h1 tag.
// If your entry happens to have an h1 tag, generatorDefault will use those as link title.
// Otherwise it just sticks with link title previously obtained from step 1.
func generatorDefault(
	src string,
	ignore func(path string) bool,
	parent string,
	siblings []fs.FileInfo,
	template []byte,
) (
	string,
	error,
) {
	output := bytes.NewBuffer(template)
	if len(template) == 0 {
		// Default template is a simple h1
		ssg.Fprintf(output, "# Index of %s\n\n", filepath.Base(parent))
	}

	for i := range siblings {
		sib := siblings[i]
		sibName := sib.Name()
		sibIsDir := sib.IsDir()

		// Default is to use dir/filename as link title
		linkTitle := sibName

		switch sibName {
		case "index.html", "index.md":
			if !sibIsDir {
				return "", fmt.Errorf("parent %s already had index %s", parent, sibName)
			}

		case
			ssg.MarkerHeader, // Ignore
			ssg.MarkerFooter: // Ignore
			continue
		}

		sibExt := filepath.Ext(sibName)
		if !sibIsDir && sibExt != ".md" && sibExt != ".html" {
			continue
		}
		sibPath := filepath.Join(parent, sibName)
		if ignore(sibPath) {
			continue
		}

		switch {
		case sibIsDir:
			// Find 1st-level subdir with index.html or index.md
			// e.g. /parent/article/index.html
			// or   /parent/article/index.md
			nephews, err := os.ReadDir(sibPath)
			if err != nil {
				return "", fmt.Errorf("failed to read nephew dir '%s': %w", sibName, err)
			}

			index := ""
			recurse := false
			for j := range nephews {
				nephew := nephews[j]
				if nephew.IsDir() {
					continue
				}
				name := nephew.Name()
				if name == "index.html" || name == "index.md" {
					index = name
					break
				}
				if name == MarkerIndex {
					nephewMarker := filepath.Join(parent, sibName, name)
					nephewMarkerH1, err := extractTitle(nephewMarker)
					if err == nil && len(nephewMarkerH1) != 0 {
						linkTitle = string(nephewMarkerH1)
					}
					recurse = true
					break
				}
			}

			// Use dir as childTitle
			if recurse {
				break // switch
			}
			// Use dir as childTitle
			// No need to extract and change title from Markdown
			if index == "index.html" {
				break // switch
			}
			// No index in child, won't build index for the sibling
			if index == "" {
				continue
			}

			// Get linkTitle from nephew's content
			title, err := extractTitle(filepath.Join(sibPath, index))
			if err != nil {
				return "", err
			}
			if len(title) != 0 {
				linkTitle = string(title)
			}

		case sibExt == ".md":
			title, err := extractTitle(sibPath)
			if err != nil {
				return "", err
			}
			if len(title) != 0 {
				linkTitle = string(title)
			}
			sibName = ssg.ChangeExt(sibName, ".md", ".html")
		}

		rel, err := filepath.Rel(src, parent)
		if err != nil {
			return "", err
		}
		link := filepath.Join(rel, sibName)
		if sibIsDir {
			link += "/"
		}

		ssg.Fprintf(output, "- [%s](/%s)\n\n", linkTitle, link)
	}

	// ssg.Fprintln(os.Stdout, "Generated index for", parent)
	// ssg.Fprint(os.Stdout, "======= START =======\n")
	// ssg.Fprintln(os.Stdout, content.String())
	// ssg.Fprint(os.Stdout, "======== END ========\n")

	return output.String(), nil
}

func extractTitle(path string) ([]byte, error) {
	data, err := ssg.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read article file %s for title extraction: %w", path, err)
	}
	title := ssg.GetTitleFromTag(data)
	if len(title) != 0 {
		return title, nil
	}

	return ssg.GetTitleFromH1(data), nil
}
