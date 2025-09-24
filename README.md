# This repository has been archived

Development of ssg-go moved to [soyart/ssg-go](https://github.com/soyart/ssg-go),
while soyweb moved to [soyart/soyweb](https://github.com/soyart/soyweb).

---

# ssg (static site generator)

ssg is a Markdown static site generator.

ssg generates a website from directory tree,
with Markdown files being converted and assembled into HTMLs.

## ssg implementations

This Nix Flake provides 2 implementations of ssg.

- [Original ssg](https://romanzolotarev.com/ssg.html)

  The original POSIX shell script is [copied from rgz.ee](https://romanzolotarev.com/bin/ssg).

  Through [`flake.nix`](./flake.nix), ssg's runtime dependencies will be included
  in the derivation.

  ```shell
  # Original shell implementation
  nix build          # Build default package - pure POSIX shell ssg
  nix build .#impure # Build directly from ssg.sh
  ```

- [ssg-go](./ssg-go/)

  Go implementation of ssg, using [github.com/gomarkdown/markdown](https://github.com/gomarkdown/markdown).

  This implementation is good for deploying ssg remotely,
  because it's just 1 Go executable with 0 runtime dependencies.

  In addition to the executables, ssg-go also provides a library
  for extending ssg as Go module `github.com/soyart/ssg/ssg-go`.

  [soyweb](./soyweb/) is another ssg-go wrapper that uses the exposed ssg-go API
  to extend ssg-go with non-core features such as minifiers and index file generator.
  Like with ssg-go, soyweb provides its library as a Go module `github.com/soyart/ssg/soyweb`.

  > Note: both ssg-go and soyweb will probably not work on Windows due to
  > Windows path delimiter being different than POSIX's

  ```shell
  # Go implementation
  nix build .#ssg-go # Build executables from Go implementation of ssg
  nix build .#soyweb # Build soyweb programs
  ```

## Usage for both implementations

```sh
ssg <src> <dst> <title> <url>
```

- Files or directories whose names start with `.` are ignored.

  Files listed in `${src}/.ssgignore` are also ignored in a fashion similar
  to `.gitignore`. To see how `.ssgignore` works in ssg-go, see
  [the test `TestSsgignore`](./ssg-go/ssg_test.go).

- Files with extensions other than `.md` and `html` will simply be copied
  into mirrored `${dst}`.

  If we have `foo.html` and `foo.md`, the HTML file wins.

- ssg reads Markdown files under `${src}`, converts each to HTML,
  and prepends and appends the resulting HTML with `_header.html`
  and `_footer.html` respectively.

  The assembled output file is then mirrored into `${dst}`
  with `.html` extension.

- In the end, ssg generates metadata such as `${dst}/sitemap.xml` with data
  from the CLI parameter and the output tree, and `${dst}/.files` to remember
  what files it had processed.

- HTML tags `<head><title>` is extracted from the first Markdown h1 (default),
  or a default value provided at the command-line (the 3rd argument).

  > With ssg-go, the titles can also be extracted from special line starting
  > with `:ssg-title` tag. This line will be removed from the output.

## Differences between ssg and ssg-go

### ssg-go ignores `.files`

In the original ssg, filenames listed in `.files` are
ignored and not re-generated. Unlike the original ssg, ssg-go ignores `${dst}/.files`
simply because it adds needless complexity.

By ignoring `.files`, we can be sure that the output directory is generated in a
functional fashion, i.e. we'll always get the same output with the same source material.

To do caching from previous run, an option to store file hashes in `${dst}/.files.sha256`
seems attractive. But upon closer inspection, it seems problems will arise when people
use ssg-go with other wrappers that read other files or do substitutions.

### ssg-go concurrent writers

ssg-go has built-in concurrent output writers.

Environment variable `SSG_WRITERS` sets the number of concurrent writers in ssg-go,
i.e. at any point in time, at most `SSG_WRITERS` number of threads are writing output
files.

The default value for concurrent writer is 20. If the supplied value is illegal,
ssg-go falls back to 20 concurrent writers.

> To write outputs sequentially, set the write concurrency value to 1:
>
> ```shell
> SSG_WRITERS=1 ssg mySrc myDst myTitle myUrl
> ```

### ssg-go custom title tag for `_header.html`

ssg-go also parses `_header.go` for title replacement placeholder.
Currently, ssg-go recognizes 2 placeholders:

- `{{from-h1}}`

  This will prompt ssg-go to use the first Markdown line starting with `#` value as head title.
  For example, if this is your Markdown:

  ```markdown
  ## This is H2

  # This is H1

  :ssg-title This is also an H1

  This is paragraph
  ```

  then `This is H1` will be used as the page's title.

- `{{from-tag}}`

  Like with `{{from-h1}}`, but finds the first line starting with `:ssg-title` instead,
  i.e. `This is also an H1` from the example above will be used as the page's title.

  > Note: `{{from-tag}}` directive will make ssg look for pattern `:ssg-title YourTitle\n\n`,
  > so users must always append an empty line after the title tag line.

For example, consider the following header/footer templates and a Markdown page:

```html
<!-- _header.html -->

<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-tag}}</title>
</head>
<body>
```

```html
<!-- _footer.html -->

</body>
</html>
 ```

```markdown
<!-- some/path/foo.md -->

Mar 24 2024

:ssg-title Real Header

# Some Header 2

Some para
```

This is the generated HTML equivalent, in `${dst}/some/path/foo.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Real Header</title>
</head>
<body>
<p>Mar 24 2024</p>
<h1>Some Header 2</p>
<p>Some para</p>
</body>
</html>
```

Note how `{{from-tag}}` in `_header.html` will cause ssg-go to use `Real Header`
as the document head title.

On the other hand, the `{{from-h1}}` will cause ssg-go to use `Some Header 2`
as the document head title.

### Cascading header and footer templates

ssg-go cascades `_header.html` and `_footer.html` down the directory tree

If your tree looks like this:

```
├── _header.html
├── blog
│   ├── 2023
│   │   ├── _header.html
│   │   ├── bar.md
│   │   ├── baz
│   │   │   └── index.md
│   │   └── foo.md
│   ├── _header.html
│   └── index.md
└── index.md  
```

Then:

- `/index.md` will use `/_header.html`

- `/blog/index.md` will use `/blog/_header.html`

- `/blog/2023/baz/index.md` will use `/blog/2023/_header.html`

