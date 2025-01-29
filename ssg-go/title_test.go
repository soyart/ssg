package ssg_test

import (
	"bytes"
	"testing"

	"github.com/soyart/ssg/ssg-go"
)

func TestTitleFromH1(t *testing.T) {
	type testCase struct {
		head          string
		markdown      string
		expectedTitle string
		expectedHead  string
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
			expectedTitle: "Some h1",
			expectedHead: `
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

:ssg-title Not a title

## Some h2

# Some h1

Some para`,
			expectedTitle: "Some h1",
			expectedHead: `
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
		title := ssg.GetTitleFromH1([]byte(tc.markdown))
		if string(title) != tc.expectedTitle {
			t.Logf("Expected='%s'", tc.expectedTitle)
			t.Logf("Actual='%s'", string(title))
			t.Fatalf("unexpected title for case %d", i+1)
		}

		actual := ssg.AddTitleFromH1([]byte{}, []byte(tc.head), []byte(tc.markdown))
		if !bytes.Equal(actual, []byte(tc.expectedHead)) {
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expectedHead)
			t.Logf("Actual:\nSTART===\n%s\nEND===", actual)

			t.Fatalf("unexpected value for case %d", i+1)
		}
	}
}

func TestTitleFromTag(t *testing.T) {
	type testCase struct {
		head             string
		markdown         string
		expectedTitle    string
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

:ssg-title My title

# Some h1

Some para`,
			expectedTitle: "My title",
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

	:ssg-title Not actually title

:ssg-title This is the title

# Some h1

Some para  `,
			expectedTitle: "This is the title",
			expectedHead: `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>This is the title</title>
</head>`,
			expectedMarkdown: `
Mar 24 1998

	:ssg-title Not actually title

# Some h1

Some para  `,
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

	:ssg-title Not actually title

:ssg-title This is the title

:ssg-title This should persist

# Some h1

Some para  `,
			expectedTitle: "This is the title",
			expectedHead: `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>This is the title</title>
</head>`,
			expectedMarkdown: `
Mar 24 1998

	:ssg-title Not actually title

:ssg-title This should persist

# Some h1

Some para  `,
		},
	}

	for i := range tests {
		tc := &tests[i]
		title := ssg.GetTitleFromTag([]byte(tc.markdown))
		if string(title) != tc.expectedTitle {
			t.Logf("Expected='%s'", tc.expectedTitle)
			t.Logf("Actual='%s'", string(title))
			t.Fatalf("unexpected title for case %d", i+1)
		}

		head, markdown := ssg.AddTitleFromTag([]byte{}, []byte(tc.head), []byte(tc.markdown))
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
