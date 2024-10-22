package main

import (
	"os"

	"github.com/soyart/ssg"
)

func main() {
	path := "./manifest.json"
	l := len(os.Args)

	if l < 2 {
		build(path)
		return
	}

	for i := 1; i < l-1; i++ {
		build(os.Args[i])
	}
}

func build(path string) {
	err := ssg.BuildManifest(path)
	if err != nil {
		panic(err)
	}
}
