package ssg

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func generateCaching(s *Ssg) error {
	// Reset
	s.cache = nil
	stat, err := os.Stat(s.Src)
	if err != nil {
		return err
	}
	dist, err := s.buildV2()
	if err != nil {
		return err
	}
	err = writeOutFromCache(s)
	if err != nil {
		return err
	}
	err = GenerateMetadata(s.Src, s.Dst, s.Url, s.files, dist, stat.ModTime())
	if err != nil {
		return err
	}

	s.pront(len(dist) + 2)
	return nil
}

// writeOutFromCache blocks and concurrently writes out s.writes
// to their target locations.
//
// If targets is empty, writeOutFromCache writes to s.dst
func writeOutFromCache(s *Ssg) error {
	_, err := os.Stat(s.Dst)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.Dst, os.ModePerm)
	}
	if err != nil {
		return err
	}
	err = WriteOut(s.cache, s.writers)
	if err != nil {
		return err
	}
	return nil
}

// TestGenerateStreaming tests that all files are properly flushed to destination when streaming,
// and that all outputs are identical
func TestGenerateStreaming(t *testing.T) {
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

	// Generate with streaming
	streaming := New(src, dstStreaming, title, url)
	streaming.With(Writers(uint(WritersDefault)))

	// Generate without streaming, and with caching
	// (old v2 flow)
	caching := New(src, dst, title, url)
	caching.With(
		Caching(),
		Writers(uint(WritersDefault)),
	)

	testCmpDeepEqual(t, &caching, &streaming)
}

func testCmpDeepEqual(t *testing.T, s1, s2 *Ssg) {
	err := os.RemoveAll(s1.Dst)
	if err != nil {
		panic(err)
	}
	err = os.RemoveAll(s2.Dst)
	if err != nil {
		panic(err)
	}
	err = s1.Generate()
	if err != nil {
		panic(err)
	}
	err = s2.Generate()
	if err != nil {
		panic(err)
	}

	testDeepEqual(t, s1.Dst, s2.Dst)
}

func testDeepEqual(t *testing.T, dst1, dst2 string) {
	fn := deepEqual(t, dst1, dst2)
	err := filepath.WalkDir(dst1, fn)
	if err != nil {
		panic(err)
	}

	err = os.RemoveAll(dst1)
	if err != nil {
		panic(err)
	}
	err = os.RemoveAll(dst2)
	if err != nil {
		panic(err)
	}
}

func deepEqual(t *testing.T, dst1, dst2 string) fs.WalkDirFunc {
	// Walk on s1, and compare with the same output from s2
	return func(path1 string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dst1, path1)
		if err != nil {
			panic(err)
		}

		path2 := filepath.Join(dst2, rel)
		if d.IsDir() {
			entries, err := os.ReadDir(path1)
			if err != nil {
				panic(err)
			}
			entries2, err := os.ReadDir(path2)
			if err != nil {
				return err
			}

			if l, ls := len(entries), len(entries2); l != ls {
				for i := range entries {
					t.Logf("expected entry for %s: %s", path1, entries[i].Name())
				}
				t.Fatalf("unexpected len of entries in '%s': expected=%d, actual=%d", path2, l, ls)
			}
			for i := range entries {
				name1 := entries[i].Name()
				name2 := entries2[i].Name()

				if name1 != name2 {
					t.Fatalf("unexpected filename: expected='%s', actual='%s'", name1, name2)
				}
			}

			return nil
		}

		stat1, err := os.Stat(path1)
		if err != nil {
			panic(err)
		}
		stat2, err := os.Stat(path2)
		if err != nil {
			t.Fatalf("unexpected error from stat '%s': %v", path2, err)
		}
		if sz, szStreaming := stat1.Size(), stat2.Size(); sz != szStreaming {
			t.Fatalf("unexpected size from '%s': expected=%d, actual=%d", path2, sz, szStreaming)
		}

		bytesExpected, err := os.ReadFile(path1)
		if err != nil {
			panic(err)
		}
		bytesStreaming, err := os.ReadFile(path2)
		if err != nil {
			t.Fatalf("unexpected error from reading '%s'", path2)
		}
		if !bytes.Equal(bytesExpected, bytesStreaming) {
			t.Fatalf("unexpected bytes from '%s'", path2)
		}

		return nil
	}
}
