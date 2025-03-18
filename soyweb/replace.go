package soyweb

import (
	"bytes"
	"fmt"

	"github.com/soyart/ssg/ssg-go"
)

func Replacer(r Replaces) ssg.Hook {
	holders := make(map[string]string, len(r))
	for k := range r {
		holders[k] = placeholder(k)
	}
	replacer := hookReplace{
		replaces:     r,
		placeholders: holders,
	}
	return replacer.hook
}

func placeholder(s string) string {
	return fmt.Sprintf("${{ %s }}", s)
}

type hookReplace struct {
	replaces     Replaces
	placeholders map[string]string
}

func (r hookReplace) hook(path string, data []byte) ([]byte, error) {
	for k, rp := range r.replaces {
		var err error
		data, err = replace(data, []byte(k), rp)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func replace(data, holder []byte, replace ReplaceTarget) ([]byte, error) {
	switch replace.Count {
	case 0:
		data = bytes.ReplaceAll(data, []byte(holder), []byte(replace.Text))
	default:
		data = bytes.Replace(data, []byte(holder), []byte(replace.Text), int(replace.Count))
	}
	return data, nil
}
