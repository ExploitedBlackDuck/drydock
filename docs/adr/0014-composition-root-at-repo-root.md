# ADR-0014 — Composition root at the repository root

- **Status:** Accepted
- **Date:** 2026-06-19
- **Amends:** PROJECT-BOOK §5 (repo layout), §13 (getting started)

## Context

PROJECT-BOOK §5 sketches the composition root as `cmd/drydock/main.go`. The
operative rule it encodes is "exactly one thin composition root; business logic
there is a defect" — the *path* is incidental.

The Wails v2 toolchain (`wails build`, `wails dev`, `wails generate module`)
expects the `main` package at the **module/repository root**: it scans the
working directory for the main package to compile the app and to introspect bound
structs for binding generation. With `main` under `cmd/drydock/`, `wails generate
module` fails with "no Go files in <root>", and the standard build/dev flow does
not work without fighting the tooling.

`//go:embed` reinforces this: it cannot traverse parent directories, so the
embedded frontend must be referenced from a file at or above its directory. We
keep the embed in `frontend/embed.go` and the bootstrap at the root.

## Decision

The single composition root is **`main.go` at the repository root**. It remains
thin: it resolves configuration, constructs dependencies, injects them, and calls
`app.Run`. No business logic lives there. The `internal/core` boundary, the
`app`/`shell` Wails confinement, and the depguard rules are unchanged.

Where the book references `cmd/drydock/main.go`, read "the root `main.go`".

## Consequences

The Wails toolchain works without modification, including binding generation and
packaging (P9). The book's intent — one thin, lint-checked composition root — is
preserved; only its location changed. This is recorded here rather than edited
silently, per the book's amendment process (§0).
