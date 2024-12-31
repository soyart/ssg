package soyweb

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/soyart/ssg"
)

// indexGenerator returns an [ssg.Pipeline] that would look for
// marker file "_index.soyweb" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl.
func indexGenerator(s *ssg.Ssg) ssg.Pipeline {
	src := s.Src
	ignore := s.Ignore

	return func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error) {
		switch {
		case
			d.IsDir(),
			filepath.Base(path) != MarkerIndex:
			return path, data, d, nil

		case ignore(path):
			panic("unexpected ignored file for index-generator: " + path)
		}

		parent := filepath.Dir(path)
		ssg.Fprintf(os.Stdout, "found index-generator marker: marker=\"%s\", parent=\"%s\"\n", path, parent)

		entries, err := os.ReadDir(parent)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to read marker dir '%s': %w", path, err)
		}
		template, err := os.ReadFile(path)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to read marker '%s': %w", path, err)
		}

		index, err := genIndex(src, ignore, parent, entries, template)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
		}

		return filepath.Join(parent, "index.md"), []byte(index), d, nil
	}
}

func genIndex(
	src string,
	ignore func(path string) bool,
	parent string,
	siblings []fs.DirEntry,
	template []byte,
) (
	string,
	error,
) {
	content := bytes.NewBuffer(template)
	if content.Len() == 0 {
		ssg.Fprintf(content, "# Index of %s\n\n", filepath.Base(parent))
	}

	for i := range siblings {
		sib := siblings[i]
		sibName := sib.Name()
		linkTitle := sibName

		switch sibName {
		case
			ssg.MarkerHeader,
			ssg.MarkerFooter,
			MarkerIndex:
			continue
		}

		isDir := sib.IsDir()
		sibExt := filepath.Ext(sibName)
		if !isDir && (sibExt != ".md" && sibExt != ".html") {
			continue
		}

		sibPath := filepath.Join(parent, sibName)
		if ignore(sibPath) {
			continue
		}

		switch {
		case isDir:
			// Find 1st-level subdir with index.html or index.md
			// e.g. /parent/article/index.html
			// or   /parent/article/index.md
			children, err := os.ReadDir(sibPath)
			if err != nil {
				return "", fmt.Errorf("failed to read child dir %s: %w", sibName, err)
			}

			index := ""
			recurse := false
			for j := range children {
				name := children[j].Name()
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
				break // switch
			}
			// No index in child, won't build index line
			if index == "" {
				continue
			}
			// No need to extract and change title
			if index == "index.html" {
				break // switch
			}
			// Try to extract and change link title
			title, err := extractTitle(filepath.Join(sibPath, index))
			if err != nil {
				return "", err
			}
			if len(title) != 0 {
				linkTitle = string(title)
			}

		case sibExt == ".md" || sibExt == ".html":
			title, err := extractTitle(sibPath)
			if err != nil {
				return "", err
			}
			if len(title) != 0 {
				linkTitle = string(title)
			}
			if sibExt == ".md" {
				sibName = ssg.ChangeExt(sibName, ".md", ".html")
			}

		default:
			panic("unhandled case for child: " + filepath.Join(parent, sibName))
		}

		rel, err := filepath.Rel(src, parent)
		if err != nil {
			return "", err
		}
		link := filepath.Join(rel, sibName)
		if isDir {
			link += "/"
		}

		ssg.Fprintf(content, "- [%s](/%s)\n\n", linkTitle, link)
	}

	ssg.Fprintln(os.Stdout, "Generated Markdown index for directory", parent)
	ssg.Fprint(os.Stdout, "======= START =======\n")
	ssg.Fprintln(os.Stdout, content.String())
	ssg.Fprint(os.Stdout, "======== END ========\n")

	return content.String(), nil
}

func extractTitle(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read article file %s for title extraction: %w", path, err)
	}
	title := ssg.GetTitleFromTag(data)
	if len(title) != 0 {
		return title, nil
	}

	return ssg.GetTitleFromH1(data), nil
}
