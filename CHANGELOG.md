# Changelog

All notable changes to Drydock are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Log-stream backpressure (ADR-0021).** A bounded, coalescing buffer
  (`internal/core/stream.LineBuffer`) sits between the engine log reader and the
  UI event bus: a flood is rate-limited by a ticker and, when lines overflow the
  cap, the oldest are dropped and a `⟪ N line(s) dropped — stream fell behind ⟫`
  marker is emitted, so a busy log never grows an unbounded buffer.
- **Exec stdin half-close (ADR-0022).** `ExecStream.CloseStdin` and the
  `CloseExecInput` binding send EOF to a running command without ending the
  session, so a command that reads stdin to completion finishes while its output
  keeps streaming. (Reconnect-driven resync, opaque correlation-id unification,
  and non-interactive stdout/stderr separation remain tracked follow-ups.)

- **Typed option catalog with secret-capture redaction (ADR-0023).** An embedded
  TOML catalog (`internal/core/options/catalogs/docker@1.51.toml`) defines each
  `run`/`create`/`exec` option with a type, risk, and a `secret` flag, and a
  builder validates a selection's types, `requires`, and `conflicts_with`. Secret
  -flagged values (container env, `--env-file`, registry credentials, secret
  build args) are now **redacted on capture** — recorded as `‹redacted›` in the
  persisted operation, the audit detail, and logs, never in cleartext. Exec gains
  an `Env` field whose values are redacted. Closes a v1-blocking gap and unblocks
  the option-rich run/create UI deferred since P5.
- **Consistent database backup (ADR-0024).** A `Back up…` action (and
  `BackupDatabase` binding) writes a single consistent snapshot of the database
  via SQLite `VACUUM INTO` — safe under WAL with concurrent writers, never a raw
  file copy. The destination is never silently overwritten, and the snapshot is a
  plaintext DB outside the sealed store (the operator is told to protect it). A
  database written by a newer build now refuses to open with a dedicated
  `ERR_STORE_SCHEMA_NEWER` error. Closes a v1-blocking gap from the project book.

### Changed

- **Audit chain is now HMAC-keyed and truncation-aware (ADR-0025).** Entries are
  authenticated with `HMAC-SHA256(K, prev_mac || canonical(entry))` under a
  per-install audit key held in the OS keyring (separate from the data key), and
  the latest `(seq, mac)` is persisted as an external high-water mark outside the
  database so tail-truncation is detectable. Verification now reports one of four
  states — intact / in-place-tampered / truncated / key-unavailable — surfaced in
  the Audit view, which states its guarantee honestly (tamper-evident, not
  tamper-proof). The `audit_log` columns are renamed `prev_hash`/`hash` →
  `prev_mac`/`mac` (migration `0002`). Closes a v1-blocking gap from the updated
  project book; fresh installs are keyed from the first entry (a pre-existing
  unkeyed chain reads as tampered until re-based).

## [0.1.0] - 2026-06-20

First tagged release: the complete P0–P9 build plan plus the interactive exec
terminal.

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
- **P9 — packaging, signing & release.**
  - Release workflow (`.github/workflows/release.yml`) triggered by a signed
    `v*` tag: a signed + notarized + stapled macOS zip, a Linux `.deb` (nfpm) and
    AppImage (linuxdeploy), a reproducible-build attestation, and a published
    `SHA256SUMS` over every artifact — all from pinned Go/Node/Wails/nfpm/
    linuxdeploy toolchains (ADR-0012, §9).
  - macOS hardened-runtime entitlements kept minimal (JIT for WKWebView + the
    network client), with `scripts/package-macos.sh` doing codesign → notarytool
    → staple; forks without an Apple identity still build an unsigned artifact.
  - Linux packaging metadata: `nfpm.yaml` (`.deb` with desktop entry + icon +
    dependencies) and a freedesktop `drydock.desktop`.
  - `scripts/reproducible-build.sh` builds the binary twice and compares it
    (normalising away the macOS signature), exposed as `task build:reproducible`;
    `scripts/checksums.sh` emits verifiable `SHA256SUMS`.
  - `docs/RELEASING.md` documents the tag-driven process, the signing secrets,
    download verification, and the pinned toolchain.
- **Interactive exec terminal** (completes the P4 deferral, §7.11.4).
  - The `ExecStream` port gained `Resize`; the Docker adapter resizes the remote
    pseudo-TTY by exec id. App bindings `StartExec`/`SendExecInput`/`ResizeExec`/
    `StopExec` run a guarded, audited session whose output streams base64-encoded
    on `exec:<id>` and whose teardown closes the connection (no leak).
  - GUI: an xterm-based **Shell** tab in the container detail drawer, disabled on
    observe-only hosts and for stopped containers, fitted to the pane and resized
    live. The command is a shell binary (argv), never `sh -c` of input (ADR-0004).
  - Tagged integration test opens a TTY exec, writes, reads the echoed bytes,
    resizes, and closes against a real daemon with no goroutine leak.

[Unreleased]: https://github.com/ExploitedBlackDuck/drydock/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ExploitedBlackDuck/drydock/releases/tag/v0.1.0
