package ssg

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

const (
	manifestJSON = `
{
	"johndoe.com": {
		"name": "JohnDoe.com",
		"url": "https://johndoe.com",
		"src": "testdata/johndoe.com/src",
		"dst": "testdata/johndoe.com/dst",
		"cleanup": true,
		"copies": {
			"testdata/assets/style.css": {
				"target": "testdata/johndoe.com/src/style.css",
				"force": true
			},
			"testdata/assets/some.txt": "testdata/johndoe.com/src/some-txt.txt",
			"testdata/assets/some": {
				"force": true,
				"target": "testdata/johndoe.com/src/drop"
			}
		}
	}
}`
)

func prefix(p1, p2 string) string {
	return fmt.Sprintf("%s/%s", p1, p2)
}

func TestManifest(t *testing.T) {
	var manifests Manifest
	err := json.Unmarshal([]byte(manifestJSON), &manifests)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err.Error())
	}

	const siteKey = "johndoe.com"
	m, ok := manifests[siteKey]
	if !ok {
		t.Fatalf("missing manifest for siteKey '%s'", siteKey)
	}

	dir := "testdata"
	copies := []string{
		prefix(dir, "assets/style.css"),
		prefix(dir, "assets/some.txt"),
		prefix(dir, "assets/some"),
	}
	for i := range copies {
		assertExists(t, m.Copies, copies[i])
	}

	err = os.RemoveAll(m.Dst)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("cannot remove dst '%s': %v", m.Dst, err)
	}

	err = Apply(manifests, StagesAll)
	if err != nil {
		t.Fatalf("error building manifest: %v", err)
	}

	cpDirs := []string{
		"/drop",
	}
	cpFiles := []string{
		"/style.css",
		"/drop/fonts/fake-font",
		"/drop/fonts/fake-font-bold",
	}

	for i := range cpDirs {
		dir := cpDirs[i]
		assertFs(t, prefix(m.Src, dir), true)
		assertFs(t, prefix(m.Dst, dir), true)
	}
	for i := range cpFiles {
		file := cpFiles[i]
		assertFs(t, prefix(m.Src, file), false)
		assertFs(t, prefix(m.Dst, file), false)
	}

	err = os.RemoveAll(m.Dst)
	if err != nil {
		t.Logf("failed to cleaning up directory after")
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
		t.Fatalf("expecting isDir=%v, got=%v", dir, stat.IsDir())
	}
}
