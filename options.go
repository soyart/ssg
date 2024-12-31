package ssg

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
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

	// Pipeline is called during directory tree walks.
	// ssg-go provides path and data from the file,
	// and Pipeline is free to do whatever it wants
	Pipeline func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error)

	options struct {
		hookAll      HookAll
		hookGenerate HookGenerate
		pipeline     Pipeline
		caching      bool
		writers      int
	}
)

// WritersFromEnv returns an option that sets the parallel writes
// to whatever [GetEnvWriters] returns
func WritersFromEnv() Option {
	return func(s *Ssg) {
		writes := GetEnvWriters()
		s.options.writers = int(writes)
	}
}

// GetEnvWriters returns ENV value for parallel writes,
// or default value if illgal or undefined
func GetEnvWriters() int {
	writesEnv := os.Getenv(WritersEnvKey)
	writes, err := strconv.ParseUint(writesEnv, 10, 32)
	if err == nil && writes != 0 {
		return int(writes)
	}

	return WritersDefault
}

func Caching() Option {
	return func(s *Ssg) {
		s.options.caching = true
	}
}

func Writers(u uint) Option {
	return func(s *Ssg) {
		s.options.writers = int(u)
	}
}

// WithHookAll will make [Ssg] call hook(path, fileContent)
// on every unignored files.
func WithHookAll(hook HookAll) Option {
	return func(s *Ssg) {
		s.options.hookAll = hook
	}
}

// WithHookGenerate assigns hook to be called on full output of files
// that will be converted by ssg from Markdown to HTML.
func WithHookGenerate(hook HookGenerate) Option {
	return func(s *Ssg) {
		s.options.hookGenerate = hook
	}
}

// WithPipelines returns an option that set option.pipeline to
// pipelines chained together.
//
// pipelines can be of type Pipeline or func(*Ssg) Pipeline
func WithPipelines(pipelines ...any) Option {
	return func(s *Ssg) {
		pipes := make([]Pipeline, len(pipelines))
		for i, f := range pipelines {
			switch actual := f.(type) {
			case Pipeline:
				pipes[i] = actual

			case func(*Ssg) Pipeline:
				pipes[i] = actual(s)

			default:
				panic(fmt.Errorf("[pipeline %d] unexpected pipeline type '%s'", i+1, reflect.TypeOf(f).String()))
			}
		}

		s.options.pipeline = Chain(pipes...)
	}
}

func Chain(pipes ...Pipeline) Pipeline {
	return func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error) {
		var err error
		for i := range pipes {
			path, data, d, err = pipes[i](path, data, d)
			if err != nil {
				return "", nil, nil, fmt.Errorf("[middleware %d] error: %w", i+1, err)
			}
		}
		return path, data, d, nil
	}
}
