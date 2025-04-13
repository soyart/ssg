package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg/soyweb"
	"github.com/soyart/ssg/ssg-go"
)

type noMinifyFlags struct {
	NoMinifyHtmlGenerate bool `arg:"--no-min-html,env:NO_MIN_HTML" help:"Do not minify converted HTML outputs"`
	NoMinifyHtmlCopy     bool `arg:"--no-min-html-copy,env:NO_MIN_HTML_COPY" help:"Do not minify all copied HTML"`
	NoMinifyCss          bool `arg:"--no-min-css,env:NO_MIN_CSS" help:"Do not minify CSS files"`
	NoMinifyJs           bool `arg:"--no-min-js,env:NO_MIN_JS" help:"Do not minify Javascript files"`
	NoMinifyJson         bool `arg:"--no-min-json,env:NO_MIN_JSON" help:"Do not minify JSON files"`
}

type cli struct {
	Src string `arg:"positional" default:"src"`
	Dst string `arg:"positional" default:"dist"`

	soyweb.FlagsNoMinify
}

type minifier struct {
	src  string
	dst  string
	dist []ssg.OutputFile

	soyweb.FlagsNoMinify
}

func main() {
	c := cli{}
	arg.MustParse(&c)

	err := run(&c)
	if err != nil {
		panic(err)
	}
}

func run(c *cli) error {
	m := minifier{
		src:           c.Src,
		dst:           c.Dst,
		FlagsNoMinify: c.FlagsNoMinify,
	}

	writes, err := m.minify()
	if err != nil {
		return err
	}

	err = ssg.WriteOutSlice(writes, ssg.GetEnvWriters())
	if err != nil {
		return fmt.Errorf("error writing out: %w", err)
	}

	return nil
}

func (m *minifier) minify() ([]ssg.OutputFile, error) {
	if m.src == m.dst {
		return nil, fmt.Errorf("dst overwrites src: %s", m.src)
	}

	stat, err := os.Stat(m.src)
	if err != nil {
		return nil, fmt.Errorf("failed to stat src: %w", err)
	}
	if stat.IsDir() {
		err = filepath.WalkDir(m.src, m.walk)
		if err != nil {
			return nil, err
		}

		return m.dist, nil
	}

	data, err := os.ReadFile(m.src)
	if err != nil {
		return nil, err
	}
	output, err := soyweb.MinifyAll(m.src, data)
	if err != nil {
		output = data
	}

	out := ssg.Output(m.dst, m.src, output, stat.Mode().Perm())
	return []ssg.OutputFile{out}, nil
}

func (m *minifier) walk(path string, d fs.DirEntry, e error) error {
	if e != nil {
		return e
	}
	if d.IsDir() {
		return nil
	}
	info, err := d.Info()
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(m.src, path)
	if err != nil {
		return err
	}
	dst := filepath.Join(m.dst, rel)
	if m.Skip(filepath.Ext(path)) {
		copied, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m.dist = append(m.dist, ssg.Output(dst, path, copied, info.Mode().Perm()))
		return nil
	}

	minified, err := soyweb.MinifyFile(path)
	if err != nil {
		return err
	}
	m.dist = append(m.dist, ssg.Output(dst, path, minified, info.Mode().Perm()))
	return nil
}
