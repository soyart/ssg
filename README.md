# ssg

This Nix Flake provides 2 implementations of ssg.

## 1. The original POSIX ssg

> See also: [romanzolotarev.com](https://romanzolotarev.com/ssg.html)

The original script is copied from [rgz.ee](https://romanzolotarev.com/bin/ssg).

Through [`flake.nix`](./flake.nix), ssg's runtime dependencies will be included
in the derivation.

## 2. Go implementation of ssg

My own implementation, using [github.com/gomarkdown/markdown](https://github.com/gomarkdown/markdown).

This implementation is good for using ssg remotely, because it's just 1 executable.

TODO:

- Identical arguments and behavior

- `.ssgignore`

- `sitemap.xml`

- `.files` (maybe 'won't do')
