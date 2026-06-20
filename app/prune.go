package app

import (
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/prune"
)

// GetPruneImpact computes the prune preview for a host: reclaimable space by
// category (build cache first-class) and the per-volume candidate list. This is
// a read, so it works even on observe-only hosts (PROJECT-BOOK §7.4).
func (a *App) GetPruneImpact(hostID string) (domain.PruneImpact, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return domain.PruneImpact{}, err
	}
	du, err := eng.DiskUsage(ctx)
	if err != nil {
		return domain.PruneImpact{}, err
	}
	return prune.Compute(du), nil
}

// PruneImages removes dangling images (or all unused when all is true). Requires
// acknowledgement; records the confirmed impact and bytes reclaimed.
func (a *App) PruneImages(hostID string, all, ack bool) (int64, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.PruneImages(ctx, hostID, all, ack)
}

// PruneContainers removes stopped containers (requires acknowledgement).
func (a *App) PruneContainers(hostID string, ack bool) (int64, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.PruneContainers(ctx, hostID, ack)
}

// PruneBuildCache removes unused build cache (requires acknowledgement).
func (a *App) PruneBuildCache(hostID string, ack bool) (int64, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.PruneBuildCache(ctx, hostID, ack)
}

// RemoveVolume removes a single named volume (per-volume; requires
// acknowledgement). There is no bulk volume-prune binding by design (§7.4).
func (a *App) RemoveVolume(hostID, name string, ack bool) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.RemoveVolume(ctx, hostID, name, ack)
}
