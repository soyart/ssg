package ssg

import (
	"bytes"
	"errors"
	"fmt"
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
		testGenerate(t, func(s *Ssg) ([]string, []OutputFile, error) {
			return s.Build(nil)
		})
	})
}

func testGenerate(t *testing.T, buildFn func(s *Ssg) ([]string, []OutputFile, error)) {
	root := "../testdata/johndoe.com"
	src := filepath.Join(root, "/src")
	dst := filepath.Join(root, "/dst")
	title := "JohnDoe.com"
	url := "https://johndoe.com"

	err := os.RemoveAll(dst)
	if err != nil {
		panic(err)
	}

	s := New(src, dst, title, url)
	_, outputs, err := buildFn(&s)
	if err != nil {
		t.Errorf("unexpected error from scan: %v", err)
	}
	if !s.preferred.Contains(filepath.Join(src, "/blog/index.html")) {
		t.Fatalf("missing preferred html file /blog/index.html")
	}

	for i := range s.result.cache {
		o := &s.result.cache[i]

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
		header, ok := s.headers.values[dirname]
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

		ignorer := &gitIgnorer{GitIgnore: ignores}
		ignored := ignorer.Ignore(tc.path)
		if tc.expected == ignored {
			continue
		}

		t.Fatalf("[case %d] unexpected ignore value, expecting %v, got %v", i+1, tc.expected, ignored)
	}
}

// TestBuildAndWriteOut tests that Build+WriteOut both
// work as expected (identical to Generate)
func TestBuildAndWriteOut(t *testing.T) {
	root := "../testdata/johndoe.com"
	src := filepath.Join(root, "/src")
	dstGenerate := filepath.Join(root, "/dstGenerate")
	dstBuild := filepath.Join(root, "/dstBuild")
	title := "JohnDoe.com"
	url := "https://johndoe.com"

	err := os.RemoveAll(dstGenerate)
	if err != nil {
		panic(err)
	}
	err = os.RemoveAll(dstBuild)
	if err != nil {
		panic(err)
	}
	// Build with nil outputs (no concurrent writer)
	files, cache, err := Build(src, dstBuild, title, url, nil)
	if err != nil {
		panic(err)
	}
	err = WriteOutSlice(cache, 1)
	if err != nil {
		panic(err)
	}
	stat, err := os.Stat(src)
	if err != nil {
		panic(err)
	}
	err = GenerateMetadata(src, dstBuild, url, files, cache, stat.ModTime())
	if err != nil {
		panic(err)
	}
	// Generate output
	err = Generate(src, dstGenerate, title, url)
	if err != nil {
		panic(err)
	}
	// Assert equal
	testDeepEqual(t, dstGenerate, dstBuild)
}

func TestErrors(t *testing.T) {
	retBreakPipe := func() error {
		return ErrBreakPipelines
	}
	wrapBreakPipe := func(err error) error {
		return fmt.Errorf("foo bar %w: %w", err, ErrBreakPipelines)
	}
	retSkipCore := func() error {
		return ErrSkipCore
	}
	wrapSkipCore := func(err error) error {
		return fmt.Errorf("foo bar %w: %w", err, ErrSkipCore)
	}

	tests := map[error][]error{
		ErrBreakPipelines: {
			retBreakPipe(),
			wrapBreakPipe(errors.New("some error")),
			fmt.Errorf("%w-%w", ErrBreakPipelines, ErrSkipCore),
			fmt.Errorf("%w%w", ErrBreakPipelines, ErrSkipCore),
			fmt.Errorf("%w%w", ErrSkipCore, ErrBreakPipelines),
		},
		ErrSkipCore: {
			retSkipCore(),
			wrapSkipCore(errors.New("some other error")),
		},
	}

	for target, errs := range tests {
		for i := range errs {
			if errors.Is(errs[i], target) {
				continue
			}
			t.Fatalf("unexpected unwrapping error %v for case %d", target, i+1)
		}
	}
}
