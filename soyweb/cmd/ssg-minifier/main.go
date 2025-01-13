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

	soyweb.NoMinifyFlags
}

func main() {
	c := cli{}
	arg.MustParse(&c)

	err := run(&c)
	if err != nil {
		panic(err)
	}
}

func run(c *cli) error {
	opts := []ssg.Option{ssg.WritersFromEnv()}
	opts = append(
		opts,
		soyweb.SsgOptions(soyweb.Flags{
			MinifyFlags: soyweb.MinifyFlags{
				MinifyHtmlGenerate: true,
				MinifyHtmlCopy:     true,
				MinifyCss:          true,
				MinifyJson:         true,
			},
			NoMinifyFlags: c.NoMinifyFlags,
		})...,
	)

	return ssg.Generate(c.Src, c.Dst, c.Title, c.Url, opts...)
}
