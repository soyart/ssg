package soyweb_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	. "github.com/soyart/ssg/soyweb"
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
				{
					"target": "johndoe.com/src/style.css",
					"force": true
				},
				{
					"target": "johndoe.com/src/style-copy-0.css",
					"force": true
				},
				"johndoe.com/src/style-copy-1.css"
			]
		},
		"replaces": {
			"replace-me-0": "replaced-text-0",
			"replace-me-1": {
				"text": "replaced-text-1",
				"count": 3
			}
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
			Copies: map[string]CopyTargets{
				"./assets/some.txt": {
					{Target: "johndoe.com/src/some-txt.txt", Force: false},
				},
				"./assets/some": {
					{Target: "johndoe.com/src/drop", Force: true},
				},
				"./assets/style.css": {
					{Target: "johndoe.com/src/style.css", Force: true},
					{Target: "johndoe.com/src/style-copy-0.css", Force: true},
					{Target: "johndoe.com/src/style-copy-1.css", Force: false},
				},
			},
			Replaces: map[string]ReplaceTarget{
				"replace-me-0": {
					Text: "replaced-text-0", Count: 0,
				},
				"replace-me-1": {
					Text: "replaced-text-1", Count: 3,
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
			t.Logf("dsts actual:   %+v", dsts)
			t.Logf("dsts expected: %+v", dstsExpected)

			t.Fatalf("unexpected len for copies[%s] %d, expecting=%d", src, len(dsts), len(dstsExpected))
		}
		for i, dstExpected := range dstsExpected {
			dst := dsts[i]

			if dst != dstExpected {
				t.Logf("dst actual   %+v", dst)
				t.Logf("dst expected %+v", dstExpected)

				t.Fatalf("unexpected value at copies[%s][%d]: actual=%+v, expecting=%+v", src, i, dst, dstExpected)
			}
		}
	}

	for keyword, replaceExpected := range expected.Replaces {
		replace, ok := site.Replaces[keyword]
		if !ok {
			t.Log("actual.replaces", site.Replaces)
			t.Fatalf("missing replaces[%s]", keyword)
		}
		if replace != replaceExpected {
			t.Fatalf("unexpected value for replaces[%s]: actual=%+v, expecting=%+v", keyword, replace, replaceExpected)
		}
	}

	t.Run("error on invalid json manifest", func(t *testing.T) {
		tests := []string{
			`{
				"johndoe.com": {
					"name": "JohnDoe.com",
					"url": "https://johndoe.com",
					"dst": "johndoe.com/dst",
					"copies": [1]
				}
			}`,
			// Invalid copies
			`{
				"johndoe.com": {
					"name": "JohnDoe.com",
					"url": "https://johndoe.com",
					"src": "johndoe.com/src",
					"dst": "johndoe.com/dst",
					"copies": [1]
				}
			}`,
			// Invalid cleanup
			`{
				"johndoe.com": {
					"name": "JohnDoe.com",
					"url": "https://johndoe.com",
					"src": "johndoe.com/src",
					"dst": "johndoe.com/dst",
					"cleanup": "johndoe.com/cleanhere"
				}
			}`,
			// Invalid copies
			`{
				"johndoe.com": {
					"name": "JohnDoe.com",
					"url": "https://johndoe.com",
					"src": "johndoe.com/src",
					"dst": "johndoe.com/dst",
					"cleanup": false,
					"copies": [
						{
							"force": true,
							"target": "johndoe.com/src/drop"
						}
					]
				}
			}`,
			// Invalid replaces.count
			`{
				"johndoe.com": {
					"name": "JohnDoe.com",
					"url": "https://johndoe.com",
					"src": "johndoe.com/src",
					"dst": "johndoe.com/dst",
					"cleanup": false,
					"replaces": {
						"replace-me-0": "new-text-0",
						"replace-me-1": {
							"text": "new-text-1",
							"count": "1"
						}
					}
				}
			}`,

			// Negative count -1
			`{
				"johndoe.com": {
					"name": "JohnDoe.com",
					"url": "https://johndoe.com",
					"src": "johndoe.com/src",
					"dst": "johndoe.com/dst",
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
					},
					"replaces": {
						"replace-me-0": "new-text-0",
						"replace-me-1": {
							"text": "new-text-1",
							"count": -1
						}
					}
				}
			}`,
		}

		for i, s := range tests {
			var dummy any
			err := json.Unmarshal([]byte(s), &dummy)
			if err != nil {
				panic(err) // check that it's a valid json, except for empty string
			}

			var m Manifest
			err = json.Unmarshal([]byte(s), &m)
			if err != nil {
				t.Logf("[ok] invalids[%d] got expected error: %v", i, err)
				continue
			}
			t.Fatalf("invalids[%d] unexpected nil error: %v", i, err)
		}
	})
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
		newDirsBoth  []string            // new dirs  (in src and dst)
		newFilesBoth []string            // new files (in src and dst)
		newFilesDst  []string            // new files (in dst only)
		containsDst  map[string][]string // if <key> file should contain <val> bytes (in dst only)
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
			newFilesDst: []string{
				"/testreplace/_index.soyweb", // Index generator not enabled
				"/testreplace/testreplace1.html",
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
				},
				"generate-index": true
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
			newFilesDst: []string{
				"/testreplace/index.html", // Index generator enabled
				"/testreplace/testreplace1.html",
			},
			containsDst: map[string][]string{
				"/testreplace/index.html": {
					"replace-me-0",
				},
				"/testreplace/testreplace1.html": {
					"replace-me-0",
				},
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
				},
				"generate-index": true,
				"replaces": {
					"replace-me-0": "replaced-text-0",
					"replace-me-1": {
						"text": "replaced-text-1",
						"count": 3
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
			newFilesDst: []string{
				"/testreplace/index.html", // Index generator enabled
				"/testreplace/testreplace1.html",
			},
			containsDst: map[string][]string{
				"/testreplace/index.html": {
					"replaced-text-0",
					"replaced-text-1",
					"${{ replace-me-  }}", // unreplaced
				},
				"/testreplace/testreplace1.html": {
					"replaced-text-0",
					"replaced-text-1",
					`<strong>replaced-text-0</strong>`,
					"${{ replace-me-1 }}",
				},
			},
		},
	}

	for i := range tests {
		t.Logf("TestManifest Case=%d", i)
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

		t.Logf("[case %d] generate-index=%v", i, m.GenerateIndex)

		copies := make([]string, len(tc.copies))
		for i := range copies {
			copies[i] = prefix(tc.dir, tc.copies[i])
		}
		for i := range copies {
			assertExists(t, m.Copies, copies[i])
		}

		src, dst := m.Src(), m.Dst()

		err = os.RemoveAll(dst)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("[case %d] cannot remove dst '%s': %v", i, dst, err)
		}

		err = ApplyManifestV2(manifests, FlagsV2{}, StageAll)
		if err != nil {
			t.Fatalf("[case %d] error building manifest: %v", i, err)
		}
		for _, dir := range tc.newDirsBoth {
			assertFs(t, prefix(src, dir), true)
			assertFs(t, prefix(dst, dir), true)
		}
		for _, file := range tc.newFilesBoth {
			assertFs(t, prefix(src, file), false)
			assertFs(t, prefix(dst, file), false)
		}
		for _, file := range tc.newFilesDst {
			assertFs(t, prefix(dst, file), false)
		}
		for path, matches := range tc.containsDst {
			b, err := os.ReadFile(prefix(dst, path))
			if err != nil {
				t.Fatal(err)
			}
			for j, s := range matches {
				if bytes.Contains(b, []byte(s)) {
					continue
				}
				t.Logf("file:\n%s", string(b))
				t.Fatalf("[case %d][matches %d] unmatched '%s' in file '%s'", i, j, s, path)
			}
		}

		err = os.RemoveAll(dst)
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
			original: StageAll,
			skips:    []Stage{StageAll},
			oks:      []Stage{},
		},
		{
			original: StageAll,
			skips:    []Stage{StageBuild},
			oks:      []Stage{StageCleanUp, StageCopy},
		},
		{
			original: StageAll,
			skips:    []Stage{StageBuild, StageCopy},
			oks:      []Stage{StageCleanUp},
		},
		{
			original: StageAll,
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
		t.Fatalf("failed to stat path '%s': %v", p, err)
	}
	if dir != stat.IsDir() {
		t.Fatalf("expecting isDir=%v, got=%v for path='%s", dir, stat.IsDir(), p)
	}
}
