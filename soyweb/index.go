package soyweb

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/soyart/ssg/ssg-go"
)

// IndexGenerator returns an [ssg.Pipeline] that would look for
// marker file "_index.soyweb" within a directory.
//
// Once it finds a marked directory, it inspects the children
// and generate a Markdown list with name index.md,
// which is later sent to supplied impl.
func IndexGenerator(s *ssg.Ssg) ssg.Pipeline {
	return IndexGeneratorV2(nil)(s)
}

func generateIndex(
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
