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

[Unreleased]: https://github.com/drydock/drydock/commits/main
