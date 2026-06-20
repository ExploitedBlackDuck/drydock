package dockerengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapDiskUsageFromFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "api", "system_df.json"))
	require.NoError(t, err)
	var du types.DiskUsage
	require.NoError(t, json.Unmarshal(data, &du))

	mapped := mapDiskUsage(du)

	assert.Equal(t, int64(500000000), mapped.LayersSize)

	require.Len(t, mapped.Images, 3)
	assert.False(t, mapped.Images[0].Dangling)
	assert.True(t, mapped.Images[0].InUse, "image with a container is in use")
	assert.True(t, mapped.Images[1].Dangling, "<none>:<none> is dangling")
	assert.False(t, mapped.Images[2].InUse, "tagged, unreferenced image is unused")
	assert.Equal(t, int64(20000000), mapped.Images[2].SharedSize)

	require.Len(t, mapped.Containers, 2)
	assert.True(t, mapped.Containers[0].Running)
	assert.False(t, mapped.Containers[1].Running)

	require.Len(t, mapped.Volumes, 2)
	assert.True(t, mapped.Volumes[0].InUse)
	assert.Equal(t, int64(104857600), mapped.Volumes[0].Size)
	assert.False(t, mapped.Volumes[1].InUse)

	require.Len(t, mapped.BuildCache, 2)
	assert.False(t, mapped.BuildCache[0].InUse)
	assert.Equal(t, int64(300000000), mapped.BuildCache[0].Size)
}
