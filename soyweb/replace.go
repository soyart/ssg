package soyweb

import (
	"bytes"
	"fmt"

	"github.com/soyart/ssg/ssg-go"
)

func Replacers(r Replaces) ssg.Hook {
	holders := make(map[string]string, len(r))
	for k := range r {
		holders[k] = fmt.Sprintf("${{ %s }}", k)
	}
	replacer := hookReplace{
		replaces:     r,
		placeholders: holders,
	}
	return replacer.hook
}

type hookReplace struct {
	replaces     Replaces
	placeholders map[string]string
}

func (r hookReplace) hook(path string, data []byte) ([]byte, error) {
	for k, replace := range r.replaces {
		holder := r.placeholders[k]
		switch replace.Count {
		case 0:
			data = bytes.ReplaceAll(data, []byte(holder), []byte(replace.Text))

		default:
			data = bytes.Replace(data, []byte(holder), []byte(replace.Text), int(replace.Count))
		}
	}
	return data, nil
}
