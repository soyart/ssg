package ssg

import (
	"io/fs"
	"os"
	"strconv"
)

type (
	Option func(*Ssg)

	// HookAll takes in a path and reads file data,
	// returning modified output to be written at destination
	HookAll func(path string, data []byte) (output []byte, err error)

	// HookGenerate takes in converted HTML bytes
	// and returns modified HTML output (e.g. minified) to be written at destination
	HookGenerate func(generatedHtml []byte) (output []byte, err error)

	// Impl is called during directory tree walks.
	// ssg-go provides path and data from the file,
	// and Impl is free to do whatever it wants
	Impl func(path string, data []byte, d fs.DirEntry) error

	options struct {
		hookAll      HookAll
		hookGenerate HookGenerate
		streaming
		impl           Impl
		parallelWrites int
	}
)

// ParallelWritesEnv returns an option that sets the parallel writes
// to whatever [GetEnvParallelWrites] returns
func ParallelWritesEnv() Option {
	return func(s *Ssg) {
		writes := GetEnvParallelWrites()
		s.parallelWrites = int(writes)
	}
}

// GetEnvParallelWrites returns ENV value for parallel writes,
// or default value if illgal or undefined
func GetEnvParallelWrites() int {
	writesEnv := os.Getenv(ParallelWritesEnvKey)
	writes, err := strconv.ParseUint(writesEnv, 10, 32)
	if err == nil && writes != 0 {
		return int(writes)
	}

	return ParallelWritesDefault
}

func WriteStreaming() Option {
	return func(s *Ssg) {
		s.streaming.c = make(chan OutputFile)
	}
}

// WithHookAll will make [Ssg] call f(path, fileContent)
// on every unignored files.
func WithHookAll(f HookAll) Option {
	return func(s *Ssg) {
		s.hookAll = f
	}
}

// WithHookGenerate assigns f to be called on full output of files
// that will be converted by ssg from Markdown to HTML.
func WithHookGenerate(f HookGenerate) Option {
	return func(s *Ssg) {
		s.hookGenerate = f
	}
}

// WithImpl takes f, which will called during build process.
func WithImpl(f Impl) Option {
	return func(s *Ssg) {
		s.impl = f
	}
}
