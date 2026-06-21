# ADR-0024 — SQLite durability: WAL mode, a real backup path, and downgrade refusal

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

§7.7 described backup as a "file-copy of the data dir while quiesced." Under
SQLite **WAL mode** (the right default for a responsive app doing concurrent
reads), a bare copy of the main DB file omits uncheckpointed WAL/SHM and can be
inconsistent; the journal mode was unstated. Forward-only migrations also left
"an older app opens a newer DB" undefined.

This ADR is numbered 0024 (the project book numbers it ADR-0022); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's gap-review ADRs are recorded here shifted by two.

## Decision

The database runs in **WAL mode** with `synchronous=NORMAL` (already configured
at Open via the DSN pragmas). **Backup uses `VACUUM INTO`** — a bound-parameter
destination, never interpolated SQL — to produce a single consistent file that
is safe under WAL with concurrent readers and writers; a raw file copy is never
used and the docs no longer instruct one. The destination must not already
exist, so a backup never silently clobbers a previous one.

On open, the app checks `schema_migrations`: a DB **newer than the app refuses
to open** with the dedicated `ErrSchemaNewer` sentinel (carrying the
`ERR_STORE_SCHEMA_NEWER` code, and also wrapping `ErrMigration`) rather than risk
corrupting it; an older DB migrates forward as before.

## Consequences

Backups are consistent and downgrades fail safe. Cost: a backup routine using
`VACUUM INTO` and a one-line version-guard on open. The snapshot is a plaintext
database **outside** Drydock's sealed store; the operator is told to protect the
destination. Enforced by a test that a concurrent-writer backup opens as a valid,
queryable database at a positive schema version, that a second backup to the same
path is refused, and that a newer-than-app DB is rejected with `ErrSchemaNewer`.
