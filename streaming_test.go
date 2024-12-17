package ssg

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestStreaming tests that all files are properly flushed to destination when streaming,
// and that all outputs are identical
func TestStreaming(t *testing.T) {
	root := "./soyweb/testdata/johndoe.com"
	src := filepath.Join(root, "/src")
	dst := filepath.Join(root, "/dst")
	dstStreaming := filepath.Join(root, "/dstStreaming")
	title := "JohnDoe.com"
	url := "https://johndoe.com"

	err := os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}
	err = os.RemoveAll(dstStreaming)
	if err != nil {
		panic(err)
	}

	// Generate without streaming
	err = Generate(src, dst, title, url,
		Concurrent(uint(ConcurrentDefault)),
	)
	if err != nil {
		panic(err)
	}
	// Generate with streaming
	err = Generate(src, dstStreaming, title, url,
		Concurrent(uint(ConcurrentDefault)),
		Streaming(),
	)
	if err != nil {
		t.Fatalf("error generating with streaming: %v", err)
	}

	filepath.WalkDir(dst, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dst, path)
		if err != nil {
			panic(err)
		}

		pathStreaming := filepath.Join(dstStreaming, rel)

		if d.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			entriesStreaming, err := os.ReadDir(pathStreaming)
			if err != nil {
				return err
			}
			if l, ls := len(entries), len(entriesStreaming); l != ls {
				for i := range entries {
					t.Logf("expected entry for %s: %s", path, entries[i].Name())
				}
				t.Fatalf("unexpected len of entries in '%s': expected=%d, actual=%d", pathStreaming, l, ls)
			}

			for i := range entries {
				name := entries[i].Name()
				nameStreaming := entriesStreaming[i].Name()
				if name != nameStreaming {
					t.Fatalf("unexpected filename '%s'", nameStreaming)
				}
			}

			return nil
		}

		stat, err := os.Stat(path)
		if err != nil {
			panic(err)
		}
		statStreaming, err := os.Stat(pathStreaming)
		if err != nil {
			t.Fatalf("unexpected error from stat '%s': %v", pathStreaming, err)
		}
		if sz, szStreaming := stat.Size(), statStreaming.Size(); sz != szStreaming {
			t.Fatalf("unexpected size from '%s': expected=%d, actual=%d", pathStreaming, sz, szStreaming)
		}

		bytesExpected, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		bytesStreaming, err := os.ReadFile(pathStreaming)
		if err != nil {
			t.Fatalf("unexpected error from reading '%s'", pathStreaming)
		}
		if !bytes.Equal(bytesExpected, bytesStreaming) {
			t.Fatalf("unexpected bytes from '%s'", pathStreaming)
		}

		return nil
	})

	err = os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}
	err = os.RemoveAll(dstStreaming)
	if err != nil {
		panic(err)
	}
}
