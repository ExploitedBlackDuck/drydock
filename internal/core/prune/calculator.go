// Package prune computes the preview shown before any destructive cleanup
// (PROJECT-BOOK §7.4, ADR-0011): the exact reclaimable space, broken out by
// category, with build cache first-class and volumes listed individually so they
// are never bulk-deleted.
package prune

import "github.com/drydock/drydock/internal/core/domain"

// Compute derives a PruneImpact from a DiskUsage snapshot. It is pure (no I/O)
// and table-tested against captured `system df` fixtures.
//
// Categories report what a prune would reclaim: stopped containers, dangling
// images (removed by a default prune), unused tagged images (removed only by
// `prune -a`), and build cache. The total sums the categories; volumes are NOT
// included in the total — each is confirmed on its own.
func Compute(du domain.DiskUsage) domain.PruneImpact {
	impact := domain.PruneImpact{}

	var stoppedCount int
	var stoppedBytes int64
	for _, c := range du.Containers {
		if !c.Running {
			stoppedCount++
			stoppedBytes += c.SizeRw
		}
	}
	addCategory(&impact, domain.PruneStoppedContainers, "Stopped containers", stoppedCount, stoppedBytes)

	var danglingCount, unusedCount int
	var danglingBytes, unusedBytes int64
	for _, img := range du.Images {
		if img.InUse {
			continue
		}
		if img.Dangling {
			danglingCount++
			danglingBytes += img.Size
		} else {
			unusedCount++
			// Shared layers are not reclaimed, so only the unique size frees up.
			unusedBytes += img.Size - img.SharedSize
		}
	}
	addCategory(&impact, domain.PruneDanglingImages, "Dangling images", danglingCount, danglingBytes)
	addCategory(&impact, domain.PruneUnusedImages, "Unused images", unusedCount, unusedBytes)

	var cacheCount int
	var cacheBytes int64
	for _, bc := range du.BuildCache {
		if !bc.InUse {
			cacheCount++
			cacheBytes += bc.Size
		}
	}
	addCategory(&impact, domain.PruneBuildCacheKind, "Build cache", cacheCount, cacheBytes)

	// Volumes are listed individually (never bulk): each carries its size and
	// in-use status for per-volume confirmation (PROJECT-BOOK §7.4).
	for _, v := range du.Volumes {
		impact.Volumes = append(impact.Volumes, domain.VolumeRef(v))
	}

	return impact
}

func addCategory(impact *domain.PruneImpact, kind domain.PruneCategoryKind, label string, count int, bytes int64) {
	impact.Categories = append(impact.Categories, domain.PruneCategory{
		Kind:             kind,
		Label:            label,
		ObjectCount:      count,
		ReclaimableBytes: bytes,
	})
	impact.TotalReclaimable += bytes
}
