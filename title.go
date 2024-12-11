package ssg

import (
	"bufio"
	"bytes"
)

type TitleFrom int

const (
	TitleFromNone TitleFrom = 0
	TitleFromH1   TitleFrom = 1 << iota
	TitleFromTag

	keyTitleFromH1     = "# "      // The first h1 tag is used as document header title
	keyTitleFromTag    = ":title " // The first line starting with :title will be parsed as document header title
	targetFromH1       = "{{from-h1}}"
	targetFromTag      = "{{from-tag}}"
	placeholderFromH1  = "<title>" + targetFromH1 + "</title>"
	placeholderFromTag = "<title>" + targetFromTag + "</title>"
)

func GetTitleFromH1(markdown []byte) []byte {
	k := []byte(keyTitleFromH1)
	s := bufio.NewScanner(bytes.NewBuffer(markdown))

	var title []byte
	for s.Scan() {
		line := s.Bytes()
		if !bytes.HasPrefix(line, k) {
			continue
		}
		parts := bytes.Split(line, k)
		if len(parts) != 2 {
			continue
		}

		title = parts[1]
		break
	}

	return title
}

func GetTitleFromTag(markdown []byte) []byte {
	k := []byte(keyTitleFromTag)
	s := bufio.NewScanner(bytes.NewBuffer(markdown))

	var title []byte
	for s.Scan() {
		line := s.Bytes()
		if !bytes.HasPrefix(line, k) {
			continue
		}
		parts := bytes.Split(line, k)
		if len(parts) != 2 {
			continue
		}

		title = parts[1]
		break
	}

	return title
}

// AddTitleFromH1 finds the first h1 in markdown and uses the h1 title
// to write to <title> tag in header.
func AddTitleFromH1(d []byte, header []byte, markdown []byte) []byte {
	target := []byte(targetFromH1)
	title := GetTitleFromH1(markdown)
	if len(title) == 0 {
		header = bytes.Replace(header, target, d, 1)
		return header
	}

	header = bytes.Replace(header, target, title, 1)
	return header
}

// AddTitleFromTag finds title in markdown and then write it to <title> tag in header.
// It also deletes the tag line from markdown.
func AddTitleFromTag(
	d []byte,
	header []byte,
	markdown []byte,
) (
	[]byte,
	[]byte,
) {
	key := []byte(keyTitleFromTag)
	target := []byte(targetFromTag)
	s := bufio.NewScanner(bytes.NewBuffer(markdown))

	for s.Scan() {
		line := s.Bytes()
		if !bytes.HasPrefix(line, key) {
			continue
		}
		parts := bytes.Split(line, key)
		if len(parts) != 2 {
			continue
		}

		line = trimRightWhitespace(line)
		title := parts[1]

		header = bytes.Replace(header, target, title, 1)
		markdown = bytes.Replace(markdown, append(line, []byte{'\n', '\n'}...), nil, 1)
		return header, markdown
	}

	// Remove target and use default header string
	header = bytes.Replace(header, target, []byte(d), 1)
	return header, markdown
}

func trimRightWhitespace(b []byte) []byte {
	return bytes.TrimRightFunc(b, func(r rune) bool {
		switch r {
		case ' ', '\t':
			return true
		}
		return false
	})
}
