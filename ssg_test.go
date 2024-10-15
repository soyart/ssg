package ssg

import "testing"

func TestTitleFromH1(t *testing.T) {
	type testCase struct {
		d        string // Default header
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
	}

	for i := range tests {
		tc := &tests[i]
		actual := titleFromH1("", tc.head, []byte(tc.markdown))
		if actual != tc.expected {
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expected)
			t.Logf("Actual:\nSTART===\n%s\nEND===", actual)

			t.Fatalf("unexpected value for case %d", i+1)
		}
	}
}

func TestTitleFromTag(t *testing.T) {
	type testCase struct {
		d                string // Default header
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

:title My own title

# Some h1
					
Some para`,
			expectedHead: `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>My own title</title>
</head>`,
			expectedMarkdown: `
Mar 24 1998

# Some h1

Some para`,
		},
	}

	for i := range tests {
		tc := &tests[i]
		head, markdown := titleFromTag("", tc.head, []byte(tc.markdown))
		if head != tc.expectedHead {
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expectedHead)
			t.Logf("Actual:\nSTART===\n%s\nEND===", head)
			t.Logf("len(expected) = %d, len(actual) = %d", len(tc.expectedHead), len(head))

			t.Fatalf("unexpected value for case %d", i+1)
		}
		if md := string(markdown); md != tc.expectedMarkdown {
			t.Logf("Expected:\nSTART===\n%s\nEND===", tc.expectedMarkdown)
			t.Logf("Actual:\nSTART===\n%s\nEND===", md)
			t.Logf("len(expected) = %d, len(actual) = %d", len(tc.expectedMarkdown), len(markdown))

			for i := range tc.expectedMarkdown {
				e := tc.expectedMarkdown[i]
				a := md[i]
				t.Logf("%d: diff=%v actual='%c', expected='%c'", i, e != a, e, a)
			}

			t.Fatalf("unexpected value for case %d", i+1)
		}
	}
}
