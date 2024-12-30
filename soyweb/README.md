# soyweb

soyweb is a collection of [ssg-go](https://github.com/soyart/ssg)
wrappers and extensions.

soyweb extensions respect ssg-go quirks such as ssgignore,
handling of header and footer templates, preference of source HTML files
over the Markdown, file permissions, etc.

## soyweb programs

In addition to extension library, soyweb provides a few programs that
serve as a collection of tools meant to replace [webtools](https://github.com/soyart/webtools).
The main purpose is to build [artnoi.com](https://artnoi.com) with a single binary
to streamline the site's CI/CD pipelines.

> Most of these programs share the same CLI flags, and the help messages
> can be accessible via `-h` or `--help`

- [minifier](./cmd/minifier)

  A web format minifier. It minifies a single source file and writes the
  minified version to different location, or all supported files under
  the source directory

- [ssg-minifier](./cmd/ssg-minifier)

  A minifier-enabled version of standard ssg

- [ssg-manifest](./cmd/ssg-manifest)

  ssg-manifest wraps Go ssg implementation for better multi-site management,
  and is intended to be a better replacement for [webtools](https://github.com/soyart/webtools)

  > See [`manifest.json`](./testdata/manifest.json) as example
  >
  > To try ssg-manifest, go into `./testdata` and run ssg-manifest

  Synopsis:
  ```
  ssg-manifest <command> [<args>]
  ```

  ssg-manifest reads manifest(s) and apply changes specified in them.
  Because it is a multi-stage application, ssg-manifest supports 3 subcommands
  for better user experience:

  - Default mode

    It builds `./manifest.json` with all stages.

    Due to the limitation of the CLI library, this default
    mode takes no arguments.

    ```shell
    # Build from ./manifest.json (default path)
    ssg-manifest
    ```

  - `ssg-manifest build`

    This subcommands build sites from one or multiple manifests.

    We can specify skip flags to `build`, which will make ssg-manifest
    skip some particular stages during application of manifests.

    Help:

    ```shell
    ssg-manifest build -h
    ```

    Examples:

    ```shell
    # Build from ./manifest.json (same with default behavior)
    ssg-manifest build

    # Build from ./m1.json and ./m2.sjon
    ssg-manifest build ./m1.json ./m2.json

    # Build from ./manifest.json without copying
    ssg-manifest build --no-copy

    # Build from ./m1.json and ./m2.json
    # without actually building HTMLs from Markdowns
    ssg-manifest build --no-build ./m1.json ./m2.json

    # Build from ./m1.json and ./m2.json
    # and minify all HTML files built from Markdown
    ssg-manifest build ./m1.json ./m2.json --min-html

    # Like above, but minify all HTML files
    ssg-manifest build ./m1.json ./m2.json --min-html --min-html-copy

    # Like above, but minify all HTML files and CSS files
    ssg-manifest build ./m1.json ./m2.json --min-html --min-html-copy --min-css
    ```

  - `ssg-manifest clean`

    Removes target files specified in the manifests' `copies` directive

    Help:

    ```shell
    ssg-manifest clean -h
    ssg-manifest cleanup -h
    ```

  - `ssg-manifest copy`

    Copy files specified in the manifests' `copies` directive

    Help:

    ```shell
    ssg-manifest copy -h
    ```

## soyweb ssg-go options

soyweb extends ssg-go options using `ssg.Option` type.

### Minifiers

soyweb provides webformat minifiers opitions for ssg, implemented as
hooks that map 1 input data to 1 output data.

The minifiers is available to all programs under soyweb.

### [Index generator](./index.go)

soyweb provides an automatic index generator implemented as a [ssg.Pipeline](/options.go).
This pipeline will automatically generate index sibling Markdowns, HTMLs, and directories.

It looks for marker file `_index.soyweb` somewhere under `${src}`, and,
if found, lists all links to the children (i.e. "articles").

The marker `_index.soyweb` can be empty, or contain template. If not empty, the
template inside the marker will be treated as Markdown.

To be considered an entry by the generator, the marker's sibling has to satisfy
at least one of the criteria:

- A file with `.md` or `.html` extension

  The generated index will point to HTML extensions

- A directory with `index.html` or `index.md`

  The generated index will point to HTML extensions

- A directory with another marker `_index.soyweb` (recursive)

  The generated index will point to `${sibling}/index.html`

The generator is currently available to `ssg-manifest` via the site manifest specification.

#### Index generator: templates in markers

The generator allows markers to contain partial template.

The marker `_index.soyweb` could be a Markdown, and apart from having its content
appended by the generated index, the file is handled normally like with other
ssg-go input files.

If the marker `_index.soyweb` is empty, a default content header will be written.
If the marker has some template, then the index list will be appended to the template
in the output.

#### Index generator: how it's generated

The generated indexes are treated just like any other source Markdown files.

In other words, `_header.html` and `_footer.html` will surround the index generated from
marker files. [`ssg.TitleFrom`](../title.go) tags are respected and title extraction
for the generated index is handled in the familiar fashion.

#### Index generator: link title extraction (files)

Only Markdown and HTML siblings are to be linked. For example, if we have 2 files
`1.md` and `2.html`, then links will be generated for `1.html` and `2.html`.

Link titles are extracted from tag `:ssg-title` first (`:ssg-title FooTitle`),
and if there's no such title, then the generator falls back to Markdown h1 titles
(`# FooTitle`) will be picked as the child title *within* the index.

#### Index generator: link title extraction (directories)

If a marker's sibling is a directory with `index.md`, then the titles will be extracted
from the Markdown index like with files.

If there's no `index.md`, then the directory names will be used as titles. And instead
of linking to `/some/path/targetdir/index.md`, the directory links ends with a slash,
so `/some/path/targetdir/` is the hyperlink generated.

#### Index generator: practical examples

Consider a ssg source directory `src`:

```
src
│
├── _footer.html
├── _header.html
├── _index.soyweb
│
├── foo.md
├── bar.md
│
├── 2022
│   ├── _index.soyweb
│   ├── lol.md
│   └── not_article.svg
│
├── 2023
│   ├── hahaha
│   │   └── index.md
│   ├── _index.soyweb
│   └── baz.md
│
└── somedir
    └── index.md
```

We see 3 markers, in `src`, `src/2022`, and `src/2023`. This means
that we should get 3 generated `index.html`s. But we also see that
`src/2022/index.html` already exists.

Because ssg-go gives preferences to HTML files with matching base names,
and because the generator respects this behavior, the source HTML will be
copied and new index not generated.

If we focus on the root marker. We see that it has 5 siblings, namely,
`foo.md`, `bar.md`, `2022`, `2023`, and `somedir`.

The generated index should have links to these 3 children.
If the destination is `dst`, then the generated index from this root marker
will be at `dst/index.html`, mirroring `src/_index.soyweb` location in `src`.

We can look up the special ssg files so that we can compare the output:

`src/_header.html`:

```html
<!-- My blog header! -->
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{from-tag}}</title>
</head>
<body>
```

`src/_footer.html`:

```html
<!-- My blog footer! -->
</body>
</html>
```

`src/_index.soyweb`:

```markdown
:ssg-title My blog!

# Welcome to my blog!

Below is the list of my articles!

```

`src/foo.md`:

```
# Foo article

Foo is better than fu
```

`src/bar.md`:

```
:ssg-title Bar article

# Barbarbarbar

Greeks called other peoples babarians because all they hear is barbarbar
```

`src/somedir/index.md`:

```markdown
:ssg-title SomeDir title

# Welcome to SomeDir!

This is some directory
```

Now, the generated index `dst/index.html` looks like this:

```html
<!-- My blog header! -->
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>My blog!</title>
</head>
<body>
<h1>Welcome to my blog!</h1>
<p>Below is the list of my articles!</p>
<ul>
  <li><p><a href="/2023/">2023</a></p></li>
  <li><p><a href="/2022/">2022</a></p></li>
  <li><p><a href="/bar.html">Bar article</a></p></li>
  <li><p><a href="/foo.html">Foo article</a></p></li>
  <li><p><a href="/somedir/">SomeDir title</a></p></li>
</ul>
<!-- My blog footer! -->
</body>
</html>
```

What about `src/2023/_index.soyweb` marker? If it's not empty,
then the output will be generated in a fashion similar
to the root marker example above.

But what if the marker `src/2023/_index.soyweb` is empty?
If so, then the default heading will be used, and the generated
`dst/2023/index.html` looks something like this:

```html
<!-- My blog header! -->
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>SomeDefaultSsgGoTitle</title>
</head>
<body>
<h1>Index of somedir</h1>
<ul>
  <li><p><a href="/hahaha/">BarTitle</a></p></li>
  <li><p><a href="/baz.html">BazTitle</a></p></li>
</ul>
<!-- My blog footer! -->
</body>
</html>
```

