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
)

// EngineEvent is a mapped Docker event (PROJECT-BOOK §7.3). It drives live UI
// updates and restart-loop detection (§7.6).
type EngineEvent struct {
	Type          string
	Action        string
	ContainerID   string
	ContainerName string
	At            time.Time
}
