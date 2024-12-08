package main

import (
	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	s := ssg.New(
		"testdata/myblog/src",
		"testdata/myblog/dst",
		"TestArticleListGenerator",
		"https://mybloggyblogblog.com",
	)

	generator := soyweb.ArticleListGenerator(s.Src, s.Dst, s.ImplDefault())
	optGenerator := ssg.WithImpl(generator)
	s.With(optGenerator)

	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
