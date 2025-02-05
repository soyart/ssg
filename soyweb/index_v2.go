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

type IndexMode string

const (
	IndexModeDefault IndexMode = ""
	IndexModeModTime IndexMode = "mod-time"
)

func IndexGeneratorV2(mode IndexMode) func(*ssg.Ssg) ssg.Pipeline {
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

			switch mode {
			case IndexModeModTime:
				sort.Slice(infos, func(i, j int) bool {
					infoI, infoJ := infos[i], infos[j]
					cmp := infoI.ModTime().Compare(infoJ.ModTime())
					if cmp == 0 {
						return infoI.Name() < infoJ.Name()
					}

					return cmp == -1
				})
			}

			template, err := ssg.ReadFile(path)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to read marker '%s': %w", path, err)
			}
			index, err := generateIndexV2(s.Src, s.Ignore, parent, infos, template)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to generate article links for marker %s: %w", path, err)
			}

			return filepath.Join(parent, "index.md"), []byte(index), d, nil
		}
	}
}

func generateIndexV2(
	src string,
	ignore func(path string) bool,
	parent string,
	siblings []fs.FileInfo,
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
		sibIsDir := sib.IsDir()

		switch sibName {
		case "index.html", "index.md":
			if !sibIsDir {
				return "", fmt.Errorf("parent %s already had index %s", parent, sibName)
			}

		case
			MarkerIndex,      // Marker itself
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

		// Default is to use dir/filename as link title
		linkTitle := sibName

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

		ssg.Fprintf(content, "- [%s](/%s)\n\n", linkTitle, link)
	}

	// ssg.Fprintln(os.Stdout, "Generated index for", parent)
	// ssg.Fprint(os.Stdout, "======= START =======\n")
	// ssg.Fprintln(os.Stdout, content.String())
	// ssg.Fprint(os.Stdout, "======== END ========\n")

	return content.String(), nil
}
