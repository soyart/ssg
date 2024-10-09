package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("usage: ssg.sh src dst title base_url")
		syscall.Exit(1)
	}

	src, dst, _, baseUrl := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	s := ssg.New(src, dst)
	if err := s.Generate(baseUrl); err != nil {
		panic(err)
	}
}
