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

[Unreleased]: https://github.com/drydock/drydock/commits/main
