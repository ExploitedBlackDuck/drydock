package domain

import "time"

// TimelineSource distinguishes a timeline row's origin (PROJECT-BOOK §7.12.4).
type TimelineSource string

const (
	// TimelineEngine is a mapped engine event — untrusted input, never part of
	// the audit chain (ADR-0018).
	TimelineEngine TimelineSource = "engine"
	// TimelineAudit is a reference to a Drydock audit entry.
	TimelineAudit TimelineSource = "audit"
)

// TimelineEntry is one row in a host's timeline (§7.12.4, ADR-0018): a mapped
// engine event or a reference to an audit entry, interleaved by time. An engine
// row carries the host's clock; an audit row carries the desktop's — neither is
// silently adjusted, and clock skew is surfaced separately.
type TimelineEntry struct {
	HostRef      string
	At           time.Time
	Source       TimelineSource
	Kind         string
	Subject      string
	ExitCode     *int
	HealthStatus string
	Detail       map[string]any
}
