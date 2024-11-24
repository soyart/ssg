package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

const (
	yes string = "1"

	envNoHtml    string = "NO_HTML"
	envNoHtmlAll string = "NO_HTML_ALL"
	envNoCss     string = "NO_CSS"
	envNoJson    string = "NO_JSON"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprint(os.Stdout, "usage: ssg src dst title base_url\n")
		syscall.Exit(1)
	}

	f := negateFromEnvs(soyweb.MinifyFlags{
		MinifyHtml:    true,
		MinifyHtmlAll: true,
		MinifyCss:     true,
		MinifyJson:    true,
	})

	opts := append(
		[]ssg.Option{ssg.ParallelWritesEnv()},
		soyweb.SsgOptions(soyweb.Flags{MinifyFlags: f})...,
	)

	src, dst, title, url := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	s := ssg.NewWithOptions(src, dst, title, url, opts...)

	err := s.Generate()
	if err != nil {
		panic(err)
	}
}

type envs struct {
	noHtml    bool
	noHtmlAll bool
	noCss     bool
	noJson    bool
}

func parseEnvs() envs {
	result := envs{}
	if os.Getenv(envNoHtml) == yes {
		result.noHtml = true
	}
	if os.Getenv(envNoHtmlAll) == yes {
		result.noHtmlAll = true
	}
	if os.Getenv(envNoCss) == yes {
		result.noCss = true
	}
	if os.Getenv(envNoJson) == yes {
		result.noJson = true
	}
	return result
}

func negateFromEnvs(f soyweb.MinifyFlags) soyweb.MinifyFlags {
	e := parseEnvs()
	if e.noHtml {
		f.MinifyHtml = false
	}
	if e.noHtmlAll {
		f.MinifyHtmlAll = false
	}
	if e.noCss {
		f.MinifyCss = false
	}
	if e.noJson {
		f.MinifyJson = false
	}
	return f
}
