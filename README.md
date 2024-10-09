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

### TODO for ssg-go

- `.ssgignore`

- `.files` (maybe 'won't do')

### Differences between ssg and ssg-go

- ssg-go does not extract title from H1 tag,
  and will do nothing on the argument given

  > But ssg-go still takes `title` as its 3rd argument,
  > so as to be the original ssg's drop-in replacement.

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
