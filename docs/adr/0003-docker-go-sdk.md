# ADR-0003 — Talk to the Engine API via the official Go SDK; do not shell out to the `docker` CLI

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Docker exposes a complete HTTP Engine API; the first-party Go SDK
(`github.com/docker/docker/client`) negotiates API versions and gives typed
access to containers, images, volumes, networks, `system df`, logs, stats, and
the event stream. Shelling out to the CLI means parsing human-formatted text,
losing streaming, and adding a binary dependency.

## Decision

Use the Docker Go SDK behind an `Engine` port. The same client speaks to any
API-compatible engine reachable through the `Dialer` (including a Podman
socket). Compose actions use the Compose-over-API path where available, falling
back to documented project semantics — never string-built shell.

## Consequences

Typed, streaming, version-negotiated access; no text scraping. Cost: we track
the SDK and a minimum supported API version (ADR-0008). The `Engine` adapter is
the main thing to test against captured fixtures.
