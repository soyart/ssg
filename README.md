# ssg static site generator

This Nix Flake provides 2 implementations of ssg.

## POSIX shell ssg (original implementation)

> See also: [romanzolotarev.com](https://romanzolotarev.com/ssg.html)

The original script is copied from [rgz.ee](https://romanzolotarev.com/bin/ssg).

Through [`flake.nix`](./flake.nix), ssg's runtime dependencies will be included
in the derivation.

## Go implementation of ssg (ssg-go)

My own implementation, using [github.com/gomarkdown/markdown](https://github.com/gomarkdown/markdown).

This implementation is good for deploying ssg remotely,
because it's just 1 Go executable.

## Usage

```sh
ssg <src> <dst> <title> <url>
```

ssg reads Markdown files from `src`, prepends it with `_header.html`,
and appends it with content of `_footer.html`. The output files are mirrored
into `dst`. Files or directories whose names start with `.` are skipped.

Files listed in `${src}/.ssgignore` are also ignored in a fashion similar
to `.gitignore`. To see how `.ssgignore` works in Go implementation, see
[the test `TestSsgignore`](./ssg_test.go).

If we have `foo.html` and `foo.md`, the HTML file wins.

Files with extensions other than `.md` will simply be copied
into mirrored `dst`.

ssg also generates `dst/sitemap.xml` with data from the CLI parameter.

### Differences between ssg and ssg-go

- Like the original, ssg-go accepts a CLI parameter `title` (3rd arg)
  that will be used as default `<title>` tag inside `<head>` (*head title*).

  ssg-go also parses `_header.go` for title replacement placeholder. Currently,
  ssg-go recognizes 2 placeholders:

  - `{{from-h1}}`

    This will prompt ssg-go to use the first `<h1>` tag value as head title.

  - `{{from-tag}}`

    This will prompt ssg-go to find the first line starting with `:title`,
    and use it as the document head title.

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

  :title Real Header

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

- ssg-go cascades `_header.html` and `_footer.html` down the directory tree

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

## Manifests

My original use case for ssg was with a [shell wrapper](https://github.com/soyart/webtools)
that facilitates building multiple sites with ssg.

The wrapper tool used to have a declarative JSON manifest that specifies
source, destination, files to link or copy, and flag for cleaning up garbage.

Those capabilities are now implemented by [`Manifest`](./manifest.go),
and accessible on the command-line via [`ssg-manifest`](./cmd/ssg-manifest/).

See `manifest.json` as example, or clone this repo and run [`./cmd/ssg-manifest/`]
to see its effects.

### ssg-manifest

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

  Synopsis:

  ```shell
  ssg-manifest build [--no-cleanup] [--no-copy] [--no-build] [...manifests]
  ```

  Examples:

  ```shell
  # Build from ./manifest.json (same with default behavior)
  ssg-manifest build

  # Build from ./m1.json and ./m2.sjon
  ssg-manifest build ./m1.json ./m2.json

  # Build from ./manifest.json without copying
  ssg-manifest build --no-copy

  # Build from ./m1 and ./m2.json
  # without actually building HTMLs from Markdowns
  ssg-manifest build --no-build ./m1.json ./m2.json
  ```

- `ssg-manifest clean`

  Removes target files specified in the manifests' `copies` directive

  Synopsis:

  ```shell
  ssg-manifest clean [...manifests]
  ```

- `ssg-manifest copy`

  Copy files specified in the manifests' `copies` directive

  Synopsis:

  ```shell
  ssg-manifest copy [...manifests]
  ```
