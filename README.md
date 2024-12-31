# ssg (static site generator)

> This repository also hosts [soyweb](./soyweb/),
> an ssg wrapper and replacement for [webtools](https://github.com/soyart/webtools)

This Nix Flake provides 2 implementations of ssg.

- POSIX shell ssg

  > See also: [romanzolotarev.com](https://romanzolotarev.com/ssg.html)

  The original script is copied from [rgz.ee](https://romanzolotarev.com/bin/ssg).

  Through [`flake.nix`](./flake.nix), ssg's runtime dependencies will be included
  in the derivation.

- Go implementation of ssg (ssg-go)

  My own implementation, using [github.com/gomarkdown/markdown](https://github.com/gomarkdown/markdown).

  This implementation is good for deploying ssg remotely,
  because it's just 1 Go executable.

  In addition to the executables, ssg-go also provides
  extensible ssg implementations with Go API.

  A Go wrapper for ssg-go is also available in [`soyweb`](./soyweb/).

  > Note: both ssg-go and soyweb will probably not work on Windows due to
  > Windows path delimiter being different than POSIX's

## Build from Nix flake

```sh
nix build          # Build default package - pure POSIX shell ssg
nix build .#impure # Build directly from ssg.sh
nix build .#ssg-go # Build Go implementation of ssg
nix build .#soyweb # Build soyweb programs
```

## Usage for both implementations

```sh
ssg <src> <dst> <title> <url>
```

ssg reads Markdown files under `${src}`, converts each to HTML,
and prepends and appends the resulting HTML with `_header.html`
and `_footer.html` respectively. The output file tree is mirrored into `${dst}`.

Files or directories whose names start with `.` are ignored.
Files listed in `${src}/.ssgignore` are also ignored in a fashion similar
to `.gitignore`. To see how `.ssgignore` works in Go implementation, see
[the test `TestSsgignore`](./ssg_test.go).

If we have `foo.html` and `foo.md`, the HTML file wins.

Files with extensions other than `.md` will simply be copied
into mirrored `${dst}`.

ssg also generates `${dst}/sitemap.xml` with data from the CLI parameter.

HTML tags `<head><title>` is extracted from the first Markdown h1 (default),
or a default value provided at the command-line (the 3rd argument).

> With ssg-go, the titles can also be extracted from special line starting
> with `:ssg-title` tag. This line will be removed from the output.

## Differences between ssg and ssg-go

### Custom title tag for `_header.html`

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

## ssg-go quirks

### Concurrent writers

ssg-go has built-in concurrent output writers.

The number of writers can be set at runtime by environment variable `SSG_WRITERS`.
At any point in time, at most `SSG_WRITERS` number of threads are writing output files.

The default value for concurrent writer is 20. If the supplied value is illegal,
ssg-go falls back to 20 concurrent writers.

> To write outputs sequentially, set the write concurrency value to 1:
> 
> ```shell
> SSG_WRITERS=1 ssg mySrc myDst myTitle myUrl
> ```

### Streaming and caching builds

To minimize runtime memory usage, ssg-go builds and writes concurrently.
There're 2 main ssg threads: one is for building the outputs,
and the other is the write thread.

The build thread *sequentially* reads, builds and sends outputs
to the write thread via a buffered Go channel.

Bufffering allows the builder thread to continue to build and send outputs
to the writer until the buffer is full.

This helps reduce back pressure, and keeps memory usage low.
The buffer size is, by default, 2x of the number of writers.

This means that, at any point in time during a generation of any number of files
with 20 writers, ssg-go will at most only hold 40 output files
in memory (in the buffered channel).

If you are importing ssg-go to your code and you don't want this
streaming behavior, you can use the exposed function `Build`, `WriteOut`,
and `GenerateMetadata`:

```go
dist, err := ssg.Build(src, dst, title, url)
if err != nil {
  panic(err)
}

err = ssg.WriteOut(dist)
if err != nil {
  panic(err)
}

err = GenerateMetadata(url, dst, dist, time.Time{})
if err != nil {
  panic(err)
}
```

# Extending ssg-go

Go programmers can extend ssg-go via its [`Option` type](./options.go).

[soyweb](./soyweb/) also extends ssg via `Option`,
and provides extra functionality such as index generator and minifiers.

## `Hook` option

`Hook` is a Go function called on every unignored input file.
soyweb uses this option to implement global minifiers.

It is enabled with `WithHook(hook)`.

## `HookGenerate` option

`HookGenerate` is a Go function called on every generated HTML.
soyweb uses this option to implement output minifier.

It is enabled with `WithHookGenerate(hook)`

## `Pipeline` option

`Pipeline` is a Go function called on a file during directory walk.
To reduce complexity, ignored files and ssg headers/footers are not sent
to `Pipeline`. This preserves the core functionality of the original ssg.

Pipelines can be chained together with `WithPipelines(p1, p2, p3)`

