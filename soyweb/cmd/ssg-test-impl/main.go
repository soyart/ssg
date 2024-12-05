package main

import (
	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	s := ssg.New("testdata/johndoe.com/src", "testdata/johndoe.com/dst", "TestWithImpl", "https://johndoe.com")
	generator := ssg.WithImpl(soyweb.ArticleGenerator(s.ImplDefault()))
	s.With(generator)

	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
