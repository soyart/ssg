package main

import "github.com/soyart/ssg"

func main() {
	err := ssg.Build("./manifest.json")
	if err != nil {
		panic(err)
	}
}
