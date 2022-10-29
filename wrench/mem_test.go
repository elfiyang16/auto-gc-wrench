package wrench

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMem(t *testing.T) {
	const mb = 1024 * 1024

	heap := make([]byte, 100*mb+1)
	inuse := readMemoryInuse()
	t.Logf("mem inuse: %d MB", inuse/mb)
	require.GreaterOrEqual(t, inuse, uint64(100*mb))
	heap[0] = 0 // unset
}
