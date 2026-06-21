# ADR-0023 — Captured parameters are redacted/sealed: the tool never persists in cleartext the secrets it handles

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

Operation capture and the audit log persist each operation's **resolved
parameters** (ADR-0010, §7.7/§7.8), and the option builder governs
`run`/`create`/`exec` options including `-e` / `--env` / `--env-file` and registry
auth. A container env secret is therefore a "resolved parameter" that would
otherwise be written to `operations.option_set` JSON and the audit `detail` **in
cleartext**. ADR-0009's sealing covered connection secrets (SSH/TLS keys,
passphrases, the data key) but not this capture path — a latent way for the tool
to leak the very secrets it was used to set.

This ADR is numbered 0023 (the project book numbers it ADR-0021); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's gap-review ADRs are recorded here shifted by two.

## Decision

The option catalog (§7.5) is now a real, embedded, typed artifact
(`internal/core/options/catalogs/docker@<apiversion>.toml`) loaded and validated
by a builder; each option carries a **`secret` flag** (env values, `--env-file`
contents, registry credentials, secret build args). On capture, secret-flagged
values are **recorded as present but redacted** (stable key, value `‹redacted›`)
in `operations.option_set` and the audit `detail` by routing every captured
option set through `Catalog.Redact` before it reaches the store, the audit log,
or slog. The operations service takes a `Redactor` (the catalog) and redacts in
its single record path, so no recording site can bypass it. The builder
validates types, `requires`, and `conflicts_with` and assembles API parameters —
never a shell (ADR-0004).

## Consequences

Drydock cannot, through its own history or audit, leak a secret it was used to
set. Cost: a `secret` classification per catalog option and a redaction step on
the capture path. Enforced by a boundary test: an exec carrying a known env
secret records `env = ‹redacted›` in both the persisted operation and the audit
detail, and the secret string appears in neither — mirroring the §2.4 logs
redaction test. The catalog and builder also unblock the option-rich
`run`/`create` UI deferred since P5.
