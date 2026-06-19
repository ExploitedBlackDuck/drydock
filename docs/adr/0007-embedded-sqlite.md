# ADR-0007 — Embedded SQLite (pure-Go) for host profiles, audit log, and short-horizon resource history

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Connection details for multiple hosts, an audit trail, and a rolling window of
resource samples (to catch a leak or a restart loop) must persist and be
queryable; live container state is ephemeral and stays with the engine.
Cross-platform Wails builds favour a CGO-free toolchain.

## Decision

SQLite via the **pure-Go `modernc.org/sqlite` driver** — a single file in the
platform data dir — with **versioned, embedded, forward-only migrations** behind
a schema-version check. The core depends on a `Store` port; SQL lives only in
the `sqlitestore` adapter. Resource history is a bounded, rolling retention; full
live state is never persisted.

## Consequences

Queryable history without a server; CGO-free keeps cross-compilation trivial.
Cost: pure-Go SQLite has no transparent at-rest encryption, so sensitive values
are sealed at the app layer (ADR-0009). Alternatives (flat files, bbolt) were
rejected for a history/query workload.
