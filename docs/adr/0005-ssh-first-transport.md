# ADR-0005 — Remote access is SSH-first; the daemon API is treated as root-equivalent; the tool never exposes or encourages exposing the socket

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

The Docker daemon runs as root, so access to its API (the
`/var/run/docker.sock` Unix socket) is equivalent to unrestricted root on the
host. Exposing an unauthenticated TCP socket (`tcp://0.0.0.0:2375`) is an
internet-scannable path to a root shell; even TLS-over-TCP grants root to any
holder of the client certificate. SSH transport reuses existing keys and the
host's own access controls.

## Decision

Drydock reaches remote engines by **dialing the remote Unix socket over SSH**
(reusing the operator's keys/agent), as the default and recommended path.
**mTLS-over-TCP** is supported only for hosts already configured that way, with
the client key treated as a root-equivalent secret (ADR-0009). Drydock **never
opens, configures, or suggests opening an unauthenticated TCP socket**, and
surfaces a warning if it detects it is talking to one. A connected host is shown
with its transport and trust level so the operator always knows how
root-equivalent access is travelling.

## Consequences

Safe-by-default remote management with no new attack surface on the host. Cost:
SSH connection/tunnel management must be robust (the precise area where existing
TUIs are unreliable) — it gets dedicated supervision and tests.
