package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

type cli struct {
	Src string `arg:"positional" default:"src"`
	Dst string `arg:"positional" default:"dist"`

	soyweb.NoMinifyFlags
}

type minifier struct {
	src  string
	dst  string
	dist []ssg.OutputFile

	soyweb.NoMinifyFlags
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
		NoMinifyFlags: c.NoMinifyFlags,
	}

	writes, err := m.minify()
	if err != nil {
		return err
	}

	err = ssg.WriteOut(writes, ssg.GetEnvConcurrent())
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

	output, err := soyweb.MinifyFile(m.src)
	if err != nil {
		output, err = os.ReadFile(m.src)
		if err != nil {
			return nil, err
		}
	}

	out := ssg.Output(m.dst, output, stat.Mode().Perm())
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
	if m.NoMinifyFlags.Skip(filepath.Ext(path)) {
		copied, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		m.dist = append(m.dist, ssg.Output(dst, copied, info.Mode().Perm()))
		return nil
	}

	minified, err := soyweb.MinifyFile(path)
	if err != nil {
		return err
	}

	m.dist = append(m.dist, ssg.Output(dst, minified, info.Mode().Perm()))
	return nil
}
