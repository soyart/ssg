package ssg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func (s *Ssg) GenerateStreaming() error {
	if s.streaming == nil {
		panic("nil streaming channel")
	}

	var wg sync.WaitGroup

	var errGen error
	wg.Add(1)
	go func() {
		defer wg.Done()
		dist, err := s.buildV2()
		if err != nil {
			errGen = err
		}
		if len(dist) != 0 {
			panic("dist is not empty")
		}

		close(s.streaming)
	}()

	var errWrites error
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := WriteOutStreaming(s.streaming, s.parallelWrites)
		if err != nil {
			errWrites = err
		}
	}()

	wg.Wait()

	if errGen != nil {
		return errGen
	}
	if errWrites != nil {
		return errWrites
	}

	return nil
}

func WriteOutStreaming(c <-chan OutputFile, parallelWrites int) error {
	if parallelWrites == 0 {
		parallelWrites = ParallelWritesDefault
	}

	wg := new(sync.WaitGroup)
	errs := make(chan writeError)
	guard := make(chan struct{}, parallelWrites)

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

			fmt.Fprintln(os.Stdout, w.target)

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
		return errors.Join(wErrs...)
	}

	return nil
}
