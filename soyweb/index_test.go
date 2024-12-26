package soyweb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soyart/ssg"
)

func TestGenerateIndex(t *testing.T) {
	src := "./testdata/myblog/src"
	dst := "./testdata/myblog/dst-test-generate-index"
	title := "TestTitle"
	url := "https://my.blog"

	err := os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}

	defaultTitleHtml := fmt.Sprintf("<title>%s</title>", title)
	contains := map[string][]string{
		"_index.soyweb": {
			`<title>My blog (title tag)</title>`,
			`<li><p><a href="/2023/">2023</a></p></li>`,
			`<li><p><a href="/2022/">2022</a></p></li>`,
			`<li><p><a href="/testdir/">testdir</a></p></li>`,
		},
		"2022/_index.soyweb": {
			`<title>twenty and twenty-two (from-tag)</title>`,
			`<li><p><a href="/2022/bar/">bar</a></p></li>`,
			`<li><p><a href="/2022/foo.html">Foo</a></p></li>`,
		},
		"2023/_index.soyweb": {
			defaultTitleHtml,
			`Index of 2023`, // because _index.soyweb was empty
			`<li><p><a href="/2023/baz.html">Bazketball</a></p></li>`,
			`<li><p><a href="/2023/recurse/">recurse</a></p></li>`,
			`<li><p><a href="/2023/lol/">LOLOLOL</a></p></li>`,
		},
		"2023/recurse/_index.soyweb": {
			defaultTitleHtml,
			`<h1 id="recurse-index">Recurse Index</h1>`,
			"<li><p><a href=\"/2023/recurse/r1/\">Recursive 1</a></p></li>",
			"<li><p><a href=\"/2023/recurse/r2/\">Recursive 2</a></p></li>",
		},
		"testdir/_index.soyweb": {
			`<title>testdir Title from tag</title>`,
			`<li><p><a href="/testdir/dir1/">Dir-1-Title-From-Tag</a></p></li>`,
			`<li><p><a href="/testdir/dir2/">Dir-2</a></p></li>`,
			`<li><p><a href="/testdir/testprefer/">testprefer</a></p></li>`,
		},
	}

	notContains := map[string][]string{
		"_index.soyweb": {
			"ignore1",
			"ignore2",
		},
		"testdir/_index.soyweb": {
			"ignore3",
			"ignore4",
		},
	}

	// Sanity checks
	for marker := range contains {
		markerPath := filepath.Join(src, marker)
		assertFs(t, markerPath, false)

		index := formatIndexPath(markerPath)
		_, err := os.Stat(index)
		if err == nil {
			t.Fatalf("unexpected index.html before generator runs")
		}
	}

	err = ssg.Generate(src, dst, title, url, IndexGenerator())
	if err != nil {
		t.Fatalf("error during ssg generation: %v", err)
	}

	t.Run("should contain expected generated HTML", func(t *testing.T) {
		for marker, entries := range contains {
			markerPath := filepath.Join(dst, marker)
			index := formatIndexPath(markerPath)
			assertFs(t, index, false)

			content, err := os.ReadFile(index)
			if err != nil {
				t.Fatalf("failed to read back index %s: %v", index, err)
			}
			for i := range entries {
				entry := entries[i]
				if strings.Contains(string(content), entry) {
					continue
				}

				t.Fatalf("missing #%d entry '%s' in %s", i+1, entry, index)
			}
		}
	})

	t.Run("ssgignore should work", func(t *testing.T) {
		for marker, entries := range notContains {
			markerPath := filepath.Join(dst, marker)
			index := formatIndexPath(markerPath)
			assertFs(t, index, false)

			content, err := os.ReadFile(index)
			if err != nil {
				t.Fatalf("failed to read back index %s: %v", index, err)
			}
			for i := range entries {
				entry := entries[i]
				if !strings.Contains(string(content), entry) {
					continue
				}

				t.Fatalf("unexpected #%d entry '%s' in %s", i+1, entry, index)
			}
		}
	})

	err = os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}
}

func formatIndexPath(marker string) string {
	marker = filepath.Dir(marker)
	return filepath.Join(marker, "index.html")
}
