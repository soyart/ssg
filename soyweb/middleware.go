package soyweb

import (
	"fmt"
	"io/fs"

	"github.com/soyart/ssg"
)

func Chain(wares ...ssg.Pipeline) ssg.Pipeline {
	return func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error) {
		var err error
		for i := range wares {
			mw := wares[i]
			path, data, d, err = mw(path, data, d)
			if err != nil {
				return "", nil, nil, fmt.Errorf("[middleware %d] error: %w", i+1, err)
			}
		}

		return path, data, d, nil
	}
}
