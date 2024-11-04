package main

import (
	"flag"
	"fmt"

	"github.com/soyart/ssg"
)

var (
	noCleanup bool
	noCopy    bool
	noBuild   bool
)

func init() {
	flag.BoolVar(&noCleanup, "nocleanup", false, "skip cleanup stage")
	flag.BoolVar(&noCopy, "nocopy", false, "skip build stage")
	flag.BoolVar(&noBuild, "nobuild", false, "skip copy stage")
	flag.Parse()
}

func main() {
	path := "./manifest.json"
	args := flag.Args()
	l := len(args)

	stages := ssg.StagesAll
	if noCleanup {
		fmt.Println("nocleanup")
		stages &^= ssg.StageCleanUp
	}
	if noCopy {
		fmt.Println("nocopy")
		stages &^= ssg.StageCopy
	}
	if noBuild {
		fmt.Println("nobuild")
		stages &^= ssg.StageBuild
	}

	if l == 0 {
		args = []string{path}
	}

	for i := range args {
		build(args[i], stages)
	}
}

func build(path string, do ssg.Stage) {
	err := ssg.BuildManifestFromPath(path, do)
	if err != nil {
		panic(err)
	}
}
