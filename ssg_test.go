package ssg

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepare(t *testing.T) {
	src := "./testdata/johndoe.com/src"
	ignores, err := prepare(src, "some/destination")
	if err != nil {
		t.Errorf("unexpected error from prepare: %v", err)
	}

	expected := []string{
		"testignore/ignored.md",
		"testignore/ignoreroot",
	}

	if lenExpected, lenActual := len(expected), len(ignores); lenExpected != lenActual {
		t.Fatalf("unexpected length of ssgignore (expecting %d): %d", lenExpected, lenActual)
	}
	for i := range expected {
		ignored := expected[i]
		if ignores.contains(filepath.Join(src, ignored)) {
			continue
		}

		t.Fatalf("expecting %s in ssgignore", ignored)
	}
}

func TestScan(t *testing.T) {
	root := "./testdata/johndoe.com"
	src := filepath.Join(root, "/src")
	dst := filepath.Join(root, "/dst")
	title := "JohnDoe.com"
	url := "https://johndoe.com"

	ssg := New(src, dst, title, url)
	err := filepath.WalkDir(root, ssg.scan)
	if err != nil {
		t.Errorf("unexpected error from scan: %v", err)
	}

	if !ssg.preferred.contains(filepath.Join(src, "/blog/index.html")) {
		t.Fatalf("missing preferred html file /blog/index.html")
	}

	for i := range ssg.dist {
		o := &ssg.dist[i]

		if strings.HasSuffix(o.target, "_header.html") {
			t.Fatalf("unexpected _header.html output in '%s'", o.target)
		}
		if strings.HasSuffix(o.target, "_footer.html") {
			t.Fatalf("unexpected _footer.html output in '%s'", o.target)
		}
	}

	headers := map[string]from{
		"/_header.html":           fromH1,
		"/blog/_header.html":      fromTag,
		"/blog/2023/_header.html": fromNone,
	}

	for h, from := range headers {
		filename := filepath.Join(src, h)
		dirname := filepath.Dir(filename)

		header, ok := ssg.headers.perDir.values[dirname]
		if !ok {
			t.Fatalf("missing header '%s' for dir '%s'", filename, dirname)
		}

		if header.titleFrom != from {
			t.Fatalf("unexpected from '%d', expecting %d", header.titleFrom, from)
		}
	}
}

func TestTitleFromH1(t *testing.T) {
	type testCase struct {
		head     string
		markdown string
		expected string
	}

	tests := []testCase{
		{
			head: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-h1}}</title>
</head>`,
			markdown: `
Mar 24 1998

# Some h1

Some para`,
			expected: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Some h1</title>
</head>`,
		},
		{
			head: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-h1}}</title>
</head>`,
			markdown: `
Mar 24 1998

## Some h2

# Some h1

Some para`,
			expected: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Some h1</title>
</head>`,
		},
	}

	for i := range tests {
		tc := &tests[i]
		actual := titleFromH1([]byte{}, []byte(tc.head), []byte(tc.markdown))
		if !bytes.Equal(actual, []byte(tc.expected)) {
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expected)
			t.Logf("Actual:\nSTART===\n%s\nEND===", actual)

			t.Fatalf("unexpected value for case %d", i+1)
		}
	}
}

func TestTitleFromTag(t *testing.T) {
	type testCase struct {
		head             string
		markdown         string
		expectedHead     string
		expectedMarkdown string
	}

	tests := []testCase{
		{
			head: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-tag}}</title>
</head>`,
			markdown: `
Mar 24 1998

:title My title

# Some h1

Some para`,
			expectedHead: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>My title</title>
</head>`,
			expectedMarkdown: `
Mar 24 1998

# Some h1

Some para`,
		},
		{
			head: `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-tag}}</title>
</head>`,
			markdown: `
Mar 24 1998

	:title Not actually title

:title This is the title

# Some h1

Some para  `,
			expectedHead: `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>This is the title</title>
</head>`,
			expectedMarkdown: `
Mar 24 1998

	:title Not actually title

# Some h1

Some para  `,
		},
	}

	for i := range tests {
		tc := &tests[i]
		head, markdown := titleFromTag([]byte{}, []byte(tc.head), []byte(tc.markdown))
		if !bytes.Equal(head, []byte(tc.expectedHead)) {
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expectedHead)
			t.Logf("Actual:\nSTART===\n%s\nEND===", head)
			t.Logf("len(expected) = %d, len(actual) = %d", len(tc.expectedHead), len(head))

			t.Fatalf("unexpected substituted header value for case %d", i+1)
		}

		if md := string(markdown); md != tc.expectedMarkdown {
			t.Logf("Original:\nSTART===\n%s\nEND===", tc.markdown)
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expectedMarkdown)
			t.Logf("Actual:\nSTART===\n%s\nEND===", md)
			t.Logf("len(expected) = %d, len(actual) = %d", len(tc.expectedMarkdown), len(markdown))

			for i := range tc.expectedMarkdown {
				e := tc.expectedMarkdown[i]
				a := md[i]
				t.Logf("%d: diff=%v actual='%c', expected='%c'", i, e != a, e, a)
			}

			t.Fatalf("unexpected modified markdown value for case %d", i+1)
		}
	}
}
