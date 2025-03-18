# Test replace

soyweb will replace string keys `k` enclosed in `${{ k }}`.

i.e. if you have specified to replace `somereplace` with `sometarget`,
then `${{ somereplace }}` will be replaced with `sometarget`

As per the manifest, the lines below before `===END===` should be replaced thrice,
leaving one placeholder `replace-me-1`:

${{ replace-me-0 }}

${{ replace-me-0 }}

${{ replace-me-0 }}

${{ replace-me-0 }}

${{ replace-me-1 }}

${{ replace-me-1 }}

${{ replace-me-1 }}

${{ replace-me-1 }}

===END===

It should also work inside some Markdown construct:

The replace text should appear in `code` tag: `${{ replace-me-0 }}`

The replace text should appear in italic tag: *${{ replace-me-0 }}*
