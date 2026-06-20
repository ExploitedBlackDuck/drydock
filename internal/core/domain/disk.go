package domain

// DiskUsage is the engine's `system df`, mapped to domain types (PROJECT-BOOK
// §7.6). It is the input to the prune-impact calculator.
type DiskUsage struct {
	Images     []DiskImage
	Containers []DiskContainer
	Volumes    []DiskVolume
	BuildCache []DiskBuildCache
	LayersSize int64
}

// DiskImage is an image's disk footprint.
type DiskImage struct {
	ID         string
	Size       int64
	SharedSize int64
	Dangling   bool
	InUse      bool
}

// DiskContainer is a container's writable-layer footprint.
type DiskContainer struct {
	ID      string
	Name    string
	SizeRw  int64
	Running bool
}

// DiskVolume is a volume's footprint.
type DiskVolume struct {
	Name  string
	Size  int64
	InUse bool
}

// DiskBuildCache is one build-cache record.
type DiskBuildCache struct {
	ID    string
	Size  int64
	InUse bool
}

// PruneCategoryKind identifies a reclaimable category (PROJECT-BOOK §7.4).
type PruneCategoryKind string

const (
	// PruneStoppedContainers are stopped containers removable by prune.
	PruneStoppedContainers PruneCategoryKind = "stopped-containers"
	// PruneDanglingImages are untagged images removable by a default prune.
	PruneDanglingImages PruneCategoryKind = "dangling-images"
	// PruneUnusedImages are tagged-but-unreferenced images removable by `prune -a`.
	PruneUnusedImages PruneCategoryKind = "unused-images"
	// PruneBuildCacheKind is the build cache — frequently the largest, most
	// overlooked consumer (PROJECT-BOOK §7.6).
	PruneBuildCacheKind PruneCategoryKind = "build-cache"
)

// PruneCategory is one reclaimable bucket in a prune impact.
type PruneCategory struct {
	Kind             PruneCategoryKind
	Label            string
	ObjectCount      int
	ReclaimableBytes int64
}

// PruneImpact is the preview computed before any destructive cleanup
// (PROJECT-BOOK §7.4, ADR-0011). Volumes are deliberately a separate list, never
// a bulk category: each named volume must be confirmed individually.
type PruneImpact struct {
	Categories       []PruneCategory
	Volumes          []VolumeRef
	TotalReclaimable int64
}

// VolumeRef is a single volume candidate for individual confirmation.
type VolumeRef struct {
	Name  string
	Size  int64
	InUse bool
}
