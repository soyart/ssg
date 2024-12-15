package soyweb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soyart/ssg"
)

func TestGenerateIndex(t *testing.T) {
	src := "./testdata/myblog/src"
	dst := "./testdata/myblog/dst"

	err := os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}

	markers := map[string][]string{
		"_index.soyweb": {
			`<li><p><a href="/2023/">2023</a></p></li>`,
			`<li><p><a href="/2022/">2022</a></p></li>`,
		},
		"2022/_index.soyweb": {
			`<li><p><a href="/2022/bar/">bar</a></p></li>`,
			`<li><p><a href="/2022/foo.html">Foo</a></p></li>`,
		},
		"2023/_index.soyweb": {
			`<li><p><a href="/2023/baz.html">Bazketball</a></p></li>`,
			`<li><p><a href="/2023/lol/">LOLOLOL</a></p></li>`,
		},
	}

	for marker := range markers {
		markerPath := filepath.Join(src, marker)
		assertFs(t, markerPath, false)

		index := toGenerated(markerPath)
		_, err := os.Stat(index)
		if err == nil {
			t.Fatalf("unexpected index.html before generator runs")
		}
	}

	err = ssg.Generate(src, dst, "TestTitle", "https://my.blog", IndexGenerator())
	if err != nil {
		t.Fatalf("error during ssg generation: %v", err)
	}

	for marker, entries := range markers {
		markerPath := filepath.Join(dst, marker)
		index := toGenerated(markerPath)
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

	err = os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}
}

func toGenerated(s string) string {
	s = filepath.Dir(s)
	return filepath.Join(s, "index.html")
}
