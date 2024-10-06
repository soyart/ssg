package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: ssg.sh src dst")
		// fmt.Println("usage: ssg.sh src dst title base_url")
		syscall.Exit(1)
	}

	site := ssg.NewSsg(os.Args[1], os.Args[2])
	err := site.Generate()
	if err != nil {
		panic(err)
	}
}
