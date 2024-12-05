package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/soyart/ssg"
)

func main() {
	s := ssg.New("soyweb/testdata/johndoe.com/src", "soyweb/testdata/johndoe.com/dst", "TestWithImpl", "https://johndoe.com")
	implDefault := s.ImplDefault()

	customFilter := ssg.WithImpl(func(path string, data []byte, d fs.DirEntry) error {
		fmt.Fprintf(os.Stdout,
			"test log: path=%s, lenData=%d, isDir=%v\n",
			path, len(data), d.IsDir(),
		)

		if filepath.Base(path) == "755" {
			fmt.Fprintf(os.Stdout, "skipping file %s", path)
			return nil
		}

		return implDefault(path, data, d)
	})

	s.With(customFilter)
	err := s.Generate()
	if err != nil {
		panic(err)
	}
}
