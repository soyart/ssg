package main

import (
	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
)

type cli struct {
	Build   *struct{} `arg:"subcommand:build"` // Default (no subcommand) is also to build
	Copy    *struct{} `arg:"subcommand:copy"`
	Clean   *struct{} `arg:"subcommand:clean"`
	CleanUp *struct{} `arg:"subcommand:cleanup"` // Same with clean

	Manifests []string `arg:"-m,--manifest" help:"Path to JSON manifest"`
	NoCleanup bool     `arg:"--no-cleanup" help:"Skip cleanup stage"`
	NoCopy    bool     `arg:"--no-copy" help:"Skip scopy stage"`
	NoBuild   bool     `arg:"--no-build" help:"Skip build stage"`
}

func main() {
	cli := cli{}
	arg.MustParse(&cli)
	cli.run()
}

func (s *cli) run() {
	stages := ssg.StagesAll

	if len(s.Manifests) == 0 {
		s.Manifests = []string{"./manifest.json"}
	}

	switch {
	case s.Copy != nil:
		stages = ssg.StageCopy

	case s.Clean != nil, s.CleanUp != nil:
		stages = ssg.StageCleanUp

	case s.Build != nil:
		fallthrough

	default:
		if s.NoCleanup {
			stages &^= ssg.StageCleanUp
		}
		if s.NoCopy {
			stages &^= ssg.StageCopy
		}
		if s.NoBuild {
			stages &^= ssg.StageBuild
		}
	}

	for i := range s.Manifests {
		build(s.Manifests[i], stages)
	}
}

func build(path string, do ssg.Stage) {
	err := ssg.ApplyManifest(path, do)
	if err != nil {
		panic(err)
	}
}
