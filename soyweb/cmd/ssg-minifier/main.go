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

	soyweb.FlagsV2
}

func main() {
	c := cli{}
	arg.MustParse(&c)
	if err := run(&c); err != nil {
		panic(err)
	}
}

func run(c *cli) error {
	var hookGenerate ssg.HookGenerate
	if c.MinifyHtmlGenerate {
		hookGenerate = soyweb.MinifyHtml
	}
	return ssg.Generate(
		c.Src, c.Dst, c.Title, c.Url,
		ssg.WritersFromEnv(),
		ssg.WithHooks(c.FlagsV2.Hooks()...),
		ssg.WithHooksGenerate(hookGenerate),
	)
}
