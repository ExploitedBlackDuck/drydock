# Changelog

All notable changes to Drydock are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Volume snapshots (Phase 2, P14, ADR-0020).** The Volumes view offers a
  **Snapshot…** action: it previews the destination, estimated size, and
  duration, then captures the volume to a `tar` file via a throwaway helper
  container that mounts the volume **read-only** (argv, never a shell). It is
  **never automatic and never a precondition for deletion** — an offered
  safeguard. The helper is a **digest-pinned** image (in the catalog), preferred
  if present and otherwise pulled, **failing closed on an air-gapped host**.
  Because it starts a container, snapshot (and restore) are **observe-mode
  blocked** and **audited**; the helper is always removed. The archive is
  plaintext outside the sealed store — the UI says to protect it.
- **Image provenance & staleness (Phase 2, P13, ADR-0019).** The Images view
  gains a provenance badge — **untagged**, **`:latest`** (ambiguous), or, after a
  check, **drifted** / **current** — and a per-image **check** action.
  `provenance.Assess` derives age and `:latest`/untagged status from local image
  data with **no network call** (no background phone-home); the explicit
  `CheckImageDrift` resolves the tag's current registry digest **through the
  host's daemon** (distribution-inspect, the host's credentials, referenced not
  copied) and flags **tag-vs-digest drift**. Vulnerability scanning stays out of
  core. `domain.Image` gains the pulled `RepoDigest`.
- **Host timeline (Phase 2, P12, ADR-0018).** A new per-host **Timeline** view
  interleaves mapped engine events (`die` with its exit code, `oom`,
  `health_status`) with references to Drydock's audit log — "what happened on
  this host, and who." The hash-chained **audit log is never weakened**: engine
  events are untrusted input persisted to a separate `timeline_entries` table
  (rolling retention), never into the chain (proven by a separateness test).
  Swarm-scope events are filtered to keep the view host-local; host-vs-desktop
  **clock skew** and **reconnect gaps** are surfaced, not hidden. The event
  mapper now captures exit code, health status, and scope.
- **Exposure map (Phase 2, P11, ADR-0017).** A new per-host **Exposure** view
  answers "what have I exposed?": `internal/core/expose.Compute` classifies each
  published port by reach — **loopback** (`127.0.0.0/8`/`::1`), **private/LAN**,
  or **all-interfaces** (`0.0.0.0`/`::`/empty host IP) — and **flags**
  all-interfaces bindings on a host reached over a non-loopback transport
  (plausibly internet-reachable). A `network_mode: host` container is listed
  explicitly as "exposure not derivable from port bindings", never as exposing
  nothing. Read-only insight — Drydock never edits a firewall or rebinds a port —
  and the UI states its daemon-layer scope. Pure and table-tested.
- **Compose plan — `up` is preview-and-confirm (Phase 2, P10, ADR-0016).** `up`
  stops being a black box: Drydock previews what it would do before applying.
  - `internal/core/compose.Plan` classifies each service create/recreate/start/
    noop and each network/volume create by diffing the desired project against
    observed state. Recreations that interrupt a running container or drop an
    anonymous volume mark the plan **destructive** (→ §7.4 acknowledgement); an
    uncomparable config hash **degrades** to a labelled coarse image diff; and an
    inaccessible source yields an explicit **source-unavailable** plan, never a
    false "no changes." Pure and table-tested.
  - `composeparse` parses the project's compose files (pinned compose-go, local
    files only) into the desired stack; `ComputeComposePlan` builds the observed
    state from container labels and returns the plan.
  - `operations.ComposeApply` requires acknowledgement for a destructive plan and
    records its impact + ack to the audit log (tested).
  - GUI: selecting **Up** opens a `ComposePlanPanel` showing the per-service and
    per-resource changes, with destructive elements gated and degraded/source-
    unavailable plans clearly labelled — the operator confirms the plan, not raw
    `up`.

- **Log-stream backpressure (ADR-0021).** A bounded, coalescing buffer
  (`internal/core/stream.LineBuffer`) sits between the engine log reader and the
  UI event bus: a flood is rate-limited by a ticker and, when lines overflow the
  cap, the oldest are dropped and a `⟪ N line(s) dropped — stream fell behind ⟫`
  marker is emitted, so a busy log never grows an unbounded buffer.
- **Reconnect-driven resync (ADR-0021).** A `registry.Reconnect` primitive
  re-establishes a dropped engine connection, and the app's event supervisor
  now, on a dropped stream, emits `resync:<hostID>` and retries with bounded
  backoff; the frontend `ResyncWatcher` refetches host status and object lists
  on the signal (showing a brief "Resyncing live state…" indicator) instead of
  resuming stale data.
- **Exec stdin half-close (ADR-0022).** `ExecStream.CloseStdin` and the
  `CloseExecInput` binding send EOF to a running command without ending the
  session, so a command that reads stdin to completion finishes while its output
  keeps streaming. (The opaque correlation-id registry unification and
  non-interactive stdout/stderr separation remain tracked follow-ups.)

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
