package main

import (
	"fmt"
	"path/filepath"

	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
	"github.com/soyart/ssg/soyweb"
)

type cli struct {
	Build   *cmdBuild `arg:"subcommand:build"`
	Copy    *cmdOther `arg:"subcommand:copy"`
	Clean   *cmdOther `arg:"subcommand:clean"`   // Same with cleanup
	CleanUp *cmdOther `arg:"subcommand:cleanup"` // Same with clean
}

type manifests struct {
	Manifests []string `arg:"positional" help:"Paths to JSON manifests"`
}

type cmdBuild struct {
	manifests
	NoCleanup     bool `arg:"--no-cleanup" help:"Skip cleanup stage"`
	NoCopy        bool `arg:"--no-copy" help:"Skip scopy stage"`
	NoBuild       bool `arg:"--no-build" help:"Skip build stage"`
	MinifyHtml    bool `arg:"--min-html" help:"Minify HTML outputs"`
	MinifyHtmlAll bool `arg:"--min-html-all" help:"Minify all HTML outputs"`
	MinifyCss     bool `arg:"--min-css" help:"Minify CSS files"`
	MinifyJson    bool `arg:"--min-json" help:"Minify JSON files"`
}

type cmdOther struct {
	manifests
}

type pipelineFn func(path string, data []byte) ([]byte, error)
type minifyFn func([]byte) ([]byte, error)

func main() {
	cli := cli{}
	arg.MustParse(&cli)

	run(&cli)
}

func run(c *cli) {
	stages := soyweb.StagesAll
	opts := []ssg.Option{
		ssg.ParallelWritesEnv(),
	}

	var manifests []string
	switch {
	case c.Build != nil:
		manifests = c.Build.Manifests
		opts = append(opts, ssgOptions(c)...)

		if c.Build.NoCleanup {
			stages &^= soyweb.StageCleanUp
		}
		if c.Build.NoCopy {
			stages &^= soyweb.StageCopy
		}
		if c.Build.NoBuild {
			stages &^= soyweb.StageBuild
			opts = nil
		}

	case c.Copy != nil:
		manifests = c.Copy.Manifests
		stages = soyweb.StageCopy

	case c.Clean != nil:
		c.CleanUp = c.Clean
		fallthrough

	case c.CleanUp != nil:
		manifests = c.CleanUp.Manifests
		stages = soyweb.StageCleanUp
	}

	if len(manifests) == 0 {
		manifests = []string{"./manifest.json"}
	}

	for i := range manifests {
		build(manifests[i], stages, opts...)
	}
}

func ssgOptions(c *cli) []ssg.Option {
	minifiers := make(map[string]minifyFn)
	opts := []ssg.Option{}

	if c.Build.MinifyHtmlAll {
		minifiers[".html"] = soyweb.MinifyHtml
	}
	if c.Build.MinifyCss {
		minifiers[".css"] = soyweb.MinifyCss
	}
	if c.Build.MinifyJson {
		minifiers[".json"] = soyweb.MinifyJson
	}

	pipeline := pipelineMinify(minifiers)
	if pipeline != nil {
		opts = append(opts, ssg.Pipeline(pipeline))
	}

	if c.Build.MinifyHtml {
		opts = append(opts, ssg.Hook(soyweb.MinifyHtml))
	}

	return opts
}

func build(path string, do soyweb.Stage, opts ...ssg.Option) {
	err := soyweb.ApplyFromManifest(path, do, opts...)
	if err != nil {
		panic(err)
	}
}

func pipelineMinify(m map[string]minifyFn) pipelineFn {
	if len(m) == 0 {
		return nil
	}

	return func(path string, data []byte) ([]byte, error) {
		ext := filepath.Ext(path)
		f, ok := m[ext]
		if !ok {
			return data, nil
		}

		b, err := f(data)
		if err != nil {
			return nil, fmt.Errorf("error from minifier for '%s'", ext)
		}

		return b, nil
	}
}
