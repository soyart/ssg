package ssg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type streaming struct {
	c       chan OutputFile
	outputs []string
}

func (s *Ssg) generateStreaming() error {
	if s.streaming.c == nil {
		panic("nil streaming channel")
	}

	stat, err := os.Stat(s.Src)
	if err != nil {
		return fmt.Errorf("failed to stat src '%s': %w", s.Src, err)
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

		close(s.streaming.c)
	}()

	var dist []string
	var errWrites error

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error

		dist, err = WriteOutStreaming(s.streaming.c, s.parallelWrites)
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

	outputs := make([]OutputFile, len(dist))
	for i := range dist {
		outputs[i] = Output(dist[i], nil, 0)
	}

	s.dist = outputs
	sitemap, err := Sitemap(s.Dst, s.Url, stat.ModTime(), s.dist)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(s.Dst, "sitemap.xml"), []byte(sitemap), 0644)
	if err != nil {
		return err
	}

	files := bytes.NewBuffer(nil)
	writeDotFiles(s.Dst, s.dist, files)
	dotfiles := filepath.Join(s.Dst, ".files")
	err = os.WriteFile(dotfiles, files.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", dotfiles, err)
	}

	return nil
}

func WriteOutStreaming(c <-chan OutputFile, parallelWrites int) ([]string, error) {
	if parallelWrites == 0 {
		parallelWrites = ParallelWritesDefault
	}

	written := make([]string, 0)
	wg := new(sync.WaitGroup)
	errs := make(chan writeError)
	guard := make(chan struct{}, parallelWrites)
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
		return nil, errors.Join(wErrs...)
	}

	return written, nil
}
