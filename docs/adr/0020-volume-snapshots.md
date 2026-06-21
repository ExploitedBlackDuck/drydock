# ADR-0020 — A volume may be snapshotted before destruction; snapshots are explicit, never automatic, and never a precondition

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

The book already makes volume deletion preview-and-confirm and never bulk
(ADR-0011, §7.4). The natural completion of that property is an escape hatch: let
the operator capture a volume's contents *before* removing it, so a wrong confirm
is recoverable. But silent automatic backups of unknown-size volumes would be
their own surprise — blown disk, stalled operations.

This ADR is numbered 0020 (the project book numbers it ADR-0018); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's Phase 2 ADRs are recorded here shifted by two.

## Decision

Drydock offers an **explicit, operator-initiated volume snapshot**: a streamed
`tar` of the volume's contents to an operator-chosen destination, taken via a
throwaway helper container mounting the volume **read-only** (Engine API, argv —
ADR-0004), with destination, size estimate, and expected duration shown first. It
is **never automatic** and **never blocks deletion**; it is an *offered*
safeguard. The helper is a **digest-pinned minimal image** (recorded in the
versioned, embedded catalog, §7.5); Drydock prefers an image already present on
the host and otherwise pulls the pinned digest as an audited, visible step,
**failing closed with a clear error on an air-gapped host**. Because a snapshot
**starts a container**, it is a mutating side effect and is therefore **blocked
on observe-mode hosts like every other mutation** (ADR-0013). Snapshot and restore
are audited operations (ADR-0010); the helper container is always removed, even
on cancellation.

## Consequences

The data-loss footgun gains an undo. Cost: snapshotting a large volume is slow
and consumes disk (size/time stated up front; the operation is cancellable); a
snapshot is a plaintext archive **outside** Drydock's sealed store, so the UI
states plainly that the operator must protect the destination; restore is a
separate, deliberately-confirmed action; the pinned helper image is a
supply-chain input verified by digest and re-pinned through the normal
catalog-versioning process. Enforced by tests: snapshot/restore pass the
observe-mode guard (blocked on observe), both write audit entries, snapshot is
never a precondition for delete (`RemoveVolume` needs no snapshot), and the
helper runs argv with a read-only mount (review/`gosec`); the air-gapped
fail-closed path is exercised by the helper-image check. **Operational note:** the
shipped helper digest must be verified/refreshed before a public release.
