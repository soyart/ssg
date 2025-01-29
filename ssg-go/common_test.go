package ssg

import "testing"

func TestPerDir(t *testing.T) {
	type testCase struct {
		path     string
		expected int
	}

	defaultValue := 0
	pd := newPerDir(defaultValue)
	pd.add("/1", 1)
	pd.add("/1/2", 2)
	pd.add("/10/20", 20)
	pd.add("/10/20/30", 30)

	tests := []testCase{
		{
			path:     "/",
			expected: 0,
		},
		{
			path:     "/1",
			expected: 1,
		},
		{
			path:     "/1/3",
			expected: 1,
		},
		{
			path:     "/1/2",
			expected: 2,
		},
		{
			path:     "/1/2.d",
			expected: 1,
		},
		{
			path:     "/1/2/foo",
			expected: 2,
		},
		{
			path:     "/10/20/foo",
			expected: 20,
		},
		{
			path:     "/10.1/20/foo",
			expected: 0,
		},
		{
			path:     "/10.1/20/30/foo/bar/baz/1",
			expected: 0,
		},
		{
			path:     "/10/20/30/foo/bar/baz/1",
			expected: 30,
		},
	}

	for i := range tests {
		tc := &tests[i]
		actual := choose(tc.path, defaultValue, pd.values)
		if tc.expected != actual {
			t.Fatalf("unexpected value %v, expecting %v for path '%s'", actual, tc.expected, tc.path)
		}
	}
}
