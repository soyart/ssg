package main

import (
	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
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
	NoCleanup bool `arg:"--no-cleanup" help:"Skip cleanup stage"`
	NoCopy    bool `arg:"--no-copy" help:"Skip scopy stage"`
	NoBuild   bool `arg:"--no-build" help:"Skip build stage"`
}

type cmdOther struct {
	manifests
}

func main() {
	cli := cli{}
	arg.MustParse(&cli)
	cli.run()
}

func (s *cli) run() {
	stages := ssg.StagesAll
	var manifests []string

	switch {
	case s.Build != nil:
		manifests = s.Build.Manifests
		if s.Build.NoCleanup {
			stages &^= ssg.StageCleanUp
		}
		if s.Build.NoCopy {
			stages &^= ssg.StageCopy
		}
		if s.Build.NoBuild {
			stages &^= ssg.StageBuild
		}

	case s.Copy != nil:
		manifests = s.Copy.Manifests
		stages = ssg.StageCopy

	case s.Clean != nil:
		s.CleanUp = s.Clean
		fallthrough

	case s.CleanUp != nil:
		manifests = s.CleanUp.Manifests
		stages = ssg.StageCleanUp
	}

	if len(manifests) == 0 {
		manifests = []string{"./manifest.json"}
	}
	for i := range manifests {
		build(manifests[i], stages)
	}
}

func build(path string, do ssg.Stage) {
	err := ssg.ApplyFromManifest(path, do, ssg.ParallelWritesEnv())
	if err != nil {
		panic(err)
	}
}
