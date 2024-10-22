# ssg static site generator

This Nix Flake provides 2 implementations of ssg.

## 1. The original POSIX shell ssg

> See also: [romanzolotarev.com](https://romanzolotarev.com/ssg.html)

The original script is copied from [rgz.ee](https://romanzolotarev.com/bin/ssg).

Through [`flake.nix`](./flake.nix), ssg's runtime dependencies will be included
in the derivation.

## 2. Go implementation of ssg (ssg-go)

My own implementation, using [github.com/gomarkdown/markdown](https://github.com/gomarkdown/markdown).

This implementation is good for using ssg remotely, because it's just 1 executable.

## Usage

```sh
ssg <src> <dst> <title> <url>
```

ssg reads Markdown files from `src`, prepends it with `_header.html`,
and appends it with content of `_footer.html`. The output files are mirrored
into `dst`. Files or directories whose names start with `.` are skipped.

Files listed in `${src}/.ssgignore` are also ignored.

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

My original use case for ssg was with a shell wrapper that facilitates
building multiple sites.

The wrapper tool used to have a declarative manifest that specifies
source, destination, files to link or copy, and cleaning up garbage.

This is also implemented by [`Manifest`](./manifest.go), and accessible
via [`soyweb`](./cmd/soyweb/) binary.

See `manifest.json` as example, or clone this repo and run `./cmd/soyweb/`
to see its effects.
