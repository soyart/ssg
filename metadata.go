package ssg

import (
	"bytes"
	"path/filepath"
	"sort"
	"time"
)

func GenerateMetadata(
	src string,
	dst string,
	url string,
	files []string,
	dist []OutputFile,
	srcModTime time.Time,
) error {
	metadata, err := Metadata(src, dst, url, files, dist, srcModTime)
	if err != nil {
		return err
	}
	return WriteOutSlice(metadata, 2)
}

func Metadata(
	src string,
	dst string,
	url string,
	files []string,
	dist []OutputFile,
	srcModTime time.Time,
) (
	[]OutputFile,
	error,
) {
	sort.Slice(dist, func(i, j int) bool {
		return dist[i].target < dist[j].target
	})
	dotFiles, err := DotFiles(src, files)
	if err != nil {
		return nil, err
	}
	sitemap, err := Sitemap(dst, url, srcModTime, dist)
	if err != nil {
		return nil, err
	}
	return []OutputFile{
		Output(filepath.Join(dst, "sitemap.xml"), "", []byte(sitemap), 0644),
		Output(filepath.Join(dst, ".files"), "", []byte(dotFiles), 0644),
	}, nil
}

// Sitemap returns content of ${dst}/sitemap.xml
func Sitemap(
	dst string,
	url string,
	modTime time.Time,
	outputs []OutputFile,
) (
	string,
	error,
) {
	dateStr := modTime.Format(time.DateOnly)
	sm := bytes.NewBufferString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset
xmlns:xsi="https://www.w3.org/2001/XMLSchema-instance"
xsi:schemaLocation="https://www.sitemaps.org/schemas/sitemap/0.9
https://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"
xmlns="https://www.sitemaps.org/schemas/sitemap/0.9">
`)
	for i := range outputs {
		o := &outputs[i]
		target, err := filepath.Rel(dst, o.target)
		if err != nil {
			return sm.String(), err
		}

		Fprintf(sm, "<url><loc>%s/", url)

		/* There're 2 possibilities for this
		1. First is when the HTML is some/path/index.html
		<url><loc>https://example.com/some/path/</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>

		2. Then there is when the HTML is some/path/page.html
		<url><loc>https://example.com/some/path/page.html</loc><lastmod>2024-10-04</lastmod><priority>1.0</priority></url>
		*/

		base := filepath.Base(target)
		switch base {
		case "index.html":
			d := filepath.Dir(target)
			if d != "." {
				Fprintf(sm, "%s/", d)
			}
		default:
			sm.WriteString(target)
		}

		Fprintf(sm, "><lastmod>%s</lastmod><priority>1.0</priority></url>\n", dateStr)
	}

	sm.WriteString("</urlset>\n")
	return sm.String(), nil
}

// DotFiles returns content of ${dst}/.files
func DotFiles(src string, files []string) (string, error) {
	list := bytes.NewBuffer(nil)
	for _, f := range files {
		if f == "" {
			panic("found empty file for .files")
		}
		rel, err := filepath.Rel(src, f)
		if err != nil {
			return "", err
		}

		Fprintf(list, "./%s\n", rel)
	}
	return list.String(), nil
}
