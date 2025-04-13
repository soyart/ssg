package main

import (
	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg/soyweb"
	"github.com/soyart/ssg/ssg-go"
)

type cli struct {
	Src   string `arg:"required,positional"`
	Dst   string `arg:"required,positional"`
	Title string `arg:"required,positional"`
	Url   string `arg:"required,positional"`

	soyweb.FlagsNoMinify
}

func main() {
	c := cli{}
	arg.MustParse(&c)
	if err := run(&c); err != nil {
		panic(err)
	}
}

func run(c *cli) error {
	flags := c.Flags()

	opts := []ssg.Option{
		ssg.WritersFromEnv(),
		ssg.WithHooks(flags.Hooks()...),
	}
	if flags.MinifyHtmlGenerate {
		opts = append(opts, ssg.WithHooksGenerate(soyweb.MinifyHtml))
	}

	return ssg.Generate(c.Src, c.Dst, c.Title, c.Url, opts...)

}
