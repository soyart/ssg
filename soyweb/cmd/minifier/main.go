package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg/soyweb"
)

// TODO: cli option for walking dir
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stdout, "expecting 1 argument")
		syscall.Exit(1)
	}

	filename := os.Args[1]
	minified, err := soyweb.MinifyFile(filename)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", minified)
}
