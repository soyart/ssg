package main

import (
	"os"
	"syscall"

	"github.com/soyart/ssg"
)

func main() {
	if len(os.Args) < 5 {
		ssg.Fprint(os.Stdout, "usage: ssg src dst title base_url\n")
		syscall.Exit(1)
	}

	src, dst, title, url := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	err := ssg.Generate(
		src, dst, title, url,
		ssg.ConcurrentFromEnv(),
		ssg.Streaming(),
	)
	if err != nil {
		ssg.Fprintln(os.Stdout, "error with", "src", src, "dst", dst, "title", title, "url", url)
		panic(err)
	}
}
