package soyweb

import (
	"errors"
	"testing"
)

func TestErr(t *testing.T) {
	_, err := ExtToMediaType("foo")
	if !errors.Is(err, ErrNotSupported) {
		t.Fatal("unexpected result of errors.Is")
	}

	_, err = ExtToFn("foo")
	if !errors.Is(err, ErrNotSupported) {
		t.Fatal("unexpected result of errors.Is")
	}
}
