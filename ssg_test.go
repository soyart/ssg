package ssg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ignore "github.com/sabhiram/go-gitignore"
)

func TestToHTML(t *testing.T) {
	type testCase struct {
		md   string
		html string
	}

	tests := []testCase{
		{
			md:   "",
			html: "",
		},
		{
			md:   "This is a paragraph",
			html: "<p>This is a paragraph</p>\n",
		},
		{
			md: `# Some h1
Some paragraph`,
			html: `<h1 id="some-h1">Some h1</h1>

<p>Some paragraph</p>
`,
		},
		{
			md: `# Some h1
Some paragraph

## Some h2`,
			html: `<h1 id="some-h1">Some h1</h1>

<p>Some paragraph</p>

<h2 id="some-h2">Some h2</h2>
`,
		},
		{
			md: `# Some h1
Some paragraph

<p>Embedded HTML paragraph</p>

## Some h2`,
			html: `<h1 id="some-h1">Some h1</h1>

<p>Some paragraph</p>

<p>Embedded HTML paragraph</p>

<h2 id="some-h2">Some h2</h2>
`,
		},
		{
			md: `# Some h1
Some paragraph

<p>Embedded HTML paragraph</p>

## Some h2

Some paragraph2`,
			html: `<h1 id="some-h1">Some h1</h1>

<p>Some paragraph</p>

<p>Embedded HTML paragraph</p>

<h2 id="some-h2">Some h2</h2>

<p>Some paragraph2</p>
`,
		},
		{
			md: `# Some h1
Some paragraph

<p>Embedded HTML paragraph</p>

<h2 id="some-h2">Embedded HTML h2</h2>

Some paragraph2`,
			html: `<h1 id="some-h1">Some h1</h1>

<p>Some paragraph</p>

<p>Embedded HTML paragraph</p>

<h2 id="some-h2">Embedded HTML h2</h2>

<p>Some paragraph2</p>
`,
		},
		{
			md: `# Some h1
Some paragraph

<p>Embedded HTML paragraph</p>

<h2>Embedded HTML h2</h2>

Some paragraph2`,
			html: `<h1 id="some-h1">Some h1</h1>

<p>Some paragraph</p>

<p>Embedded HTML paragraph</p>

<h2>Embedded HTML h2</h2>

<p>Some paragraph2</p>
`,
		},
	}

	for i := range tests {
		tc := &tests[i]
		html := ToHtml([]byte(tc.md))
		if actual := string(html); actual != tc.html {
			t.Logf("len(expected)=%d, len(actual)=%d", len(html), len(actual))
			t.Logf("expected:\n%s", tc.html)
			t.Logf("actual:\n%s", actual)
			t.Fatalf("unexpected HTML output from case %d", i+1)
		}
	}
}

func TestGenerate(t *testing.T) {
	t.Run("build-v2", func(t *testing.T) {
		testGenerate(t, func(s *Ssg) ([]OutputFile, error) {
			return s.buildV2()
		})
	})
}

