package main

import (
	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	err := ssg.Generate(
		"testdata/myblog/src",
		"testdata/myblog/dst",
		"TestIndexGenerator",
		"https://myblog.com",
		soyweb.IndexGenerator(),
	)
	if err != nil {
		panic(err)
	}
}
