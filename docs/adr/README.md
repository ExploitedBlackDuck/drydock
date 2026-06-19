# Architecture Decision Records

These ADRs are the durable record of Drydock's architectural decisions. They are
committed as the opening history of the repository (PROJECT-BOOK §6/§13).

Deviating from a decision is a **new ADR with a "Supersedes" link**, not an
undocumented edit. Each record states its Context, Decision, and Consequences.

| ADR | Title | Status |
| --- | ----- | ------ |
| [0001](./0001-wails-v2.md) | Wails v2 now, v3 as a contained migration | Accepted |
| [0002](./0002-svelte-frontend.md) | Svelte for the frontend | Accepted |
| [0003](./0003-docker-go-sdk.md) | Talk to the Engine API via the official Go SDK | Accepted |
| [0004](./0004-argv-no-shell.md) | Any subprocess is spawned argv-style; never via a shell | Accepted |
| [0005](./0005-ssh-first-transport.md) | Remote access is SSH-first; the daemon is root-equivalent | Accepted |
| [0006](./0006-no-telemetry.md) | No telemetry | Accepted |
| [0007](./0007-embedded-sqlite.md) | Embedded pure-Go SQLite for persistence | Accepted |
| [0008](./0008-api-version-negotiation.md) | Engine API version negotiation + minimum supported engine | Accepted |
| [0009](./0009-secrets-keyring-aead.md) | Secrets in the OS keyring; sensitive data AEAD-sealed | Accepted |
| [0010](./0010-operation-capture-audit-log.md) | Operation capture and a hash-chained audit log | Accepted |
| [0011](./0011-typed-operation-catalog-preview-confirm.md) | Typed operation catalog; destruction is preview-and-confirm | Accepted |
| [0012](./0012-signed-notarized-packaging.md) | Signed/notarized macOS, reproducible Linux packaging | Accepted |
| [0013](./0013-observe-mode.md) | Per-host observe (read-only) mode | Accepted |
| [0014](./0014-composition-root-at-repo-root.md) | Composition root at the repository root | Accepted |
