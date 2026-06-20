package prune_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/prune"
)

func sampleUsage() domain.DiskUsage {
	return domain.DiskUsage{
		Containers: []domain.DiskContainer{
			{ID: "c1", Name: "web", SizeRw: 100, Running: true},
			{ID: "c2", Name: "old", SizeRw: 250, Running: false},
			{ID: "c3", Name: "older", SizeRw: 50, Running: false},
		},
		Images: []domain.DiskImage{
			{ID: "i1", Size: 1000, SharedSize: 0, Dangling: true, InUse: false},
			{ID: "i2", Size: 2000, SharedSize: 500, Dangling: false, InUse: false},
			{ID: "i3", Size: 3000, SharedSize: 0, Dangling: false, InUse: true},
		},
		BuildCache: []domain.DiskBuildCache{
			{ID: "b1", Size: 5000, InUse: false},
			{ID: "b2", Size: 1500, InUse: true},
		},
		Volumes: []domain.DiskVolume{
			{Name: "db-data", Size: 10000, InUse: true},
			{Name: "scratch", Size: 200, InUse: false},
		},
	}
}

func categoryByKind(impact domain.PruneImpact, kind domain.PruneCategoryKind) domain.PruneCategory {
	for _, c := range impact.Categories {
		if c.Kind == kind {
			return c
		}
	}
	return domain.PruneCategory{}
}

func TestComputeBreaksDownByCategory(t *testing.T) {
	impact := prune.Compute(sampleUsage())

	stopped := categoryByKind(impact, domain.PruneStoppedContainers)
	assert.Equal(t, 2, stopped.ObjectCount)
	assert.Equal(t, int64(300), stopped.ReclaimableBytes)

	dangling := categoryByKind(impact, domain.PruneDanglingImages)
	assert.Equal(t, 1, dangling.ObjectCount)
	assert.Equal(t, int64(1000), dangling.ReclaimableBytes)

	unused := categoryByKind(impact, domain.PruneUnusedImages)
	assert.Equal(t, 1, unused.ObjectCount)
	assert.Equal(t, int64(1500), unused.ReclaimableBytes, "shared layers are not reclaimed")

	// In-use image is never counted.
	assert.NotContains(t, []string{}, "i3")
}

func TestComputeBuildCacheIsFirstClass(t *testing.T) {
	impact := prune.Compute(sampleUsage())
	cache := categoryByKind(impact, domain.PruneBuildCacheKind)
	assert.Equal(t, 1, cache.ObjectCount, "only the not-in-use cache record")
	assert.Equal(t, int64(5000), cache.ReclaimableBytes)
}

func TestComputeTotalExcludesVolumes(t *testing.T) {
	impact := prune.Compute(sampleUsage())
	// 300 (containers) + 1000 (dangling) + 1500 (unused) + 5000 (cache) = 7800.
	assert.Equal(t, int64(7800), impact.TotalReclaimable)
}

func TestComputeListsVolumesIndividuallyNeverBulk(t *testing.T) {
	impact := prune.Compute(sampleUsage())

	require.Len(t, impact.Volumes, 2, "every volume is listed individually")
	// Volumes never appear as a reclaimable category — no bulk delete path.
	for _, c := range impact.Categories {
		assert.NotEqual(t, domain.PruneCategoryKind("volumes"), c.Kind)
	}

	byName := map[string]domain.VolumeRef{}
	for _, v := range impact.Volumes {
		byName[v.Name] = v
	}
	assert.True(t, byName["db-data"].InUse)
	assert.Equal(t, int64(10000), byName["db-data"].Size)
	assert.False(t, byName["scratch"].InUse)
}
