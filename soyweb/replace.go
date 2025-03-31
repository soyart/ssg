package soyweb

import (
	"bytes"
	"fmt"

	"github.com/soyart/ssg/ssg-go"
)

func Replacer(r Replaces) ssg.Hook {
	holders := make(map[string]string)
	for k := range r {
		holders[k] = placeholder(k)
	}
	return hookReplace{replaces: r, placeholders: holders}.hook
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
		holder, ok := r.placeholders[k]
		if !ok {
			panic(fmt.Errorf("missing placeholder for key %s", k))
		}
		data, err = replace(data, []byte(holder), rp)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func replace(data, holder []byte, r ReplaceTarget) ([]byte, error) {
	if r.Count == 0 {
		return bytes.ReplaceAll(data, []byte(holder), []byte(r.Text)), nil
	}
	return bytes.Replace(data, []byte(holder), []byte(r.Text), int(r.Count)), nil
}
