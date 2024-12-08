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

func ArticleGenerator(impl ssg.Impl) ssg.Impl {
	return func(path string, data []byte, d fs.DirEntry) error {
		base := filepath.Base(path)
		if !d.IsDir() && strings.Contains(path, "/blog/") && base == markerBlog {
			parent := filepath.Dir(path)
			fmt.Fprintf(os.Stdout, "found blog marker=%s, parent=%s\n", path, parent)
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("failed to read dir for blog %s: %w", path, err)
			}

			articles := []string{}
			for i := range entries {
				fname := entries[i].Name()
				if len(fname) == 0 {
					return fmt.Errorf("unexpected empty filename in %s", path)
				}
				if fname == "_header.html" {
					continue
				}
				if fname == "_footer.html" {
					continue
				}
				if fname == markerBlog {
					continue
				}

				fmt.Fprintf(os.Stdout, "found article %s\n", fname)
				articles = append(articles, fname)
			}

			heading := filepath.Base(parent)
			content := bytes.NewBufferString(fmt.Sprintf(":title Blog %s\n\n<h1>Blog %s</h1>", heading, heading))
			for i := range articles {
				article := articles[i]
				fmt.Fprintf(content, "- [%s](./%s/%s)\n\n", article, parent, article)
			}

			path = filepath.Join(parent, "index.md")
			data = content.Bytes()
		}

		return impl(path, data, d)
	}
}
