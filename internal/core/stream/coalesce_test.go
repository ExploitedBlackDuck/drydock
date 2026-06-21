package stream_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/stream"
)

func TestLineBufferBoundsAndReportsDrops(t *testing.T) {
	buf := stream.NewLineBuffer(100)
	for i := 0; i < 150; i++ {
		buf.Push(fmt.Sprintf("line-%d", i))
	}

	lines, dropped := buf.Drain()
	assert.Len(t, lines, 100, "memory is bounded to the capacity")
	assert.Equal(t, 50, dropped, "the overflow is counted, not silently lost")
	// The kept lines are the most recent ones.
	assert.Equal(t, "line-50", lines[0])
	assert.Equal(t, "line-149", lines[len(lines)-1])
}

func TestLineBufferDrainResets(t *testing.T) {
	buf := stream.NewLineBuffer(10)
	buf.Push("a")
	buf.Push("b")

	lines, dropped := buf.Drain()
	require.Equal(t, []string{"a", "b"}, lines)
	assert.Zero(t, dropped)

	// A second drain with nothing pushed is empty.
	lines, dropped = buf.Drain()
	assert.Empty(t, lines)
	assert.Zero(t, dropped)
}

func TestLineBufferNoDropsWithinCapacity(t *testing.T) {
	buf := stream.NewLineBuffer(5)
	for i := 0; i < 5; i++ {
		buf.Push("x")
	}
	lines, dropped := buf.Drain()
	assert.Len(t, lines, 5)
	assert.Zero(t, dropped, "exactly-full does not drop")
}
