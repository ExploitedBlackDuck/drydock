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
)

// Destructive reports whether the operation can lose data or in-flight work and
// therefore requires an explicit acknowledgement (ADR-0011).
func (k OperationKind) Destructive() bool {
	switch k {
	case OpContainerRemove, OpContainerKill:
		return true
	default:
		return false
	}
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
