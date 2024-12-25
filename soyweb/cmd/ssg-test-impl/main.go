package main

import (
	"os"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	src := "./testdata/myblog/src"
	dst := "./testdata/myblog/dst-cmd"
	title := "TestIndexGeneratorCMD"
	url := "https://myblog.com"

	if len(os.Args) >= 2 {
		src = os.Args[1]
	}
	if len(os.Args) >= 3 {
		dst = os.Args[2]
	}
	if len(os.Args) >= 4 {
		title = os.Args[3]
	}
	if len(os.Args) >= 5 {
		title = os.Args[4]
	}

	err := ssg.Generate(
		src, dst, title, url,
		soyweb.IndexGenerator(),
	)
	if err != nil {
		panic(err)
	}
}
