# Drydock

**A desktop control panel for Docker hosts — local and remote — built with Wails.**

Drydock makes a fleet of Docker hosts legible from one window: connect to local
and remote engines safely over SSH, browse and control containers, follow logs
and live stats, manage images/volumes/networks with a *preview before you
delete*, see exactly what's eating disk, and keep an audit trail of everything
that changed.

> **Status:** early development. The engineering contract is
> [`PROJECT-BOOK.md`](./PROJECT-BOOK.md); work proceeds phase by phase (§7.10).
> The current milestone is **P0 — skeleton & charter scaffolding**.
>
> `drydock` is a placeholder project/module name; rename freely.

## Why Drydock

- **Remote done right — SSH-first.** Remote engines are reached by dialing the
  remote Docker socket over SSH, reusing your existing keys and agent. The
  Docker API is root-equivalent, so Drydock **never opens, configures, or
  suggests opening an unauthenticated TCP socket**, and warns if it detects one
  ([ADR-0005](./docs/adr/0005-ssh-first-transport.md)).
- **Safe by construction around deletion.** Every prune/remove/`down -v` is
  preview-and-confirm with a per-category impact (containers, dangling vs unused
  images, **build cache**, volumes). **Named volumes are never bulk-deleted** —
  each is confirmed individually
  ([ADR-0011](./docs/adr/0011-typed-operation-catalog-preview-confirm.md)).
- **Accountable.** An append-only, hash-chained audit log records what changed,
  on which host, when, and what authorized it
  ([ADR-0010](./docs/adr/0010-operation-capture-audit-log.md)).
- **Observe-mode hosts** reject all mutation in the core, not just in the UI
  ([ADR-0013](./docs/adr/0013-observe-mode.md)).
- **No telemetry.** Zero analytics, crash reporting, or phone-home. Logs stay
  local ([ADR-0006](./docs/adr/0006-no-telemetry.md)).

Targets **macOS and Linux**.

## Architecture

`internal/core` is a headless library you could ship without a GUI — the Wails
layer is one consumer, not the system of record. Dependencies point inward;
adapters (Docker SDK, SSH, SQLite, OS keyring) translate at the edges. See the
book's §3 and the [ADRs](./docs/adr/).

## Building from source

Prerequisites: **Go 1.26.4+** (pinned via `go.mod` `toolchain`), **Node 22+**,
and the [Wails v2](https://wails.io) system dependencies for your platform.

```sh
# one-time: install pinned developer tooling (gofumpt, golangci-lint,
# govulncheck, wails, go-task)
task tools

# run the quality gates
task lint
task test

# build a desktop binary into build/bin/
task build

# run with hot reload
task dev
```

If you do not have [go-task](https://taskfile.dev) yet, install it with
`go install github.com/go-task/task/v3/cmd/task@v3.40.1`, then run `task tools`.

## Project layout

```
main.go              composition root (thin; ADR-0014)
app/                 Wails binding layer (only place importing wails)
shell/               Wails runtime behind our own interface (ADR-0001)
internal/
  platform/          cross-cutting infrastructure (config, logging)
  core/              headless engine (no UI imports) — added per phase
  adapters/          Docker SDK, SSH, SQLite, keyring — added per phase
frontend/            Svelte + TypeScript + Vite
docs/adr/            architecture decision records
```

## Security

Drydock holds root-equivalent access to the hosts it connects to. See
[`SECURITY.md`](./SECURITY.md) for the vulnerability disclosure policy and the
book's §8 for the threat model.

## License

[Apache-2.0](./LICENSE). Third-party notices in [`NOTICE`](./NOTICE).
