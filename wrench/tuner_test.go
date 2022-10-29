package wrench

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testHeap []byte

func TestTuner(t *testing.T) {
	is := assert.New(t)
	memLimit := uint64(100 * 1024 * 1024) //100 MB
	threshold := memLimit / 2
	turner := newTuner(threshold)
	require.Equal(t, threshold, turner.threshold.Load())
	require.Equal(t, defaultGCPercent, turner.getGCPercent())

	// no heap
	testHeap = make([]byte, 1)
	runtime.GC()
	runtime.GC()
	for i := 0; i < 100; i++ {
		runtime.GC()
		require.Equal(t, maxGCPercent, turner.getGCPercent())
	}

	// 1/4 threshold
	testHeap = make([]byte, threshold/4)
	for i := 0; i < 100; i++ {
		runtime.GC()
		require.GreaterOrEqual(t, turner.getGCPercent(), defaultGCPercent)
		require.LessOrEqual(t, turner.getGCPercent(), maxGCPercent)
	}

	// 1/2 threshold
	testHeap = make([]byte, threshold/2)
	runtime.GC()
	for i := 0; i < 100; i++ {
		runtime.GC()
		require.GreaterOrEqual(t, turner.getGCPercent(), minGCPercent)
		require.LessOrEqual(t, turner.getGCPercent(), defaultGCPercent)
	}

	// 3/4 threshold
	testHeap = make([]byte, threshold/4*3)
	runtime.GC()
	for i := 0; i < 100; i++ {
		runtime.GC()
		require.Equal(t, minGCPercent, turner.getGCPercent())
	}

	// out of threshold
	testHeap = make([]byte, threshold+1024)
	runtime.GC()
	for i := 0; i < 100; i++ {
		runtime.GC()
		require.Equal(t, minGCPercent, turner.getGCPercent())
	}
}

func TestCalcGCPercent(t *testing.T) {
	const gb = 1024 * 1024 * 1024
	// use default value when invalid params
	require.Equal(t, defaultGCPercent, calcGCPercent(0, 0))
	require.Equal(t, defaultGCPercent, calcGCPercent(0, 1))
	require.Equal(t, defaultGCPercent, calcGCPercent(1, 0))

	require.Equal(t, maxGCPercent, calcGCPercent(1, 3*gb))
	require.Equal(t, maxGCPercent, calcGCPercent(gb/2, 4*gb))
	require.Equal(t, uint32(300), calcGCPercent(1*gb, 4*gb))
	require.Equal(t, uint32(166), calcGCPercent(1.5*gb, 4*gb))
	require.Equal(t, uint32(100), calcGCPercent(3*gb, 4*gb))
	require.Equal(t, minGCPercent, calcGCPercent(5*gb, 4*gb))
}
