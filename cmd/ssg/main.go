package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprint(os.Stdout, "usage: ssg src dst title base_url\n")
		syscall.Exit(1)
	}

	src, dst, title, url := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	s := ssg.NewWithOptions(
		src,
		dst,
		title,
		url,
		ssg.ParallelWritesEnv(),
	)

	if err := s.Generate(); err != nil {
		fmt.Fprintln(os.Stdout, "error with", "src", src, "dst", dst, "title", title, "url", url)
		panic(err)
	}
}
