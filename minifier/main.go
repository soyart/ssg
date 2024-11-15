package main

import (
	"fmt"
	"github.com/soyart/ssg"
)

func main() {
	fmt.Println("test visibility", string(ssg.ToHtml([]byte("Hello, world!"))))
}
