# soyweb

soyweb is a collection of tools meant to replace [webtools](https://github.com/soyart/webtools).

At the core, it uses its own [Go implementation](https://github.com/soyart/ssg)
of [ssg](https://romanzolotarev.com/ssg.html) to convert Markdown files
into HTML files.

In addition to the Go clone of ssg, soyweb provides 3 more programs:

- [minifier](./cmd/minifier)

  A web minifier. It takes in a file, and minifies the file if
  the format is supported.

- [ssg-minifier](./cmd/ssg-minifier)

  A minifier-enabled version of standard ssg.
  It minifies all HTML and CSS files

- [ssg-manifest](./cmd/ssg-manifest)

  ssg-manifest wraps Go ssg implementation for better multi-site management

  > My original use case for ssg was with a [shell wrapper](https://github.com/soyart/webtools)
  > that facilitates building multiple sites with ssg.
  >
  > The wrapper tool used to have a declarative JSON manifest that specifies
  > source, destination, files to link or copy, and flag for cleaning up garbage.
  >
  > Those capabilities are now implemented by [`Manifest`](./manifest.go),
  > and accessible on the command-line via [`ssg-manifest`](./cmd/ssg-manifest/).
  >
  > See [`manifest.json`](./testdata/manifest.json) as example and run ssg-manifest,
  > to see its effects.

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
    ssg-manifest build [--no-cleanup] [--no-copy] [--no-build] [--min-html] [--min-html-all] [--min-css] [--min-json] [MANIFESTS [MANIFESTS ...]]
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
    ssg-manifest clean [MANIFESTS [MANIFESTS ...]]
    ```

  - `ssg-manifest copy`

    Copy files specified in the manifests' `copies` directive

    Synopsis:

    ```shell
    ssg-manifest copy [MANIFESTS [MANIFESTS ...]]
    ```
