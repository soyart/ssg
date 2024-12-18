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
		impl         Impl
		caching      bool
		concurrent   int
	}
)

// ConcurrentFromEnv returns an option that sets the parallel writes
// to whatever [GetEnvConcurrent] returns
func ConcurrentFromEnv() Option {
	return func(s *Ssg) {
		writes := GetEnvConcurrent()
		s.concurrent = int(writes)
	}
}

// GetEnvConcurrent returns ENV value for parallel writes,
// or default value if illgal or undefined
func GetEnvConcurrent() int {
	writesEnv := os.Getenv(ConcurrencyEnvKey)
	writes, err := strconv.ParseUint(writesEnv, 10, 32)
	if err == nil && writes != 0 {
		return int(writes)
	}

	return ConcurrentDefault
}

func Caching() Option {
	return func(s *Ssg) {
		s.options.caching = true
	}
}

func Concurrent(u uint) Option {
	return func(s *Ssg) {
		s.options.concurrent = int(u)
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

// WithImpl takes f, which will called during build process.
// Ignored files, _header.html and _footer.html
// are skipped by ssg-go.
func WithImpl(f Impl) Option {
	return func(s *Ssg) {
		s.impl = f
	}
}
