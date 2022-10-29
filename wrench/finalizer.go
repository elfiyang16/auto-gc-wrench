package wrench

import (
	"runtime"
	"sync/atomic"
)

type finalizerCallback func()

type finalizerRef struct {
	parent *finalizer
}

type finalizer struct {
	ref      *finalizerRef
	callback finalizerCallback
	stopped  int32
}

func finalizerHandler(f *finalizerRef) {
	if atomic.LoadInt32(&f.parent.stopped) > 0 {
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
	atomic.StoreInt32(&f.stopped, 1)
}
