# ADR-0018 — A host timeline merges the engine event stream with Drydock's audit log, without weakening the audit log

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

Drydock records what *it* changed (ADR-0010), but most of what happens to a host
happens outside Drydock — CLI actions, other tools, and the daemon itself: a
container `die` with an `exitCode`, an `oom`, a `health_status` flip. The Engine
exposes all of this on the event stream, already consumed for live updates and
restart-loop detection (§7.3/§7.6). Reconstructing "why did this die at 03:00,
and who changed it" is currently manual.

This ADR is numbered 0018 (the project book numbers it ADR-0016); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's Phase 2 ADRs are recorded here shifted by two.

## Decision

Drydock persists a bounded, rolling **host timeline** that interleaves mapped
engine events with references to its own audit entries, correlated by host /
container / time. The **audit log stays the separate, append-only, hash-chained
record of Drydock-authored mutations** (ADR-0010): engine events are **untrusted
input** — typed-mapped, fixture-tested, and **never written into the hash chain**
(they go to a separate `timeline_entries` table with no foreign key to hosts and
no link to `audit_log`). Engine rows carry the **host's** timestamp while audit
rows carry the desktop's; the merge sorts by each row's own clock and **surfaces
host-vs-desktop clock skew** rather than silently reordering. The event mapper
**filters to `scope: local`** so the timeline reflects this host, not a swarm
cluster. The timeline reuses the §7.7 rolling-retention sweep and **marks gaps**
when the event stream is interrupted by a reconnect.

## Consequences

A single "what happened on this host" view, including out-of-band change. Cost:
the event stream can gap across a reconnect, so the timeline is best-effort and
**labelled as such**, explicitly distinct from the audit log's completeness
guarantee for Drydock's own actions. Enforced by event-mapper fixture tests
(`die` exit code / `health_status` / scope), a swarm-scope filter test, a
clock-skew test, a retention test, and a test proving engine events never enter
the audit chain (the audit log stays intact under a flood of timeline writes).
