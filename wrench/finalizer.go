package wrench

import (
	"go.uber.org/atomic"
	"runtime"
)

type finalizerCallback func()

type finalizerRef struct {
	parent *finalizer
}

type finalizer struct {
	ref      *finalizerRef
	callback finalizerCallback
	stopped  atomic.Int32
}

func finalizerHandler(f *finalizerRef) {
	if f.parent.stopped.Load() > 0 {
		return
	}
	f.parent.callback()
	// reset it to prep for next GC
	runtime.SetFinalizer(f, finalizerHandler)
}

func newFinalizer(callback finalizerCallback) *finalizer {
	f := &finalizer{
		callback: callback,
	}
	f.ref = &finalizerRef{
		parent: f,
	}
	runtime.SetFinalizer(f.ref, finalizerHandler)
	// clears ptr to the ref, so GC will find it and setFinaliser be triggered
	f.ref = nil
	return f
}

func (f *finalizer) stop() {
	f.stopped.Store(1)
}
