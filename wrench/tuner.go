package wrench

import (
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"sync/atomic"
)

var (
	maxGCPercent     uint32 = 500
	minGCPercent     uint32 = 50
	defaultGCPercent uint32 = 100
)

func init() {
	gogc, err := strconv.ParseInt(os.Getenv("GOGC"), 10, 32))
	if err != nil {
		return
	}
	defaultGCPercent = uint32(gogc)
}

// Tuning sets the threshold of live heap target which will be respected by gc tuner.
// When Tuning, the env GOGC will not be take effect.
// threshold: disable tuning if threshold == 0
func Tuning(threshold uint64){
	if threshold <= 0 && globalTuner != nil {
		globalTuner.stop()
		globalTuner = nil
		return
	}
	if globalTuner == nil {
		globalTuner = newTuner(threshold)
		return
	}
	globalTuner.setThreshold(threshold)

}

func GetGCPercent() uint32 {
	if globalTuner == nil {
		return defaultGCPercent
	}
	return globalTuner.getGCPercent()
}

func GetMaxGCPercent() uint32 {
	return atomic.LoadUint32(&maxGCPercent)
}

func SetMaxGCPercent(n uint32) uint32 {
	return atomic.SwapUint32(&maxGCPercent, n)
}

func GetMinGCPercent() uint32 {
	return atomic.LoadUint32(&minGCPercent)
}

func SetMinGCPercent(n uint32) uint32 {
	return atomic.SwapUint32(&minGCPercent, n)
}


// only one tuner / process
var globalTuner *tuner = nil

/* Heap
 _______________  => limit: host/cgroup memory hard limit
|               |
|---------------| => threshold: increase GCPercent when gc_trigger < threshold
|               |
|---------------| => gc_trigger: heap_live + heap_live * GCPercent / 100 -> default with 100
|               |
|---------------|
|   heap_live   |
|_______________|
Go runtime only trigger GC when hit gc_trigger which affected by GCPercent and heap_live.
So we can change GCPercent dynamically to tuning GC performance.
*/
type tuner struct {
	finalizer *finalizer
	gcPercent uint32
	threshold uint64 // high water level, in bytes
}

func newTuner(threshold uint64)*tuner{
	t := &tuner{
		gcPercent: defaultGCPercent,
		threshold: threshold,
	}
	t.finalizer = newFinalizer(t.tuning)
	return t
}

// tuning is the callback passed to finalizer of the tuner object ptr.
// It reads heap in use, and the set threshold, and dynamically set the GC percentage based on calculated target.
func (t *tuner) tuning (){
	inUse := readMemoryInuse()
	threshold := t.getThreshold()
	// invalid threshold, bypass gc tunning
	if threshold <= 0 {
		return
	}
	t.setGCPercent(calcGCPercent(inUse, threshold))
	return
}


func (t *tuner) stop() {
	t.finalizer.stop()
}

func (t *tuner) setThreshold (threshold uint64){
	atomic.StoreUint64(&t.threshold, threshold)
}

func (t *tuner) getThreshold ()uint64{
	atomic.LoadUint64(&t.threshold)
}

func (t *tuner) setGCPercent(percentage uint32) uint32{
	atomic.StoreUint32(&t.gcPercent, percentage)
	return uint32(debug.SetGCPercent(int(percentage)))
}

func (t *tuner) getGCPercent() uint32 {
	return atomic.LoadUint32(&t.gcPercent)
}

// inUse -> heapLive
// threshold = inUse + inUse * (gcPercent / 100)
// => gcPercent = (threshold - inUse) / inUse * 100
// if threshold <= inUse*2, so gcPercent <= 100, and GC positively to avoid OOM
// if threshold > inUse*2, so gcPercent > 100, and GC negatively to reduce GC times
func calcGCPercent(inUse, threshold uint64) uint32 {
	if inUse == 0 || threshold == 0 { // 1. invalid -> default
		return defaultGCPercent
	}
	if threshold <= inUse { // 2. threshold <= live heap -> GC aggresively
		return minGCPercent
	}
	// 3. calc ideal gcPercent within [lowerBound, upperBound]
	gcPercent := uint32(math.Floor(float64(threshold-inUse)/float64(inUse)*100))
	if gcPercent < minGCPercent {
		return minGCPercent
	} else if gcPercent > maxGCPercent{
		return maxGCPercent
	}
	return gcPercent
}

