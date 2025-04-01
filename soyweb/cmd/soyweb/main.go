package main

import (
	"github.com/alexflint/go-arg"

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
	soyweb.FlagsV2
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
	var (
		manifests []string
		flags     soyweb.FlagsV2
		stages    soyweb.Stage
	)

	stages = soyweb.StageAll
	switch {
	case c.Build != nil:
		manifests, flags = c.Build.Manifests, c.Build.FlagsV2

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
		manifest := manifests[i]
		m, err := soyweb.NewManifest(manifest)
		if err != nil {
			panic(err.Error())
		}

		soyweb.ApplyManifestV2(m, flags, stages)
	}
}
