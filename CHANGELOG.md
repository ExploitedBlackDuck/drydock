# Changelog

All notable changes to Drydock are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **P0 — skeleton & charter scaffolding.** Repository layout, composition root,
  and an application that opens a window and logs startup.
  - XDG-respecting configuration with a typed TOML loader
    (`internal/platform/config`), creating the data directory with `0700`
    permissions.
  - Operational logging via `log/slog` with secret redaction at the logging
    boundary (`internal/platform/logging`).
  - Wails v2 binding layer (`app/`) and runtime isolation behind a `shell`
    interface (ADR-0001), wired through a thin composition root.
  - Svelte + TypeScript + Vite frontend talking to generated, typed bindings.
  - Application shell (§7.11): always-visible host switcher with
    transport/trust/observe state, eight-view navigation, add-host wizard, a dark
    design-token system, accessible status cues, and per-view
    loading/error/degraded/empty states backed by concern-split stores.
  - Tooling gates: `gofumpt`, curated `golangci-lint`, `go vet`, `govulncheck`,
    strict `tsc`/`svelte-check`, ESLint, and Prettier, orchestrated by a
    `Taskfile` and GitHub Actions CI.
  - Architecture Decision Records ADR-0001 … ADR-0014 and governance files
    (`LICENSE`, `NOTICE`, `SECURITY.md`, `CONTRIBUTING.md`).
- **P1 — persistence foundation.**
  - Pure-Go SQLite store (`sqlitestore`) with embedded forward-only migrations,
    a schema-version check that refuses a newer-than-supported database, and the
    §7.7 schema.
  - Append-only, hash-chained audit log (`audit`) with chain verification and
    tamper detection.
  - Secret sealing (`secret`): a `SecretStore` port + XChaCha20-Poly1305 sealing
    under a per-install data key, backed by the OS keyring adapter.
  - Store opened/migrated and audit chain verified at startup.
- **P2 — local engine (read-only).**
  - `Engine` port and a Docker Go SDK adapter (`dockerengine`) with per-host API
    version negotiation and list of containers/images/volumes/networks; pure
    SDK→domain mappers table-tested against sanitized captured fixtures.
  - Tagged integration test lists against a real daemon and asserts no leaked
    connections.
  - The local engine is surfaced in the GUI: it appears in the host switcher
    when reachable, and the Containers/Images/Volumes/Networks views render live
    data with loading/error/empty states.
  - Reviewed `govulncheck` allowlist (ADR-0015) for unreachable daemon-side Moby
    advisories with no fixed release.
- **P3 — SSH transport, multi-host, observe-mode.**
  - `sshdialer` dials the remote engine socket over SSH (agent + referenced keys,
    known_hosts verification, supervised keepalive, no leaked tunnel on Close).
  - `hosts.Registry`: persisted multi-host profiles, supervised connect/
    disconnect lifecycle, and the observe-mode guard (`Mutate`) that rejects
    observe-only hosts in the core before any request reaches the engine.
  - Unauthenticated-TCP detection flags such hosts untrusted (core + wizard).
  - GUI: backend-driven host switcher, a working add-host wizard (persists +
    connects, warns on insecure transport), multi-host switching, and an
    explicit, audited observe-mode toggle.
- **P4 — container control, logs, stats, exec.**
  - Engine gains start/stop/restart/kill/remove, log streaming, stats sampling,
    and argv exec (never a shell). The `operations` service runs every mutation
    through the observe-mode guard, records it, and audits it; destructive ops
    require acknowledgement.
  - Streamed stats are sampled into the rolling resource history.
  - GUI: contextual container actions (observe-aware, confirmed destruction) and
    a detail drawer with streamed logs and live CPU/memory/network stats.
  - Cancellation propagates to streams with no leaked goroutine (tagged
    integration test against a real container).
- **P5 — prune impact preview & destructive-op safety.**
  - `prune.Compute` derives a per-category reclaimable preview (stopped
    containers, dangling vs unused images, build cache first-class) from
    `system df`; volumes are listed individually, never as a bulk category.
  - `options.Assess`: declarative, table-tested impact rules (observe-mode →
    block, `rm -f` running → require_ack, `down -v` → require_ack, exec root →
    warn).
  - Prune flows require acknowledgement and write the confirmed impact + bytes
    reclaimed to the operation record, `prune_impacts`, and the audit log.
  - GUI: a Disk dashboard with per-category reclaimable space (build cache
    first-class), confirm-before-prune, and per-volume removal (never bulk),
    disabled on observe-only hosts.
- **P6 — disk & resource dashboard + restart-loop detection.**
  - Engine event stream (`dockerengine`, fixture-tested) feeding a
    `restartloop.Detector` that flags crash-looping containers within a sliding
    window (tested against an event fixture).
  - `history.Retention` sweeps resource samples beyond the configured window,
    run as an owned background goroutine.
  - GUI: a restart-loop callout for the active host and a CPU resource-history
    sparkline in the container detail (seeded from persisted history, extended
    live).
- **P7 — Compose stacks.**
  - `compose.Group` partitions a host's containers into stacks by the
    `com.docker.compose.project`/`.service` labels, computing per-service and
    aggregate running/total counts and a running/partial/stopped state (pure,
    unit-tested against fixtures).
  - Engine port gained `ComposeUp`/`ComposeDown(volumes)`: the stack is acted on
    as a unit over the SDK by label selection — no compose-binary shell-out — so
    it behaves identically for local and SSH-tunnelled hosts (ADR-0003/0004).
  - `operations.Service` runs compose up/down through the observe guard and
    records/audits them; `down` is destructive (removes the stack's containers)
    and `down -v` additionally removes the project's named volumes, both gated by
    acknowledgement (the impact-rule engine already flags `COMPOSE_DOWN_VOLUMES`).
  - GUI: a Compose view grouping stacks as cards with per-service status, a
    stack-state pill, up/down/`down -v` controls (observe-aware, confirmed), and
    per-container logs/stats via the shared detail drawer.
  - Tagged integration test synthesizes a labelled two-service stack, confirms
    discovery/grouping, takes it down as a unit, and asserts the containers are
    removed.
- **P8 — operation history & audit view + export.**
  - `Store.Operations` query with a `domain.OperationQuery` filter (by host,
    kind set, time window, and limit) composed entirely in the store from bound
    parameters; `domain.DestructiveKinds` drives the destructive-only filter from
    the same source of truth as `OperationKind.Destructive`.
  - `internal/core/journal` read model: history queries, an audit-trail view that
    verifies the hash chain in memory (via the extracted pure
    `audit.VerifyEntries`) for a green/red indicator, and a portable JSON export
    of every operation plus the full chain that re-verifies on its own.
  - App bindings `OperationHistory`, `AuditTrail`, and `ExportJournal`; the export
    downloads from the webview with no backend file APIs.
  - GUI: a History view (kind + destructive-only filters, per-operation result
    and reclaimed bytes) and an Audit view (chain-verification chip, entry table,
    tamper detail), both with one-click export.
  - Gates met: store query tests (host/kind/time/limit/round-trip), audit
    tamper-detection and export round-trip tests, and an independent
    exported-chain verification.

[Unreleased]: https://github.com/drydock/drydock/commits/main
