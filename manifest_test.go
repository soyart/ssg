package ssg

import (
	"os"
	"testing"
)

func TestManifest(t *testing.T) {
	filename := "./manifest.json"
	siteKey := "johndoe.com"

	manifests, err := NewManifest(filename)
	if err != nil {
		t.Fatalf("unexpected error when opening test manifest file: %v", err)
	}
	m, ok := manifests[siteKey]
	if !ok {
		t.Fatalf("missing manifest for siteKey '%s'", siteKey)
	}

	copies := []string{
		"./assets/style.css",
		"./assets/some.txt",
		"./assets/some",
	}
	for i := range copies {
		assertExists(t, m.Copies, copies[i])
	}

	err = os.RemoveAll(m.Dst)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("cannot remove dst '%s': %v", m.Dst, err)
	}

	err = BuildManifestFromPath(filename, StagesAll)
	if err != nil {
		t.Fatalf("error building manifest: %v", err)
	}

	// Test copies
	src := "johndoe.com/src"
	dst := "johndoe.com/dst"
	cpDirs := []string{
		"/drop",
	}
	cpFiles := []string{
		"/style.css",
		"/drop/fonts/fake-font",
		"/drop/fonts/fake-font-bold",
	}

	for i := range cpDirs {
		assertFs(t, src+cpDirs[i], true)
		assertFs(t, dst+cpDirs[i], true)
	}
	for i := range cpFiles {
		assertFs(t, src+cpFiles[i], false)
		assertFs(t, dst+cpFiles[i], false)
	}

	err = os.RemoveAll(m.Dst)
	if err != nil {
		t.Logf("failed to cleaning up directory after")
	}
}

func assertExists[K comparable, V any](t *testing.T, m map[K]V, k K) {
	_, ok := m[k]
	if !ok {
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
