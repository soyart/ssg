package soyweb

import (
	"bytes"
	"testing"
)

func TestReplace(t *testing.T) {
	type testCase struct {
		s             string
		placeholderFn func(string) string
		target        string
		replace       ReplaceTarget

		expected string
	}

	tests := []testCase{
		{
			s:      "12${{ foo }}34",
			target: "${{ foo }}",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12new-foo34",
		},
		{
			s:      "${{ foo }} \n ${{ foo }}",
			target: "${{ foo }}",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 2,
			},
			expected: "new-foo \n new-foo",
		},
		{
			s:      "${{ foo }} \n ${{ foo }} \n {{ foo }}",
			target: "${{ foo }}",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 2,
			},
			expected: "new-foo \n new-foo \n {{ foo }}",
		},
		{
			s:      "12 ${{ foo }} 34",
			target: "${{ foo }}",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12 new-foo 34",
		},
		{
			s:             "12${{ foo }}34",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12new-foo34",
		},
		{
			s:      "12 ${{ foo}} 34",
			target: "${{ foo }}",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12 ${{ foo}} 34",
		},
		{
			s:             "12 ${{ foo}} 34",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12 ${{ foo}} 34",
		},
		{
			s:             "12 ${{ foo }} ${{ foo}} 34",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12 new-foo ${{ foo}} 34",
		},
		{
			s:             "12 ${{ foo }} ${{ foo }} 34",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 1,
			},
			expected: "12 new-foo ${{ foo }} 34",
		},
		{
			s:             "12 ${{ foo }} ${{ foo }} 34",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 2,
			},
			expected: "12 new-foo new-foo 34",
		},
		{
			s:             "12 ${{ foo }} ${{ foo }} 34",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 3,
			},
			expected: "12 new-foo new-foo 34",
		},
		{
			s:             "12 ${{ foo }} ${{ foo }} 34 ${{ foo }}",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 3,
			},
			expected: "12 new-foo new-foo 34 new-foo",
		},
		{
			s:             "12 ${{ foo }} ${{ foo }} 34 ${{ foo }}",
			placeholderFn: placeholder,
			target:        "foo",
			replace: ReplaceTarget{
				Text:  "new-foo",
				Count: 0,
			},
			expected: "12 new-foo new-foo 34 new-foo",
		},
	}

	for i := range tests {
		tc := &tests[i]
		target := tc.target
		if tc.placeholderFn != nil {
			target = tc.placeholderFn(tc.target)
		}

		actual, err := replace([]byte(tc.s), []byte(target), tc.replace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(actual, []byte(tc.expected)) {
			t.Fatalf("unexpected value '%s', expecting='%s'", actual, tc.expected)
		}
	}
}
