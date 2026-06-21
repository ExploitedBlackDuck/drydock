# ADR-0017 — Published-port exposure is computed and surfaced; loopback vs all-interfaces is first-class

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

The book's spine is that the daemon is root-equivalent and must never be
needlessly exposed (ADR-0005). The same risk re-appears one layer up: a container
that publishes a port to `0.0.0.0`/`::` is reachable on every interface, and on a
public-IP host that is the public internet — usually unintentionally, since
`-p 5432:5432` binds all interfaces by default while `-p 127.0.0.1:5432:5432`
does not. Operators have no single place to see what is reachable from outside,
across a fleet.

This ADR is numbered 0017 (the project book numbers it ADR-0015); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's Phase 2 ADRs are recorded here shifted by two.

## Decision

Drydock computes a per-host, fleet-aggregable **exposure map** from container port
bindings (host IP, host port, container port, protocol) and classifies each
binding by reach: **loopback** (`127.0.0.0/8`/`::1`), **private/LAN**, or
**all-interfaces** (`0.0.0.0`/`::`/empty host IP). All-interfaces bindings on a
host reached over a non-loopback transport are **flagged prominently**. This is
**read-only insight, not enforcement**: Drydock never edits a firewall or rebinds
a port. A **`network_mode: host` container publishes no bindings yet shares the
host's network namespace**, so it is listed explicitly as "host network —
exposure not derivable from port bindings" rather than shown as exposing nothing
(the dangerous false negative).

## Consequences

"What have I exposed?" gets one answer. Cost: reach is classified at the daemon
layer only — Drydock cannot see an upstream cloud security group or host
firewall, so the UI states its scope honestly: it reports the *binding*, not the
full network path. Enforced by classification table tests (loopback / `::1` /
`0.0.0.0` / `::` / empty-host-IP / LAN), a test that an all-interfaces binding on
a non-loopback-transport host is flagged, and a test that a host-network
container is surfaced as "not derivable", not as exposing nothing. No code path
edits a firewall or rebinds a port (pure `internal/core/expose`, review).
