# Contributing to Drydock

`PROJECT-BOOK.md` (referred to below as the book) is the specification and these
standards are binding. Work in the phases defined in the book's §7.10, in order;
a phase is not started until the previous phase's gate is green.

## Before writing code

- Read the book's §2 (Engineering Charter) and §7 (specification).
- Two properties have NO bypass: remote access is SSH-first and we never expose
  the daemon socket (§7.2 / ADR-0005); destruction is preview-and-confirm and
  volumes are never bulk-deleted (§7.4 / ADR-0011). Do not weaken either.

## Every change

- `context.Context` is the first parameter of any I/O / connection / API call.
- Errors wrapped with `%w` + context; typed sentinels for branched-on errors
  (e.g. `ErrObserveMode`); error codes from the §8.4 catalog; no string matching
  on error text; no `panic` outside `main`/tests.
- Structured logging via `slog`; secrets are never logged (use the redaction
  helpers). The operational log is not the audit log.
- No global mutable state; dependencies are constructed and injected in the
  composition root (`main.go`, ADR-0014).
- Engine access via the Docker SDK behind the `Engine` port; container exec and
  any subprocess use argv. Never a shell.
- SSH private keys are referenced, never copied into our store; secrets live in
  the keyring; sensitive saved values are AEAD-sealed before disk.
- Mutations are rejected in the core for observe-mode hosts.
- No `wails` import outside `app/` and `shell/` (enforced by depguard).
- No authorship/tooling fingerprints in any file, commit, or metadata.

## Definition of a finished phase

Run and record the output of:

```sh
task lint               # gofumpt + golangci-lint + go vet + govulncheck + frontend
task test               # go test green on a bare machine (no daemon)
task test:integration   # tagged; runs against a real daemon/keyring if present
task build              # runnable app for the host platform
```

A phase is not done until its gate (book §7.10) passes. "It runs" is not the
gate. Install the developer tooling once with `task tools`.

## Commits

- Conventional Commits, one logical change per commit, every commit builds.
- No "wip"/"fix"/"stuff" messages.

## Not acceptable

- Placeholder packages, stub tests that assert true, or empty TODOs.
- Business logic in `app/` or `main.go`; SQL composed outside the store adapter.
- A dependency added without pinning it and recording why.
- Any code path that exposes/encourages exposing the daemon socket, bulk-deletes
  volumes, or lets a mutation reach an observe-mode host.
- Real infrastructure data (hosts, IPs, private image refs) committed anywhere.

## When the book is wrong

Open the disagreement as a proposed ADR and amend the book. Do not diverge
silently.
