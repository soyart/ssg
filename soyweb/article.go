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

func ArticleGeneratorMarkdown(impl ssg.Impl) ssg.Impl {
	return func(path string, data []byte, d fs.DirEntry) error {
		base := filepath.Base(path)
		if base != markerBlog {
			return impl(path, data, d)
		}
		if d.IsDir() {
			return impl(path, data, d)
		}
		if !strings.Contains(path, "/blog/") {
			return impl(path, data, d)
		}

		parent := filepath.Dir(path)
		fmt.Fprintf(os.Stdout, "found blog marker: marker=\"%s\", parent=\"%s\"\n", path, parent)

		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("failed to read dir for blog %s: %w", path, err)
		}

		content, err := articleLink(parent, entries)
		if err != nil {
			return fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
		}

		path = filepath.Join(parent, "index.md")
		data = []byte(content)

		return impl(path, data, d)
	}
}

func articleLink(
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
		articlePath := article.Name()
		// TODO: extract title
		articleTitle := articlePath

		switch articlePath {
		case
			"_header.html",
			"_footer.html",
			"_blog.ssg":

			continue
		}

		switch {
		case article.IsDir():
			// Find 1st-level subdir with index.html or index.md
			// e.g. /parent/article/index.html
			// or   /parent/article/index.md

			subEntries, err := os.ReadDir(filepath.Join(parent, articlePath))
			if err != nil {
				return "", fmt.Errorf("failed to read article dir %s: %w", articlePath, err)
			}

			foundIndex := false
			for j := range subEntries {
				index := subEntries[j].Name()
				if index != "index.md" && index != "index.html" {
					continue
				}

				foundIndex = true
				break
			}

			if !foundIndex {
				continue
			}

		case filepath.Ext(articlePath) != ".md":
			continue

		case filepath.Ext(articlePath) == ".md":
			articlePath = strings.TrimSuffix(articlePath, ".md")
			articlePath += ".html"
		}

		fmt.Fprintf(content, "- [%s](./%s/%s)", articleTitle, parent, articlePath)
		if i != l-1 {
			content.WriteString("\n\n")
		}

		content.WriteString("\n")
	}

	fmt.Println("Generated article list for blog", parent)
	fmt.Println("======= START =======")
	fmt.Println(content.String())
	fmt.Println("======== END ========")

	return content.String(), nil
}
