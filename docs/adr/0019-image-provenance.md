# ADR-0019 — Image provenance and staleness in core; vulnerability scanning, if ever added, is an optional adapter behind a port

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

A security-minded operator wants to know a running container is on a stale or
drifted image — the tag has moved since the container started, or the running
digest no longer matches the tag. The Engine exposes the running image's
`RepoDigests`; comparing against the tag's current registry digest answers "is
this stale." A full CVE scan is a different weight class — a large, fast-moving
vulnerability database and network egress — and must not become a core dependency
or quietly violate the no-telemetry posture (ADR-0006).

This ADR is numbered 0019 (the project book numbers it ADR-0017); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's Phase 2 ADRs are recorded here shifted by two.

## Decision

Drydock surfaces **lightweight provenance** in the core: image age, **tag-vs-
digest drift** (running digest ≠ the tag's current registry digest), and
untagged / `:latest` ambiguity — from the Engine's image data plus an **explicit,
operator-initiated** registry digest check (no background phone-home). The
**digest check runs through the daemon, not the desktop** — via the Engine's
distribution-inspect endpoint — so it reflects **the host's** registry
reachability and credentials, not the desktop's: a private registry reachable
only from the host gives a correct answer, and the desktop's own credentials are
never silently substituted. **Vulnerability scanning stays out of core:** if it
is ever added it is a `Scanner` **port** with an optional adapter, invoked
explicitly, results never silently transmitted — and it gets its own ADR then.

## Consequences

A useful staleness signal with no scope creep and no telemetry footprint, correct
even for host-only-reachable registries. Cost: the check is an operator-initiated
call through the host's engine; any auth is the operator's existing Docker
credentials, encoded per-call and **referenced, not copied or stored** (ADR-0009,
the empty-auth call uses the daemon's own credentials); if the host's daemon
cannot reach the registry the result is a clear "registry unreachable from host,"
not a desktop-side guess. Enforced by drift unit tests (running ≠ registry →
`TagDrifted`; unknown digest is never a false drift) and a binding split that
keeps listing provenance network-free while the drift check is the only path that
reaches a registry, through the host's daemon.
