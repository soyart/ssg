package main

import (
	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	s := ssg.New(
		"testdata/johndoe.com/src",
		"testdata/johndoe.com/dst",
		"TestWithImpl",
		"https://johndoe.com",
	)

	generator := soyweb.ArticleGeneratorMarkdown(s.Src, s.Dst, s.ImplDefault())
	optGenerator := ssg.WithImpl(generator)
	s.With(optGenerator)

	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
