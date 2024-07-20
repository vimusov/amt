package main

import "sync"

type errKeeper struct {
	mut sync.Mutex
	wg  sync.WaitGroup
	err error
}

func (e *errKeeper) set(err error) {
	defer e.mut.Unlock()
	e.mut.Lock()
	if e.err == nil {
		e.err = err
	}
}

func (e *errKeeper) done() {
	e.wg.Done()
}

func (e *errKeeper) get() error {
	e.wg.Wait()
	return e.err
}

func newErrKeeper(delta int) *errKeeper {
	result := &errKeeper{err: nil}
	result.wg.Add(delta)
	return result
}
