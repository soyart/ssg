package soyweb

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestManifestUnmarshal(t *testing.T) {
	s := `{
	"johndoe.com": {
		"name": "JohnDoe.com",
		"url": "https://johndoe.com",
		"src": "johndoe.com/src",
		"dst": "johndoe.com/dst",
		"cleanup": true,
		"generate_blog": true,
		"copies": {
			"./assets/some.txt": "johndoe.com/src/some-txt.txt",
			"./assets/some": {
				"force": true,
				"target": "johndoe.com/src/drop"
			},
			"./assets/style.css": [
				"johndoe.com/src/style-copy-0.css",
				{
					"target": "johndoe.com/src/style-copy-1.css",
					"force": true
				},
				{
					"target": "johndoe.com/src/style-copy-2.css",
					"force": true
				}
			]
		}
	}
}`

	var m Manifest
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	manifestExpected := Manifest{
		"johndoe.com": Site{
			Copies: map[string][]WriteTarget{
				"./assets/some.txt": {
					{Target: "johndoe.com/src/some-txt.txt", Force: false},
				},
				"./assets/some": {
					{Target: "johndoe.com/src/drop", Force: true},
				},
				"./assets/style.css": {
					{Target: "johndoe.com/src/style-copy-0.css", Force: false},
					{Target: "johndoe.com/src/style-copy-1.css", Force: true},
					{Target: "johndoe.com/src/style-copy-2.css", Force: true},
				},
			},
			CleanUp:       true,
			GenerateIndex: true,
		},
	}

	site := m["johndoe.com"]
	expected := manifestExpected["johndoe.com"]

	if site.CleanUp != expected.CleanUp {
		t.Fatalf("unexpected cleanup %v, expecting=%v", site.CleanUp, expected.CleanUp)
	}

	for src, dstsExpected := range expected.Copies {
		dsts, ok := site.Copies[src]
		if !ok {
			t.Fatalf("missing copies[%s]", src)
		}
		if len(dsts) != len(dstsExpected) {
			t.Logf("dsts actual: %+v", dsts)
			t.Logf("dsts expected: %+v", dstsExpected)

			t.Fatalf("unexpected len for copies[%s] %d, expecting=%d", src, len(dsts), len(dstsExpected))
		}
		for i, dstExpected := range dstsExpected {
			dst := dsts[i]

			if dst != dstExpected {
				t.Logf("dst actual %+v", dst)
				t.Logf("dst expected %+v", dstExpected)

				t.Fatalf("unexpected value at copies[%s][%d]: actual=%+v, expecting=%+v", src, i, dst, dstExpected)
			}
		}
	}
}

func prefix(p1, p2 string) string {
	return fmt.Sprintf("%s/%s", p1, p2)
}

