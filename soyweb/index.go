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

// IndexGenerator returns an [ssg.Impl] that would look for
// marker file "_index.ssg" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl.
func IndexGenerator(
	src string,
	_dst string, //nolint:unused
	impl ssg.Impl,
) ssg.Impl {
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

		if len(template) != 0 {
			fmt.Fprintf(os.Stdout, "found template for marker '%s'\n", path)
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
	entries []fs.DirEntry,
	template []byte,
) (
	string,
	error,
) {
	var content *bytes.Buffer
	switch len(template) {
	case 0:
		heading := filepath.Base(parent)
		heading = fmt.Sprintf(":title Blog %s\n\n<h1>Blog %s</h1>\n\n", heading, heading)
		content = bytes.NewBufferString(heading)

	default:
		content = bytes.NewBuffer(template)
	}

	l := len(entries)
	for i := range entries {
		article := entries[i]
		articleFname := article.Name()
		articleTitle := articleFname

		switch articleFname {
		case
			ssg.MarkerHeader,
			ssg.MarkerFooter,
			markerIndex:
			continue
		}

		if !article.IsDir() && filepath.Ext(articleFname) != ".md" {
			continue
		}

		switch {
		case article.IsDir():
			// Find 1st-level subdir with index.html or index.md
			// e.g. /parent/article/index.html
			// or   /parent/article/index.md
			articleDir := filepath.Join(parent, articleFname)
			subEntries, err := os.ReadDir(articleDir)
			if err != nil {
				return "", fmt.Errorf("failed to read article dir %s: %w", articleFname, err)
			}

			index := ""
			recurse := false
			for j := range subEntries {
				name := subEntries[j].Name()
				if name == "_index.ssg" {
					index = "index.html"
					recurse = true
					break
				}

				if name != "index.md" && name != "index.html" {
					continue
				}

				index = name
				break
			}

			if !recurse && index == "" {
				continue
			}

			articleFname = filepath.Join(articleFname, "index.html")
			if recurse {
				break // switch
			}

			titleFromTag, err := extractTitleFromTag(filepath.Join(articleDir, index))
			if err != nil {
				return "", err
			}
			if titleFromTag != nil {
				articleTitle = string(titleFromTag)
			}

		case filepath.Ext(articleFname) == ".md":
			articlePath := filepath.Join(parent, articleFname)
			titleFromTag, err := extractTitleFromTag(articlePath)
			if err != nil {
				return "", err
			}

			if len(titleFromTag) > 0 {
				articleTitle = string(titleFromTag)
			}

			articleFname = strings.TrimSuffix(articleFname, ".md")
			articleFname += ".html"

		default:
			panic("unhandled case for blog: " + filepath.Join(parent, articleFname))
		}

		rel, err := filepath.Rel(src, parent)
		if err != nil {
			return "", err
		}

		fmt.Fprintf(content, "- [%s](./%s/%s)", articleTitle, rel, articleFname)
		if i < l-1 {
			content.WriteString("\n\n")
			continue
		}

		content.WriteString("\n")
	}

	fmt.Println("Generated article list for blog", parent)
	fmt.Println("======= START =======")
	fmt.Println(content.String())
	fmt.Println("======== END ========")

	return content.String(), nil
}

func extractTitleFromTag(path string) ([]byte, error) {
	articleData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read article file %s: %w", path, err)
	}

	return ssg.TitleFromTag(articleData), nil
}
