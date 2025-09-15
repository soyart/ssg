package ssg

import "io/fs"

// OutputFile is the main output struct for ssg-go.
//
// Its values are not supposed to be changed by other packages,
// and thus the only ways other packages can work with OutputFile
// is via the constructor [Output] and the type's getter methods.
type OutputFile struct {
	target     string
	originator string
	data       []byte
	perm       fs.FileMode
}

// Outputs is any collection out OutputFile.
// It could be a simple Go slice that later get iterated and written to disk,
// or a channel (as with outputsV1)
type Outputs interface {
	Add(...OutputFile)
}

type outputsV1 struct {
	stream chan<- OutputFile
}

type buildOutput struct {
	cacheOutput bool
	writer      Outputs      // Main outputs
	files       []string     // Input files read (not ignored)
	cache       []OutputFile // Cache of main outputs
}

func NewOutputsStreaming(c chan<- OutputFile) Outputs {
	return outputsV1{stream: c}
}

func (o outputsV1) Add(outputs ...OutputFile) {
	for i := range outputs {
		o.stream <- outputs[i]
	}
}

func (b *buildOutput) Add(outputs ...OutputFile) {
	if b.cacheOutput {
		b.cache = append(b.cache, outputs...)
	}
	if b.writer != nil {
		b.writer.Add(outputs...)
	}
}
