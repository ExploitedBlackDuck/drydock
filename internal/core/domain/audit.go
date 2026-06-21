// Package domain holds Drydock's core types and their validation. It knows
// nothing about the Docker SDK, SSH, SQL, or the OS keyring (PROJECT-BOOK §2.1);
// adapters translate at the edges.
package domain

import "time"

// Action is the typed vocabulary of auditable actions (PROJECT-BOOK §2.7, §7.8).
// Every consequential action is recorded under one of these, never a loose
// string.
type Action string

// The recorded action catalog. Destructive actions carry their impact and the
// acknowledgement that authorized them (ADR-0010/0011).
const (
	ActionHostConnect     Action = "host.connect"
	ActionHostDisconnect  Action = "host.disconnect"
	ActionObserveEnabled  Action = "host.observe.enabled"
	ActionObserveDisabled Action = "host.observe.disabled"

	ActionContainerStart   Action = "container.start"
	ActionContainerStop    Action = "container.stop"
	ActionContainerRestart Action = "container.restart"
	ActionContainerKill    Action = "container.kill"
	ActionContainerRemove  Action = "container.remove"
	ActionContainerExec    Action = "container.exec"

	ActionImagePrune      Action = "image.prune"
	ActionContainerPrune  Action = "container.prune"
	ActionBuildCachePrune Action = "buildcache.prune"
	ActionVolumeRemove    Action = "volume.remove"
	ActionSystemPrune     Action = "system.prune"

	ActionComposeUp   Action = "compose.up"
	ActionComposeDown Action = "compose.down"

	ActionVolumeSnapshot Action = "volume.snapshot"
	ActionVolumeRestore  Action = "volume.restore"
)

// AuditEntry is one record in the append-only, hash-chained audit log
// (PROJECT-BOOK §7.8, ADR-0010). Seq and the hashes are assigned when the entry
// is appended; callers supply the remaining fields.
type AuditEntry struct {
	// Seq is the 1-based monotonic position in the chain.
	Seq int64
	// At is when the action occurred (UTC).
	At time.Time
	// Action is what happened.
	Action Action
	// HostRef identifies the host the action targeted; empty for global actions.
	HostRef string
	// Subject is the object acted on (container id, image ref, volume name, ...).
	Subject string
	// Detail carries action-specific structured context, including, for
	// destructive operations, the computed impact and the acknowledgement.
	Detail map[string]any
	// PrevMAC is the MAC of the preceding entry ("" for the first entry).
	PrevMAC string
	// MAC authenticates this entry: HMAC-SHA256 over PrevMAC and this entry's
	// canonical encoding under the per-install audit key (ADR-0025), or a plain
	// SHA-256 in the degraded key-unavailable mode.
	MAC string
}
