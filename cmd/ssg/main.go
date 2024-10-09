package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expecting at least 1 argument")
		syscall.Exit(1)
	}

	err := ssg.Gen(os.Args[1])
	if err != nil {
		panic(err)
	}
}
