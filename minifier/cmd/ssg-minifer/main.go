package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/minifier"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprint(os.Stdout, "usage: ssg src dst title base_url\n")
		syscall.Exit(1)
	}

	src, dst, title, url := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	s := ssg.NewWithOptions(src, dst, title, url,
		ssg.ParallelWritesEnv(),
		ssg.Pipeline(minifier.Minify),
		ssg.Hook(minifier.MinifyHtml),
	)

	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
