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

const markerIndex = "_index.ssg"

// indexGenerator returns an [ssg.Pipeline] that would look for
// marker file "_index.ssg" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl.
func indexGenerator(src string, impl ssg.Impl) ssg.Impl {
	return func(path string, data []byte, d fs.DirEntry) error {
		switch {
		case
			d.IsDir(),
			filepath.Base(path) != markerIndex:

			return impl(path, data, d)
		}

		parent := filepath.Dir(path)
		fmt.Fprintf(os.Stdout, "found blog marker: marker=\"%s\", parent=\"%s\"\n", path, parent)

		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("failed to read marker dir '%s': %w", path, err)
		}

		template, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read marker '%s': %w", path, err)
		}

		content, err := genIndex(src, parent, entries, template)
		if err != nil {
			return fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
		}

		path = filepath.Join(parent, "index.md")
		data = []byte(content)
		return impl(path, data, d)
	}
}

func genIndex(
	src string,
	parent string,
	children []fs.DirEntry,
	template []byte,
) (
	string,
	error,
) {
	content := bytes.NewBuffer(template)
	if len(template) == 0 {
		heading := filepath.Base(parent)
		heading = fmt.Sprintf(":title Blog %s\n\n<h1>Blog %s</h1>\n\n", heading, heading)
		content = bytes.NewBufferString(heading)
	}

	for i := range children {
		child := children[i]
		childPath := child.Name()
		childTitle := childPath

		switch childPath {
		case
			ssg.MarkerHeader,
			ssg.MarkerFooter,
			markerIndex:
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
			grandChildren, err := os.ReadDir(childDir)
			if err != nil {
				return "", fmt.Errorf("failed to read child dir %s: %w", childPath, err)
			}

			index := ""
			recurse := false
			for j := range grandChildren {
				name := grandChildren[j].Name()
				if name == "_index.ssg" {
					index = "index.html"
					recurse = true
					break
				}
				if name == "index.md" || name == "index.html" {
					index = name
					break
				}
			}

			// No index
			if index == "" {
				continue
			}

			// Use dir as childTitle
			if recurse {
				break // switch
			}

			titleFromTag, err := extractTitleFromTag(filepath.Join(childDir, index))
			if err != nil {
				return "", err
			}
			if len(titleFromTag) != 0 {
				childTitle = string(titleFromTag)
			}

		case filepath.Ext(childPath) == ".md":
			articlePath := filepath.Join(parent, childPath)
			titleFromTag, err := extractTitleFromTag(articlePath)
			if err != nil {
				return "", err
			}

			if len(titleFromTag) != 0 {
				childTitle = string(titleFromTag)
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

		fmt.Fprintf(content, "- [%s](/%s)\n\n", childTitle, linkPath)
	}

	fmt.Fprintln(os.Stdout, "Generated index for directory", parent)
	fmt.Fprint(os.Stdout, "======= START =======\n")
	fmt.Fprintln(os.Stdout, content.String())
	fmt.Fprint(os.Stdout, "======== END ========\n")

	return content.String(), nil
}

func extractTitleFromTag(path string) ([]byte, error) {
	articleData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read article file %s for title extraction: %w", path, err)
	}

	return ssg.TitleFromTag(articleData), nil
}
