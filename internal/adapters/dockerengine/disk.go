package dockerengine

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"

	"github.com/drydock/drydock/internal/core/domain"
)

// DiskUsage returns the engine's `system df`, mapped to domain types.
func (c *Client) DiskUsage(ctx context.Context) (domain.DiskUsage, error) {
	du, err := c.cli.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return domain.DiskUsage{}, fmt.Errorf("querying disk usage on host %q: %w", c.hostRef, err)
	}
	return mapDiskUsage(du), nil
}

// mapDiskUsage converts the SDK disk-usage report to domain types. Pure and
// fixture-tested.
func mapDiskUsage(du types.DiskUsage) domain.DiskUsage {
	out := domain.DiskUsage{LayersSize: du.LayersSize}

	for _, img := range du.Images {
		if img == nil {
			continue
		}
		_, _, dangling := splitRepoTag(img.RepoTags)
		out.Images = append(out.Images, domain.DiskImage{
			ID:         img.ID,
			Size:       img.Size,
			SharedSize: img.SharedSize,
			Dangling:   dangling,
			InUse:      img.Containers > 0,
		})
	}

	for _, ctr := range du.Containers {
		if ctr == nil {
			continue
		}
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
		}
		out.Containers = append(out.Containers, domain.DiskContainer{
			ID:      ctr.ID,
			Name:    name,
			SizeRw:  ctr.SizeRw,
			Running: ctr.State == "running",
		})
	}

	for _, vol := range du.Volumes {
		if vol == nil {
			continue
		}
		var size int64
		inUse := false
		if vol.UsageData != nil {
			size = vol.UsageData.Size
			inUse = vol.UsageData.RefCount > 0
		}
		out.Volumes = append(out.Volumes, domain.DiskVolume{Name: vol.Name, Size: size, InUse: inUse})
	}

	for _, bc := range du.BuildCache {
		if bc == nil {
			continue
		}
		out.BuildCache = append(out.BuildCache, domain.DiskBuildCache{
			ID:    bc.ID,
			Size:  bc.Size,
			InUse: bc.InUse,
		})
	}

	return out
}

// PruneContainers removes stopped containers.
func (c *Client) PruneContainers(ctx context.Context) (int64, error) {
	report, err := c.cli.ContainersPrune(ctx, filters.Args{})
	if err != nil {
		return 0, fmt.Errorf("pruning containers on host %q: %w", c.hostRef, err)
	}
	return int64(report.SpaceReclaimed), nil //nolint:gosec // reclaimed bytes fit int64
}

// PruneImages removes dangling images, or all unused images when all is true.
func (c *Client) PruneImages(ctx context.Context, all bool) (int64, error) {
	args := filters.NewArgs()
	if all {
		args.Add("dangling", "false")
	}
	report, err := c.cli.ImagesPrune(ctx, args)
	if err != nil {
		return 0, fmt.Errorf("pruning images on host %q: %w", c.hostRef, err)
	}
	return int64(report.SpaceReclaimed), nil //nolint:gosec // reclaimed bytes fit int64
}

// PruneBuildCache removes unused build cache.
func (c *Client) PruneBuildCache(ctx context.Context) (int64, error) {
	report, err := c.cli.BuildCachePrune(ctx, build.CachePruneOptions{All: false})
	if err != nil {
		return 0, fmt.Errorf("pruning build cache on host %q: %w", c.hostRef, err)
	}
	if report == nil {
		return 0, nil
	}
	return int64(report.SpaceReclaimed), nil //nolint:gosec // reclaimed bytes fit int64
}

// RemoveVolume removes a single named volume.
func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	if err := c.cli.VolumeRemove(ctx, name, force); err != nil {
		return fmt.Errorf("removing volume %q on host %q: %w", name, c.hostRef, err)
	}
	return nil
}
