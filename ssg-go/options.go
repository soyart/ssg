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

	// Hook takes in a path and reads file data,
	// returning modified output to be written at destination
	Hook func(path string, data []byte) (output []byte, err error)

	// HookGenerate takes in converted HTML bytes
	// and returns modified HTML output (e.g. minified) to be written at destination
	HookGenerate func(generatedHtml []byte) (output []byte, err error)

	// Pipeline is called during directory tree walks.
	// ssg-go provides path and data from the file,
	// and Pipeline is free to do whatever it wants.
	Pipeline func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error)

	options struct {
		hook         Hook
		hookGenerate HookGenerate
		pipelines    []Pipeline
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

// Caching allows outputs to be built and retained for later use.
// This is enabled in [Build].
func Caching() Option {
	return func(s *Ssg) {
		s.options.caching = true
	}
}

// Writers set the number of concurrent output writers.
func Writers(u uint) Option {
	return func(s *Ssg) {
		s.options.writers = int(u)
	}
}

// WithHook will make [Ssg] call hook(path, fileContent)
// on every unignored files.
func WithHook(hook Hook) Option {
	return func(s *Ssg) {
		s.options.hook = hook
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
func WithPipelines(pipes ...interface{}) Option {
	return func(s *Ssg) {
		pipelines := make([]Pipeline, len(pipes))
		for i, p := range pipes {
			switch pipe := p.(type) {
			case Pipeline:
				pipelines[i] = pipe

			case func(string, []byte, fs.DirEntry) (string, []byte, fs.DirEntry, error):
				pipelines[i] = pipe

			case func(*Ssg) Pipeline:
				pipelines[i] = pipe(s)

			default:
				panic(fmt.Errorf("unexpected pipelines[%d] type '%s'", i, reflect.TypeOf(p).String()))
			}
		}
		s.options.pipelines = pipelines
	}
}
