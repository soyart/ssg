package main

import (
	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
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
	minifyOpts := soyweb.SsgOptions(soyweb.Flags{
		MinifyFlags: soyweb.MinifyFlags{
			MinifyHtmlGenerate: true,
			MinifyHtmlCopy:     true,
			MinifyCss:          true,
			MinifyJson:         true,
		},
		NoMinifyFlags: c.NoMinifyFlags,
	})

	opts := append(
		[]ssg.Option{ssg.ParallelWritesEnv()},
		minifyOpts...,
	)

	return ssg.GenerateWithOptions(c.Src, c.Dst, c.Title, c.Url, opts...)
}
