package ssg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type streaming struct {
	c chan OutputFile
}

func generateStreaming(s *Ssg) error {
	const bufferMultiplier = 2
	stat, err := os.Stat(s.Src)
	if err != nil {
		return fmt.Errorf("failed to stat src '%s': %w", s.Src, err)
	}

	var wg sync.WaitGroup
	s.stream = make(chan OutputFile, s.concurrent*bufferMultiplier)

	var errBuild error
	wg.Add(1)
	go func() {
		defer wg.Done()
		dist, err := s.buildV2()
		if err != nil {
			errBuild = err
		}
		if len(dist) != 0 {
			panic("dist is not empty")
		}

		close(s.stream)
	}()

	var written []string
	var errWrites error
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error

		written, err = WriteOutStreaming(s.stream, s.concurrent)
		if err != nil {
			errWrites = err
		}
	}()

	wg.Wait()

	if errBuild != nil && errWrites != nil {
		return fmt.Errorf("streaming_build_error='%w' streaming_write_error='%s'", errBuild, errWrites)
	}
	if errBuild != nil {
		return fmt.Errorf("streaming_build_error: %w", errBuild)
	}
	if errWrites != nil {
		return fmt.Errorf("streaming_write_error: %w", errWrites)
	}

	outputs := make([]OutputFile, len(written))
	for i := range written {
		outputs[i] = Output(written[i], nil, 0)
	}

	err = WriteExtraFiles(s.Url, s.Dst, outputs, stat.ModTime())
	if err != nil {
		return err
	}

	s.pront(len(written) + 2)
	return nil
}

// WriteOutStreaming blocks and concurrently writes outputs recevied from c until c is closed.
func WriteOutStreaming(c <-chan OutputFile, concurrent int) ([]string, error) {
	if concurrent == 0 {
		concurrent = 1
	}

	written := make([]string, 0)
	wg := new(sync.WaitGroup)
	errs := make(chan writeError)
	guard := make(chan struct{}, concurrent)
	mut := new(sync.Mutex)

	for o := range c {
		guard <- struct{}{}
		wg.Add(1)

		go func(w *OutputFile, wg *sync.WaitGroup) {
			defer func() {
				<-guard
				wg.Done()
			}()

			err := os.MkdirAll(filepath.Dir(w.target), os.ModePerm)
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}
				return
			}
			err = os.WriteFile(w.target, w.data, w.modeOutput())
			if err != nil {
				errs <- writeError{
					err:    err,
					target: w.target,
				}
				return
			}

			mut.Lock()
			defer mut.Unlock()
			written = append(written, w.target)
			Fprintln(os.Stdout, w.target)

		}(&o, wg)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	var wErrs []error
	for err := range errs { // Blocks here until errs is closed
		wErrs = append(wErrs, err)
	}
	if len(wErrs) > 0 {
		return nil, errors.Join(wErrs...)
	}

	return written, nil
}
