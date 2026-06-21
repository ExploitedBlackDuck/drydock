package domain

import "time"

// OperationKind is the typed catalog of mutating operations (PROJECT-BOOK §2.7).
type OperationKind string

// The mutating operation kinds Drydock performs.
const (
	OpContainerStart   OperationKind = "container.start"
	OpContainerStop    OperationKind = "container.stop"
	OpContainerRestart OperationKind = "container.restart"
	OpContainerKill    OperationKind = "container.kill"
	OpContainerRemove  OperationKind = "container.remove"
	OpContainerExec    OperationKind = "container.exec"
	OpContainerRun     OperationKind = "container.run"

	OpImagePrune      OperationKind = "image.prune"
	OpContainerPrune  OperationKind = "container.prune"
	OpBuildCachePrune OperationKind = "buildcache.prune"
	OpVolumeRemove    OperationKind = "volume.remove"

	OpComposeUp   OperationKind = "compose.up"
	OpComposeDown OperationKind = "compose.down"

	OpVolumeSnapshot OperationKind = "volume.snapshot"
	OpVolumeRestore  OperationKind = "volume.restore"
)

// Destructive reports whether the operation can lose data or in-flight work and
// therefore requires an explicit acknowledgement (ADR-0011).
func (k OperationKind) Destructive() bool {
	switch k {
	case OpContainerRemove, OpContainerKill,
		OpImagePrune, OpContainerPrune, OpBuildCachePrune, OpVolumeRemove,
		OpComposeDown, OpVolumeRestore:
		return true
	default:
		return false
	}
}

// allKinds is the full catalogue, ordered, so DestructiveKinds and any future
// enumeration stay in one place.
var allKinds = []OperationKind{
	OpContainerStart, OpContainerStop, OpContainerRestart, OpContainerKill,
	OpContainerRemove, OpContainerExec,
	OpImagePrune, OpContainerPrune, OpBuildCachePrune, OpVolumeRemove,
	OpComposeUp, OpComposeDown,
}

// DestructiveKinds returns the operation kinds that lose data or in-flight work,
// used to drive the history view's destructive-only filter (PROJECT-BOOK §7.11.8)
// from the same source of truth as Destructive.
func DestructiveKinds() []OperationKind {
	out := make([]OperationKind, 0, len(allKinds))
	for _, k := range allKinds {
		if k.Destructive() {
			out = append(out, k)
		}
	}
	return out
}

// OperationQuery filters the recorded operation history (PROJECT-BOOK §7.11.8).
// Every field is optional; the zero value matches all operations. The store
// composes the SQL — callers never do (PROJECT-BOOK §2.8).
type OperationQuery struct {
	// HostRef restricts to one host; empty matches all hosts.
	HostRef string
	// Kinds restricts to these operation kinds; empty matches all kinds.
	Kinds []OperationKind
	// Since and Until bound StartedAt (Since inclusive, Until exclusive); a zero
	// time leaves that side unbounded.
	Since time.Time
	Until time.Time
	// Limit caps the number of rows returned (most recent first); 0 means the
	// store's default cap.
	Limit int
}

// Operation is a recorded mutating operation and its outcome (PROJECT-BOOK §7.1,
// ADR-0010). It is persisted immutably.
type Operation struct {
	ID             string
	HostRef        string
	Kind           OperationKind
	Target         string
	OptionSet      map[string]any
	Result         string
	BytesReclaimed int64
	StartedAt      time.Time
	EndedAt        time.Time
}

// ResourceSample is one point in a container's rolling resource history
// (PROJECT-BOOK §7.1, §7.6).
type ResourceSample struct {
	HostRef     string
	ContainerID string
	At          time.Time
	CPUPct      float64
	MemBytes    int64
	NetRx       int64
	NetTx       int64
	BlkRead     int64
	BlkWrite    int64
}
