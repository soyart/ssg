package main

import (
	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg/soyweb"
	"github.com/soyart/ssg/ssg-go"
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
	soyweb.Flags

	NoCleanup bool `arg:"--no-cleanup" help:"Skip cleanup stage"`
	NoCopy    bool `arg:"--no-copy" help:"Skip scopy stage"`
	NoBuild   bool `arg:"--no-build" help:"Skip build stage"`
}

type cmdOther struct {
	manifests
}

func main() {
	c := cli{}
	arg.MustParse(&c)

	run(&c)
}

func run(c *cli) {
	stages := soyweb.StagesAll
	opts := []ssg.Option{
		ssg.WritersFromEnv(),
	}

	var manifests []string
	switch {
	case c.Build != nil:
		manifests = c.Build.Manifests
		opts = append(opts, soyweb.SsgOptions(c.Build.Flags)...)

		if c.Build.NoCleanup {
			stages.Skip(soyweb.StageCleanUp)
		}
		if c.Build.NoCopy {
			stages.Skip(soyweb.StageCopy)
		}
		if c.Build.NoBuild {
			stages.Skip(soyweb.StageBuild)
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
		err := soyweb.ApplyFromManifest(manifests[i], stages, opts...)
		if err != nil {
			panic(err)
		}
	}
}