func testGenerate(t *testing.T, buildFn func(s *Ssg) ([]OutputFile, error)) {
	root := "./soyweb/testdata/johndoe.com"
	src := filepath.Join(root, "/src")
	dst := filepath.Join(root, "/dst")
	title := "JohnDoe.com"
	url := "https://johndoe.com"

	err := os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}

	s := New(src, dst, title, url)
	outputs, err := buildFn(&s)
	if err != nil {
		t.Errorf("unexpected error from scan: %v", err)
	}
	if !s.preferred.ContainsAll(filepath.Join(src, "/blog/index.html")) {
		t.Fatalf("missing preferred html file /blog/index.html")
	}

	for i := range s.cache {
		o := &s.cache[i]

		if strings.HasSuffix(o.target, "_header.html") {
			t.Fatalf("unexpected _header.html output in '%s'", o.target)
		}
		if strings.HasSuffix(o.target, "_footer.html") {
			t.Fatalf("unexpected _footer.html output in '%s'", o.target)
		}
	}

	titleFroms := map[string]TitleFrom{
		"/_header.html":           TitleFromH1,
		"/blog/_header.html":      TitleFromTag,
		"/blog/2023/_header.html": TitleFromNone,
	}

	for h, from := range titleFroms {
		filename := filepath.Join(src, h)
		dirname := filepath.Dir(filename)
		header, ok := s.headers.perDir.values[dirname]
		if !ok {
			t.Fatalf("missing header '%s' for dir '%s'", filename, dirname)
		}
		if header.titleFrom != from {
			t.Fatalf("unexpected from '%d', expecting %d", header.titleFrom, from)
		}
	}

	type expected struct {
		subString string
		titleFrom TitleFrom
	}

	expecteds := map[string]expected{
		"/": {
			titleFrom: TitleFromH1,
			subString: "<!-- ROOT HEADER -->",
		},
		"/blog": {
			titleFrom: TitleFromTag,
			subString: "<!-- HEADER FOR BLOG -->",
		},
		"/blog/": {
			titleFrom: TitleFromTag,
			subString: "<!-- HEADER FOR BLOG -->",
		},
		"/blog/2022": {
			titleFrom: TitleFromTag,
			subString: "<!-- HEADER FOR BLOG -->",
		},
		"/blog/2022/3": {
			titleFrom: TitleFromTag,
			subString: "<!-- HEADER FOR BLOG -->",
		},
		"/blog/2023": {
			titleFrom: TitleFromNone,
			subString: "<!-- HEADER FOR BLOG 2023 -->",
		},
		"/notes": {
			titleFrom: TitleFromTag,
			subString: "<!-- NOTES HEADER -->",
		},
	}

	for parentDir, e := range expecteds {
		parentDir = filepath.Join(src, parentDir)
		chosen := s.headers.choose(parentDir)
		if chosen.titleFrom != e.titleFrom {
			t.Fatalf("unexpected titleFrom at dir '%s' actual=%d, expecting=%d", parentDir, chosen.titleFrom, e.titleFrom)
		}
		if !bytes.Contains(chosen.Bytes(), []byte(e.subString)) {
			t.Fatalf("missing expecting substr '%s' from dir %s", e.subString, parentDir)
		}
	}

	expectedOutputs := map[string][]string{
		"/index.html": {
			"<title>Welcome to JohnDoe.com!</title>",
		},
		"/blog/2022/index.html": {
			"<title>2022 Blog index</title>",
			"<body><h1 id=\"blog-from-the-worst-year\">Blog from the worst year</h1>",
		},
		"/testconvert/index.html": {
			"<!-- Header for testconvert -->",
			"<title>Embedded-HTML should be correctly preserved</title>",
		},
	}

	for path, e := range expectedOutputs {
		path = filepath.Join(dst, path)
		for i := range outputs {
			o := &outputs[i]
			if o.target != path {
				continue
			}
			for j := range e {
				s := e[j]
				if bytes.Contains(o.data, []byte(s)) {
					continue
				}
				t.Fatalf("missing expected substr '%s' from output %s", s, o.target)
			}
		}
	}

	err = os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}
}

// Test that the library we use actually does what we want it to
func TestSsgignore(t *testing.T) {
	type testCase struct {
		path     string
		ignores  []string
		expected bool
	}

	tests := []testCase{
		{
			ignores: []string{
				"testignore",
			},
			path:     "testignore",
			expected: true,
		},
		{
			ignores: []string{
				"testignore",
			},
			path:     "testignore/",
			expected: true,
		},
		{
			ignores: []string{
				"testignore",
			},
			path:     "testignore/one",
			expected: true,
		},
		{
			ignores: []string{
				"test*",
			},
			path:     "testignore/one",
			expected: true,
		},
		{
			ignores: []string{
				"testignore",
				"!prefix/testignore",
			},
			path:     "prefix/testignore/",
			expected: false,
		},
		{
			ignores: []string{
				"!prefix/testignore",
				"testignore",
			},
			path:     "prefix/testignore/",
			expected: true,
		},
		{
			ignores: []string{
				"testignore",
				"!testignore/important/",
			},
			path:     "testignore/important/data",
			expected: false,
		},
		{
			ignores: []string{
				"testignore",
				"!testignore/important*",
			},
			path:     "testignore/important/data",
			expected: false,
		},
		{
			ignores: []string{
				"testignore/trash/**",
				"#!testignore/trash/**/keep", // Comment
			},
			path:     "testignore/trash/some/path/keep/data",
			expected: true,
		},
	}

	for i := range tests {
		tc := &tests[i]
		ignores := ignore.CompileIgnoreLines(tc.ignores...)
		if ignores == nil {
			panic("bad ignore lines")
		}

		ignorer := &ignorerGitignore{GitIgnore: ignores}
		ignored := ignorer.ignore(tc.path)
		if tc.expected == ignored {
			continue
		}

		t.Fatalf("[case %d] unexpected ignore value, expecting %v, got %v", i+1, tc.expected, ignored)
	}
}
