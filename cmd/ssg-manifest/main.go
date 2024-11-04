package main

import (
	"github.com/alexflint/go-arg"

	"github.com/soyart/ssg"
)

type mainArg struct {
	ManifestPath []string `arg:"positional" help:"Path to JSON manifest"`
	NoCleanup    bool     `arg:"--no-cleanup" help:"Skip cleanup stage"`
	NoCopy       bool     `arg:"--no-copy" help:"Skip scopy stage"`
	NoBuild      bool     `arg:"--no-build" help:"Skip build stage"`
}

func main() {
	path := "./manifest.json"
	args := mainArg{}
	arg.MustParse(&args)

	stages := ssg.StagesAll
	if args.NoCleanup {
		stages &^= ssg.StageCleanUp
	}
	if args.NoCopy {
		stages &^= ssg.StageCopy
	}
	if args.NoBuild {
		stages &^= ssg.StageBuild
	}

	if len(args.ManifestPath) == 0 {
		args.ManifestPath = []string{path}
	}

	for i := range args.ManifestPath {
		build(args.ManifestPath[i], stages)
	}
}

func build(path string, do ssg.Stage) {
	err := ssg.BuildManifestFromPath(path, do)
	if err != nil {
		panic(err)
	}
}
