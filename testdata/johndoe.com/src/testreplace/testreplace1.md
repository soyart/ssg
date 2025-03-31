# TestReplace1

## Replace all occurences with count=0

The manifest was configured with replacement `replace-me-0`.
This replacement was defined without a specific count,
so soyweb should remove all occurrences of placeholders containing
`replace-me-0`.

The line below should be changed to `replaced-text-0`:

${{ replace-me-0 }}

The line below should also be changed to `replaced-text-0`:

${{ replace-me-0 }}

Here within the code block too:

```go
// ${{ replace-me-0 }}
```

# And also here in the h1, in the `strong` tag: **${{ replace-me-0 }}**

## Replace with specificied count

This replacement `replace-me-1` was configured with count=3.

This means that, the first 3 replacements should succeed,
while the rest of occurrences remain unchanged:

> The 1st change: ${{ replace-me-1 }}
> The 2nd change: ${{ replace-me-1 }}
> The 3rd change: ${{ replace-me-1 }}
> The 4th change: ${{ replace-me-1 }}