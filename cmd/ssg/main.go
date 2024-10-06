package main

import (
	"fmt"

	"github.com/soyart/ssg"
)

const (
	header = `
<!DOCTYPE html>
<html lang="en">

<head>
  <title>Artnoi.com</title>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="keywords" content="artnoi, Prem Phansuriyanon">
  <meta name="author" content="@artnoi">
  <meta charset="UTF-8">
  <link href="/style.css" rel="stylesheet">
</head>

<body>
  <ul class="navbar">
    <li><a href="/"><img src="/toplogo.png" alt="Artnoi.com" class="logo">artnoi</a></li>
    <li class="f-right"><a href="/cheat/">cheat</a></li>
    <li class="f-right"><a href="/blog/">blog</a></li>
    <li class="f-right"><a href="/about/">about</a></li>
  </ul>`

	footer = `<hr>
<p><a href="#top">Back to top</a></p>
<hr>
<footer>
	<p>Copyright (c) 2019 - 2023 Prem Phansuriyanon</p>
	<p>Verbatim copying and redistribution of this entire page are permitted provided this notice is preserved</p>
</footer>
</body>

</html>`

	body = `
	# Henlo, vvorld!

Hi, I'm Prem Phansuriyanon, and I'm running this website.

I'm a self-taught back-end software engineer based in Bangkok.

Here's my quick intro:

- a back-end dev by trade

- really passionate about re-inventing the wheel

- use what I write

- love to share my shitty code projects to the world

See my embarassing code on [my GitHub](https://github.com/soyart).

Or you can read my unpopular opinions on [my Twitter](https://twitter.com/artnoi).

## Credits

The website partially uses monospace [TTF Hack](https://sourcefoundry.org/hack/),
which is my desktop and terminal font face, with blue-ish color scheme from
[Iceberg](https://github.com/cocopon/iceberg.vim).

As of Nov 2023, artnoi.com is served using [GitHub Pages](https://docs.github.com/en/pages),
and built from Markdown using [webtools](https://github.com/soyart/webtools).
`
)

func main() {
	h := ssg.ToHtml([]byte(header + body + footer))
	fmt.Printf("%s\n", h)
}
