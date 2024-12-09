# soyweb

soyweb is a collection of tools meant to replace [webtools](https://github.com/soyart/webtools).

At the core, it uses its own [Go implementation](https://github.com/soyart/ssg)
of [ssg](https://romanzolotarev.com/ssg.html) to convert Markdown files
into HTML files.

soyweb provides 3 more programs and a library for working with ssg:

> Most of these programs share the same CLI flags, and the help messagee
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

# ssg options provided by soyweb

## Minifiers

soyweb provides webformat minifiers opitions for ssg, implemented as hooks that
map 1 input data to 1 output data.

The minifiers is available to all programs under soyweb.

## [Blog article index generator (PoC)](./blog.go)

soyweb provides an [ssg.Impl](/options.go) that will automatically generate indices
for blog directories. It scans for `_index.ssg`, and, if found, lists all links to the
children (i.e. "articles"). The marker `_index.ssg` can be blank or contain template,
in plaintext, Markdown, or HTML.

To be considered an article, a path has to be either:

- A directory with `index.html` or `index.md`

- A file with `.md` extension

If the marker `_index.ssg` is empty, a default content header will be written.
If the marker has some template, then the index list will be appended to the template
in the output.

The generator is currently available to `ssg-manifest` via the site manifest specification
