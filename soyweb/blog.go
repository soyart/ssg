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

const markerBlog = "_blog.ssg"

// ArticleListGenerator returns an [ssg.Impl] that would look for
// marker file "_blog.ssg" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl.
func ArticleListGenerator(
	src string,
	_dst string, //nolint:unused
	impl ssg.Impl,
) ssg.Impl {
	return func(path string, data []byte, d fs.DirEntry) error {
		switch {
		case
			d.IsDir(),
			filepath.Base(path) != markerBlog:

			return impl(path, data, d)
		}

		parent := filepath.Dir(path)
		fmt.Fprintf(os.Stdout, "found blog marker: marker=\"%s\", parent=\"%s\"\n", path, parent)

		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("failed to read dir for blog %s: %w", path, err)
		}

		content, err := articleLink(src, parent, entries)
		if err != nil {
			return fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
		}

		path = filepath.Join(parent, "index.md")
		data = []byte(content)
		return impl(path, data, d)
	}
}

func articleLink(
	src string,
	parent string,
	entries []fs.DirEntry,
) (
	string,
	error,
) {
	heading := filepath.Base(parent)
	heading = fmt.Sprintf(":title Blog %s\n\n<h1>Blog %s</h1>\n\n", heading, heading)

	l := len(entries)
	content := bytes.NewBufferString(heading)

	for i := range entries {
		article := entries[i]
		articleFname := article.Name()
		articleTitle := articleFname

		switch articleFname {
		case
			"_header.html",
			"_footer.html",
			"_blog.ssg":

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
				if name == "_blog.ssg" {
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
