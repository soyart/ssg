package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/soyart/ssg"
)

func main() {
	s := ssg.New("soyweb/testdata/johndoe.com/src", "soyweb/testdata/johndoe.com/dst", "TestWithImpl", "https://johndoe.com")
	implDefault := s.ImplDefault()

	impl := ssg.WithImpl(func(path string, data []byte, d fs.DirEntry) error {
		base := filepath.Base(path)

		if !d.IsDir() && strings.Contains(path, "/blog/") && base == "_blog.ssg" {
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
				if fname == "_blog.ssh" {
					continue
				}

				fmt.Fprintf(os.Stdout, "found article %s\n", fname)
				articles = append(articles, fname)
			}

			heading := filepath.Base(parent)
			content := bytes.NewBufferString(fmt.Sprintf("Blog %s\n\n", heading))
			for i := range articles {
				article := articles[i]
				line := fmt.Sprintf("[%s](./%s/%s)\n", article, parent, article)
				content.WriteString(line)
			}

			path = filepath.Join(parent, "index.md")
			data = content.Bytes()
		}

		return implDefault(path, data, d)
	})

	s.With(impl)
	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
