package main

import (
	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

func main() {
	s := ssg.New(
		"testdata/myblog/src",
		"testdata/myblog/dst",
		"TestIndexGenerator",
		"https://mybloggyblogblog.com",
	)

	g := soyweb.IndexGenerator(s.Src, s.Dst, s.ImplDefault())
	optGen := ssg.WithImpl(g)
	s.With(optGen)

	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
