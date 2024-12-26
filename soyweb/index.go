package soyweb

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/soyart/ssg"
)

// indexGenerator returns an [ssg.Impl] that would look for
// marker file "_index.soyweb" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl.
func indexGenerator(s *ssg.Ssg) ssg.Impl {
	src := s.Src
	next := s.ImplDefault() // HTTP middleware-style
	ignore := s.Ignore

	return func(path string, data []byte, d fs.DirEntry) error {
		switch {
		case
			d.IsDir(),
			filepath.Base(path) != MarkerIndex:
			return next(path, data, d)

		case ignore(path):
			panic("unexpected ignored file for index-generator: " + path)
		}

		parent := filepath.Dir(path)
		ssg.Fprintf(os.Stdout, "found index-generator marker: marker=\"%s\", parent=\"%s\"\n", path, parent)

		entries, err := os.ReadDir(parent)
		if err != nil {
			return fmt.Errorf("failed to read marker dir '%s': %w", path, err)
		}
		template, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read marker '%s': %w", path, err)
		}

		index, err := genIndex(src, ignore, parent, entries, template)
		if err != nil {
			return fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
		}

		return next(filepath.Join(parent, "index.md"), []byte(index), d)
	}
}

func genIndex(
	src string,
	ignore func(path string) bool,
	parent string,
	children []fs.DirEntry,
	template []byte,
) (
	string,
	error,
) {
	content := bytes.NewBuffer(template)
	if content.Len() == 0 {
		ssg.Fprintf(content, "# Index of %s\n\n", filepath.Base(parent))
	}

	for i := range children {
		child := children[i]
		childPath := child.Name()
		childTitle := childPath

		if ignore(childPath) {
			continue
		}

		switch childPath {
		case
			ssg.MarkerHeader,
			ssg.MarkerFooter,
			MarkerIndex:
			continue
		}

		if !child.IsDir() && filepath.Ext(childPath) != ".md" {
			continue
		}

		isDir := child.IsDir()
		switch {
		case isDir:
			// Find 1st-level subdir with index.html or index.md
			// e.g. /parent/article/index.html
			// or   /parent/article/index.md
			childDir := filepath.Join(parent, childPath)
			if ignore(childDir) {
				continue
			}
			grandChildren, err := os.ReadDir(childDir)
			if err != nil {
				return "", fmt.Errorf("failed to read child dir %s: %w", childPath, err)
			}

			index := ""
			recurse := false
			for j := range grandChildren {
				name := grandChildren[j].Name()
				if name == "index.md" || name == "index.html" {
					index = name
					break
				}
				if name == MarkerIndex {
					recurse = true
					break
				}
			}

			// Use dir as childTitle
			if recurse {
				break // break switch
			}
			// No index in child, won't build index line
			if index == "" {
				continue
			}

			titleFromDoc, err := extractChildTitle(filepath.Join(childDir, index))
			if err != nil {
				return "", err
			}
			if len(titleFromDoc) != 0 {
				childTitle = string(titleFromDoc)
			}

		case filepath.Ext(childPath) == ".md":
			articlePath := filepath.Join(parent, childPath)
			titleFromDoc, err := extractChildTitle(articlePath)
			if err != nil {
				return "", err
			}
			if len(titleFromDoc) != 0 {
				childTitle = string(titleFromDoc)
			}

			childPath = strings.TrimSuffix(childPath, ".md")
			childPath += ".html"

		default:
			panic("unhandled case for child: " + filepath.Join(parent, childPath))
		}

		parentRel, err := filepath.Rel(src, parent)
		if err != nil {
			return "", err
		}
		linkPath := filepath.Join(parentRel, childPath)
		if isDir {
			linkPath += "/"
		}

		ssg.Fprintf(content, "- [%s](/%s)\n\n", childTitle, linkPath)
	}

	ssg.Fprintln(os.Stdout, "Generated Markdown index for directory", parent)
	ssg.Fprint(os.Stdout, "======= START =======\n")
	ssg.Fprintln(os.Stdout, content.String())
	ssg.Fprint(os.Stdout, "======== END ========\n")

	return content.String(), nil
}

func extractChildTitle(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read article file %s for title extraction: %w", path, err)
	}
	title := ssg.GetTitleFromTag(data)
	if len(title) == 0 {
		title = ssg.GetTitleFromH1(data)
	}
	return title, nil
}
