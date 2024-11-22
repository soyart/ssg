package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg/soyweb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stdout, "expecting 1 argument")
		syscall.Exit(1)
	}

	filename := os.Args[1]
	doc, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to read input file '%s': %v\n", filename, err)
		syscall.Exit(2)
	}

	minified, err := soyweb.MinifyAll(filename, doc)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", minified)
}
