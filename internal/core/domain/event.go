package domain

import "time"

// Engine event types and actions Drydock reacts to (a small, typed subset of the
// Docker event stream).
const (
	EventTypeContainer = "container"

	EventActionStart   = "start"
	EventActionDie     = "die"
	EventActionStop    = "stop"
	EventActionKill    = "kill"
	EventActionRestart = "restart"
	EventActionDestroy = "destroy"
	EventActionHealth  = "health_status"
	EventActionOOM     = "oom"
)

// EngineEvent is a mapped Docker event (PROJECT-BOOK §7.3). It drives live UI
// updates, restart-loop detection (§7.6), and the host timeline (§7.12.4). It is
// untrusted input: typed-mapped, never written into the hash-chained audit log
// (ADR-0018).
type EngineEvent struct {
	Type          string
	Action        string
	ContainerID   string
	ContainerName string
	// ExitCode is the container's exit code on a "die" event, nil otherwise.
	ExitCode *int
	// HealthStatus is the new health on a "health_status" event (e.g. "healthy",
	// "unhealthy"), empty otherwise.
	HealthStatus string
	// Scope is the event's scope ("local" or "swarm"); the timeline keeps only
	// local-scope events so it reflects this host, not the cluster (ADR-0018).
	Scope string
	At    time.Time
}
