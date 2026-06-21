package auditmark

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
)

func TestFileMarkRoundTrip(t *testing.T) {
	ctx := context.Background()
	mark := NewFile(filepath.Join(t.TempDir(), "audit.hwm"))

	// Absent before any write.
	_, ok, err := mark.Get(ctx)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, mark.Set(ctx, audit.HighWaterMark{Seq: 7, MAC: "abc123"}))

	got, ok, err := mark.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(7), got.Seq)
	assert.Equal(t, "abc123", got.MAC)

	// A later write replaces it.
	require.NoError(t, mark.Set(ctx, audit.HighWaterMark{Seq: 8, MAC: "def456"}))
	got, _, err = mark.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(8), got.Seq)
	assert.Equal(t, "def456", got.MAC)
}
