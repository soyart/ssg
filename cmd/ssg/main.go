package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("usage: ssg src dst title base_url")
		syscall.Exit(1)
	}

	src, dst, title, url := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	s := ssg.New(src, dst, title, url)
	if err := s.Generate(); err != nil {
		fmt.Println("error with", "src", src, "dst", dst, "title", title, "url", url)
		panic(err)
	}
}
