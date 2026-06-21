package domain

import "time"

// VolumeSnapshot records an explicit, operator-initiated capture of a volume's
// contents (PROJECT-BOOK §7.12.6, ADR-0020): a streamed tar to an operator-chosen
// destination, taken read-only via a throwaway helper container. It is never
// automatic and never a precondition for deletion — an offered safeguard. The
// archive is plaintext outside Drydock's sealed store; the operator protects it.
type VolumeSnapshot struct {
	HostRef     string
	Volume      string
	Destination string
	SizeBytes   int64
	CreatedAt   time.Time
}

// VolumeSnapshotPreview is shown before a snapshot runs (ADR-0020): the
// destination, an estimated size, and the expected duration, so the cost is
// stated up front. The operation is cancellable.
type VolumeSnapshotPreview struct {
	Volume          string
	Destination     string
	EstimatedBytes  int64
	EstimatedSecond int64
}