func TestManifest(t *testing.T) {
	type testCase struct {
		manifestJSON string
		siteKey      string
		dir          string
		copies       []string
		newDirsBoth  []string // new dirs in src and dst
		newFilesBoth []string // new files in src and dst
	}

	tests := []testCase{
		{
			manifestJSON: `
{
	"johndoe.com": {
		"name": "JohnDoe.com",
		"url": "https://johndoe.com",
		"src": "../testdata/johndoe.com/src",
		"dst": "../testdata/johndoe.com/dst",
		"cleanup": true,
		"copies": {
			"../testdata/assets/style.css": {
				"target": "../testdata/johndoe.com/src/style.css",
				"force": true
			},
			"../testdata/assets/some.txt": "../testdata/johndoe.com/src/some-txt.txt",
			"../testdata/assets/some": {
				"force": true,
				"target": "../testdata/johndoe.com/src/drop"
			}
		}
	}
}`,
			siteKey: "johndoe.com",
			dir:     "../testdata",
			copies: []string{
				"assets/style.css",
				"assets/some.txt",
				"assets/some",
			},
			newDirsBoth: []string{
				"/drop",
			},
			newFilesBoth: []string{
				"/style.css",
				"/some-txt.txt",
				"/drop/nested/path/some.env",
				"/drop/fonts/fake-font.ttf",
				"/drop/fonts/fake-font-bold.ttf",
			},
		},
		{
			manifestJSON: `
{
	"johndoe.com": {
		"name": "JohnDoe.com",
		"url": "https://johndoe.com",
		"src": "../testdata/johndoe.com/src",
		"dst": "../testdata/johndoe.com/dst",
		"cleanup": true,
		"copies": {
			"../testdata/assets/style.css": {
				"target": "../testdata/johndoe.com/src/style.css",
				"force": true
			},
			"../testdata/assets/some.txt": "../testdata/johndoe.com/src/some-txt.txt",
			"../testdata/assets/some/fonts": {
				"force": true,
				"target": "../testdata/johndoe.com/src/drop"
			}
		}
	}
}`,
			siteKey: "johndoe.com",
			dir:     "../testdata",
			copies: []string{
				"assets/style.css",
				"assets/some/fonts",
			},
			newDirsBoth: []string{
				"/drop",
			},
			newFilesBoth: []string{
				"/style.css",
				"/some-txt.txt",
				"/drop/fake-font.ttf",
				"/drop/fake-font-bold.ttf",
			},
		},
		{
			manifestJSON: `
{
	"johndoe.com": {
		"name": "JohnDoe.com",
		"url": "https://johndoe.com",
		"src": "../testdata/johndoe.com/src",
		"dst": "../testdata/johndoe.com/dst",
		"cleanup": true,
		"copies": {
			"../testdata/assets/style.css": {
				"target": "../testdata/johndoe.com/src/style.css",
				"force": true
			},
			"../testdata/assets/some.txt": "../testdata/johndoe.com/src/debug/some-txt.txt",
			"../testdata/assets/some/nested/path/some.env": "../testdata/johndoe.com/src/assets/env",
			"../testdata/assets/some/fonts": {
				"force": true,
				"target": "../testdata/johndoe.com/src/assets"
			}
		}
	}
}`,
			siteKey: "johndoe.com",
			dir:     "../testdata",
			copies: []string{
				"assets/style.css",
				"assets/some/fonts",
			},
			newDirsBoth: []string{
				"/assets",
				"/debug",
			},
			newFilesBoth: []string{
				"/style.css",
				"/debug/some-txt.txt",
				"/assets/env",
				"/assets/fake-font.ttf",
				"/assets/fake-font-bold.ttf",
			},
		},
	}

	for i := range tests {
		tc := &tests[i]
		var manifests Manifest
		err := json.Unmarshal([]byte(tc.manifestJSON), &manifests)
		if err != nil {
			t.Fatalf("[case %d] failed to parse JSON: %v", i, err.Error())
		}

		m, ok := manifests[tc.siteKey]
		if !ok {
			t.Fatalf("[case %d] missing manifest for siteKey '%s'", i, tc.siteKey)
		}

		copies := make([]string, len(tc.copies))
		for i := range copies {
			copies[i] = prefix(tc.dir, tc.copies[i])
		}

		for i := range copies {
			assertExists(t, m.Copies, copies[i])
		}

		err = os.RemoveAll(m.ssg.Dst)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("[case %d] cannot remove dst '%s': %v", i, m.ssg.Dst, err)
		}

		err = ApplyManifest(manifests, StagesAll)
		if err != nil {
			t.Fatalf("[case %d] error building manifest: %v", i, err)
		}

		for i := range tc.newDirsBoth {
			dir := tc.newDirsBoth[i]
			assertFs(t, prefix(m.ssg.Src, dir), true)
			assertFs(t, prefix(m.ssg.Dst, dir), true)
		}
		for i := range tc.newFilesBoth {
			file := tc.newFilesBoth[i]
			assertFs(t, prefix(m.ssg.Src, file), false)
			assertFs(t, prefix(m.ssg.Dst, file), false)
		}

		err = os.RemoveAll(m.ssg.Dst)
		if err != nil {
			t.Logf("[case %d] failed to cleaning up directory after", i)
			break
		}
	}
}

func TestStages(t *testing.T) {
	type testCase struct {
		original Stage
		skips    []Stage
		oks      []Stage
	}

	tests := []testCase{
		{
			original: Stage(0),
			skips:    []Stage{StageBuild, StageBuild, StageCleanUp},
			oks:      []Stage{},
		},
		{
			original: StageBuild,
			skips:    []Stage{StageBuild, StageBuild, StageCleanUp},
			oks:      []Stage{},
		},
		{
			original: StageBuild,
			skips:    []Stage{StageCleanUp},
			oks:      []Stage{StageBuild},
		},
		{
			original: StagesAll,
			skips:    []Stage{StagesAll},
			oks:      []Stage{},
		},
		{
			original: StagesAll,
			skips:    []Stage{StageBuild},
			oks:      []Stage{StageCleanUp, StageCopy},
		},
		{
			original: StagesAll,
			skips:    []Stage{StageBuild, StageCopy},
			oks:      []Stage{StageCleanUp},
		},
		{
			original: StagesAll,
			skips:    []Stage{StageCopy},
			oks:      []Stage{StageCleanUp, StageBuild},
		},
	}

	for i := range tests {
		tc := &tests[i]
		tc.original.Skip(tc.skips...)
		if !tc.original.Ok(tc.oks...) {
			t.Fatalf("unexpected result for test case %d", i+1)
		}
	}
}

func assertExists[K comparable, V any](t *testing.T, m map[K]V, k K) {
	_, ok := m[k]
	if !ok {
		t.Logf("map: %v", m)
		t.Fatalf("map is missing key %v", k)
	}
}

func assertFs(t *testing.T, p string, dir bool) {
	stat, err := os.Stat(p)
	if err != nil {
		t.Fatalf("failed to stat path '%s'", p)
	}

	if dir != stat.IsDir() {
		t.Fatalf("expecting isDir=%v, got=%v for path='%s", dir, stat.IsDir(), p)
	}
}
