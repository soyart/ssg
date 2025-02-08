package main

import (
	"fmt"
	"os"

	"github.com/soyart/ssg/ssg-go/v2"
)

func main() {
	target := "testdata/myblog"
	if len(os.Args) >= 2 {
		target = os.Args[1]
	}

	root, err := ssg.Walk(target)
	if err != nil {
		panic(err)
	}

	pront(&root)
	fmt.Println("----")
	fmt.Println(root)
}

func pront(node *ssg.Node) {
	fmt.Printf("pront: %s\n", node.Path)
	for i := range node.Children {
		child := &node.Children[i]
		pront(child)
	}
}
