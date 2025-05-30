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

	// Pipeline is called for each visit during a dir walk.
	// ssg-go provides for pipeline the path and data the file being visited,
	// and Pipeline is free to do whatever it wants with that information.
	// The pipeline could use ErrSkipCore or ErrBreakPipelines as return value
	// to control subsequent operations of the walk.
	Pipeline func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error)

	Options interface {
		Hooks() []Hook
		HooksGenerate() []HookGenerate
		Pipelines() []Pipeline
		Caching() bool
		Writers() int
	}

	options struct {
		hooks        []Hook
		hookGenerate []HookGenerate
		pipelines    []Pipeline
		caching      bool
		writers      int
	}
)

func (o options) Hooks() []Hook                 { return o.hooks }
func (o options) HooksGenerate() []HookGenerate { return o.hookGenerate }
func (o options) Pipelines() []Pipeline         { return o.pipelines }
func (o options) Caching() bool                 { return o.caching }
func (o options) Writers() int                  { return o.writers }

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
	return func(s *Ssg) { s.options.caching = true }
}

// Writers set the number of concurrent output writers.
func Writers(u uint) Option {
	return func(s *Ssg) { s.options.writers = int(u) }
}

// WithHooks will make [Ssg] iterate through hooks and call hook(path, fileContent)
// on every unignored files.
func WithHooks(hooks ...Hook) Option {
	return func(s *Ssg) { s.options.hooks = append(s.options.hooks, hooks...) }
}

// PrependHooks prepends [hooks] to Ssg's existing hook options.
// e.g. if we have a Ssg with existing hook options = [hook1, hook2]
// then PrependHooks(hook3, hook4) will make
func PrependHooks(hooks ...Hook) Option {
	return func(s *Ssg) {
		hooks = append(hooks, s.options.hooks...)
		s.options.hooks = hooks
	}
}

// WithHooksGenerate assigns hook to be called on full output of files
// that will be converted by ssg from Markdown to HTML.
func WithHooksGenerate(hooks ...HookGenerate) Option {
	return func(s *Ssg) { s.options.hookGenerate = append(s.options.hookGenerate, hooks...) }
}

// WithPipelines returns an option that allows caller
// to set the pipeline(s) chained together for each file visit,
// in a fashion similar to middlewares in HTTP frameworks.
//
// pipelines can be of type Pipeline or func(*Ssg) Pipeline
func WithPipelines(pipes ...any) Option {
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
