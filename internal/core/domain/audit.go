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

	ActionImagePrune  Action = "image.prune"
	ActionVolumeprune Action = "volume.prune"
	ActionSystemPrune Action = "system.prune"

	ActionComposeUp   Action = "compose.up"
	ActionComposeDown Action = "compose.down"
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
	// PrevHash is the hash of the preceding entry ("" for the first entry).
	PrevHash string
	// Hash is SHA-256 over PrevHash and this entry's canonical encoding.
	Hash string
}
