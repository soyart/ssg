package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprint(os.Stdout, "usage: ssg src dst title base_url\n")
		syscall.Exit(1)
	}

	src, dst, title, url := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	// Reset, to avoid "too many positional arguments"
	// from parsing NoMinifyFlags
	os.Args = os.Args[4:]
	f := soyweb.NoMinifyFlags{}
	arg.MustParse(&f)

	minifyOpts := soyweb.SsgOptions(soyweb.Flags{
		MinifyFlags: soyweb.MinifyFlags{
			MinifyHtml:    true,
			MinifyHtmlAll: true,
			MinifyCss:     true,
			MinifyJson:    true,
		},
		NoMinifyFlags: f,
	})

	opts := append(
		[]ssg.Option{ssg.ParallelWritesEnv()},
		minifyOpts...,
	)

	s := ssg.NewWithOptions(src, dst, title, url, opts...)
	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
