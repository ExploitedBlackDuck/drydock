# Project Book — Drydock

**A desktop control panel for Docker hosts — local and remote — built with Wails.**

Drydock makes a fleet of Docker hosts legible from one window: connect to local and remote engines safely over SSH, browse and control containers, follow logs and live stats, manage images/volumes/networks with a *preview before you delete*, see exactly what's eating disk, and keep an audit trail of everything that changed. It is the cross-host, GUI answer to the tools that are either terminal-only and local-first, or a heavyweight web service.

> **Drydock** is a placeholder name in the same family as Conductor (rclone) and Recon Deck (ProjectDiscovery); rename freely.

Targets **macOS and Linux**. This book is the engineering contract for the project: the standards and specification any implementation is held to.

> **Verified facts this book relies on (June 2026):** Wails v3 is still pre-release (alpha line, `v3.0.0-alpha.96`); maintainers call the API "reasonably stable" with production users, but it is not a tagged stable release. Wails v2 is the stable production line. Docker exposes a full HTTP **Engine API** (over a Unix socket by default), with a first-party Go SDK (`github.com/docker/docker/client`) that negotiates API versions. The Docker daemon runs as root, so access to its API is root-equivalent on the host; the recommended way to reach a remote engine is over **SSH**, not an exposed TCP socket. These facts drive ADR-0003 and ADR-0005. Re-verify before locking dependency versions.

---

## 0. How to use this book

This document is the source of truth. Implementers do not invent architecture, pick libraries, or reinterpret scope on their own; where the book appears wrong, the path is to **raise it and amend the book**, not to diverge silently.

1. **Build in the phases defined in §7.10, in order.** Do not start a phase until the previous phase's acceptance gate is green. Do not work ahead.
2. **Every phase ends with a verifiable gate** — a command that passes or fails. "It looks done" is not a gate. Run it; record the output in the PR.
3. **The Engineering Charter (§2) is not advisory.** Each rule maps to a lint rule, a test, or a review-checklist item. Code that violates it is rejected, not patched later.
4. **Write the ADRs (§6) into the repo as the opening commits.** Deviating from a decision is a new ADR with a "supersedes" link, not an undocumented edit.
5. **No scaffolding theater.** No empty packages, no placeholder `TODO` functions, no stub tests asserting `true`. Build thin vertical slices that work end to end, then widen them.
6. **Keep `git` history legible.** Conventional Commits, one logical change per commit, no "wip"/"fix"/"stuff" messages, no commits that don't build.

The failure mode we design against: **code that compiles and demos but reads as machine-extruded** — no boundaries, no error taxonomy, no real tests, magic strings, a 600-line `main.go`. The whole book exists to prevent that.

**Two things are special about this app:** it talks to an interface that is **root-equivalent on the host**, and it can **delete data**. So remote access is **SSH-first and the tool never exposes or encourages exposing the daemon** (ADR-0005, §7.2), and **every destructive operation is preview-and-confirm with volumes never bulk-deleted** (ADR-0011, §7.4). Those two properties are the centre of the design.

---

## 1. The problem, goals and non-goals

### 1.1 The problem (validated pain points)
The research behind this book points at a consistent set of complaints; Drydock exists to answer them:

1. **Remote management is unreliable.** The popular terminal UI handles local Docker well but its SSH/remote-context support is flaky — tunnels time out, and `exec`/interactive sessions on remote hosts behave unpredictably — so people fall back to the raw CLI for remote work or write wrapper scripts just to manage the SSH tunnels and contexts.
2. **Connecting remotely *safely* is hard and dangerous.** The Docker API is root-equivalent; exposing an unauthenticated TCP socket is an internet-scannable root backdoor, and even TLS-over-TCP hands root to anyone holding the certificate. SSH is the recommended transport, but key/`DOCKER_HOST`/context setup is fiddly enough that people get it wrong.
3. **Disk fills up and cleanup is a data-loss trap.** "No space left on device" hits mid-deployment; `prune` is risky because `--volumes` can permanently delete databases and uploads; the build cache is often the largest consumer yet is easy to miss; and the dangling/named/in-use distinctions plus opaque "reclaimable" numbers leave people unsure what's safe to remove.
4. **Debugging means juggling logs and stats across containers,** piecing a story together across terminals.
5. **The desktop app is heavy and licensed; the daemon is a root single-point-of-failure;** rootless and Podman (API-compatible) are rising. A management tool shouldn't *require* the desktop app.
6. **`docker stats` is ephemeral** — no short-horizon history to catch a memory leak or a restart loop after the fact.
7. **Multiple hosts have no clean single pane.**

### 1.2 Goals
- A native-feeling control surface for **local and remote** Docker engines, with remote done *right*: **SSH-first**, reliable, multi-host.
- A **headless core** fully usable and testable without the GUI — the UI is presentation, never the system of record.
- **Safe by construction around deletion:** prune/remove/down are preview-and-confirm; volumes are never bulk-deleted; a host can be marked observe-only.
- **Legible disk and resources:** what's reclaimable, by category (including build cache), plus short-horizon stat history.
- **Accountable:** an append-only audit trail of what changed, on which host, when.
- Cross-platform (macOS + Linux) from one codebase; code quality that survives a hostile senior review and a repo credible enough to publish.

### 1.3 Non-goals
- Windows is out of scope for v1 (not excluded by design — just not a tested/supported target yet).
- Mobile is permanently out of scope.
- Not a container orchestrator, a Kubernetes dashboard, a registry, or a CI system. Drydock manages running engines; it does not schedule clusters.
- Not a multi-user server with RBAC. Drydock is a local-first, single-operator desktop tool.
- We do not reimplement Docker or fork the engine; we speak its Engine API.
- No telemetry, analytics, or phone-home. Silent by default (ADR-0006).

---

## 2. Engineering Charter — the non-negotiables

The anti-"vibe-coded" section. Each rule is enforceable; §12 checks every one.

### 2.1 Architecture boundaries
- **The core tree imports zero UI code.** No `wails` import anywhere under `internal/core/...`. The core compiles and its tests run with `go test ./internal/core/...` on a machine with no webview and no Docker daemon. Enforced by `depguard`.
- **Dependencies point inward.** Frontend → Wails bindings → application services → domain core. Never the reverse. The domain core knows nothing about the Docker SDK's types, SSH, SQLite, or the OS keyring; adapters translate at the edges.
- **Interfaces are declared by the consumer, kept small,** defined where used — not in a junk-drawer `interfaces.go`. A service that needs the engine declares a small `Engine` port next to itself; the real Docker-SDK client lives in an adapter. Likewise `Store`, `SecretStore`, and `Dialer` (transport) are consumer-defined ports with adapter implementations.

### 2.2 Errors
- **No naked `panic` in library code.** Permitted only in `main` for unrecoverable startup failures, and in tests.
- **Wrap with context using `%w`:** `fmt.Errorf("connecting to host %q: %w", host, err)`. Each wrap adds *what we were doing*.
- **Typed/sentinel errors for anything a caller branches on** (`var ErrHostUnreachable = errors.New(...)`, `var ErrObserveMode = errors.New(...)`), checked with `errors.Is`/`errors.As`. No string-matching on `err.Error()`.
- **Errors crossing the Wails boundary are mapped to a typed DTO** (`{code, message, retryable}`) drawn from the enumerated error-code catalog (§8.4). The frontend never receives or parses a raw Go error string.

### 2.3 Concurrency & lifecycle
- **`context.Context` is the first parameter** of every function that does I/O, opens a connection, or makes an API call. Cancellation propagates from the UI ("stop", "disconnect") to the SDK request and the SSH connection.
- **No goroutine without a defined exit.** Every `go func()` (log tails, stats streams, the event watcher) has an owner that can stop it and a place that waits for it.
- **No global mutable state.** No package-level vars holding services, config, clients, or host connections. Constructed in `main`/`app`, injected via constructors.
- **Connections are supervised:** open, health-check, reconnect-with-backoff, graceful close. No leaked SSH tunnels or open API streams after a host is disconnected or the app quits — verified by an integration test.

### 2.4 Logging & observability
- **Structured logging via `log/slog`.** No `fmt.Println`/`log.Printf` debugging in the tree. Levels used meaningfully.
- **Secrets never hit the logs.** SSH key paths/passphrases, TLS client keys, and the data key are redacted at the logging boundary. A `redact()` helper exists and a test proves a known secret never appears in emitted lines.
- **Operational logs are distinct from the audit log.** slog output is for the operator/developer and is rotated/disposable; the audit log (§7.8) is durable, append-only, and tamper-evident. They are never conflated.

### 2.5 Tests
- **Behavior, not coverage theater.** No `assert.True(t, true)`, no tests that only check a constructor returns non-nil.
- **Table-driven** for mappers, validators, the prune-impact calculator, and the impact-rule engine.
- **The Engine, transport, store, and secret layers are mockable** because they sit behind interfaces (§2.1). Pure logic (mapping SDK responses to domain types, prune-impact calculation, destructive-op guards, command/option assembly, impact rules) is unit-tested with no daemon and no real database.
- **Integration tests that need a real daemon or keyring** are behind `//go:build integration` and a presence check; they skip cleanly when unavailable so `go test ./...` is always green on a bare machine.
- **Mappers are tested against captured real fixtures** under `testdata/api/` — real Engine API JSON responses (container inspect, `system df`, events), sanitized (§2.10), not hand-typed approximations.
- **Store tests run against a real SQLite file** (temp dir), exercising migrations up, queries, and the encrypted-column round-trip.

### 2.6 Tooling gates (CI-enforced)
- `gofumpt` (stricter than `gofmt`).
- `golangci-lint` with a **curated, committed `.golangci.yml`**. Minimum: `govet, staticcheck, errcheck, ineffassign, unused, depguard, gocritic, revive, bodyclose, contextcheck, errorlint, nilerr, gosec, sqlclosecheck, rowserrcheck`.
- `go vet ./...` clean.
- `govulncheck ./...` clean — a tool that holds root-equivalent access must not ship a known-vulnerable dependency.
- Frontend: ESLint + Prettier + `tsc --noEmit` (strict TS). No `any` without an inline justification.
- CI fails on any of the above. No ignored "warning" tier.

### 2.7 Hygiene
- **Every exported symbol has a godoc comment** starting with the symbol name.
- **No commented-out code** in commits.
- **No `TODO`/`FIXME` without an issue reference** (`// TODO(#42): ...`). Bare TODO is a lint failure.
- **Dependencies pinned** (`go.mod`/`go.sum`; frontend lockfile committed). No `@latest` in build scripts.
- **No magic strings/numbers** crossing boundaries. Operation kinds, object types, transport kinds, error codes, and audit action types are typed constants in one place.
- **No AI/authorship fingerprints in the repository.** No "generated by" comments, no assistant names in files, commit authors, or metadata. The repository reads as a normally-authored project.

### 2.8 Frontend discipline
- Typed API only: the frontend calls **generated Wails bindings**, never hand-rolled stringly-typed bridges.
- State lives in defined stores, not scattered component-local state; runtime state (live stats/logs/events) and view state are distinguished.
- Components small and role-named. No 800-line "App.svelte".
- Event payloads from Go are typed (generated) and validated at the boundary — never `JSON.parse`-and-pray.

### 2.9 Data, history & audit handling
- **A mutating operation is a recorded event.** Every start/stop/restart/remove/prune/exec/compose-action carries its target, host, resolved parameters, and result into persistent history. History rows are immutable.
- **Capture the result.** Each operation's outcome (and, for prune, exactly what was removed and how much reclaimed) is stored with it.
- **The audit log is append-only and tamper-evident** (hash-chained — §7.8), with emphasis on destructive operations and the acknowledgements that authorized them.
- **Sensitive persisted data is encrypted at rest** (ADR-0009): any saved sensitive values are sealed before they touch disk; SSH/TLS credentials are referenced, not copied.

### 2.10 No real infrastructure data in the repository
- Fixtures under `testdata/` are sanitized: real hostnames, IPs, image references from private registries, container environment values, and any tokens are replaced with `example`/documentation placeholders.
- A pre-commit hook **and** a test scan the tree for patterns that look like real infrastructure — public IPs outside reserved/documentation ranges, non-example registrable domains, anything resembling a token or private registry path — and fail the build on a hit.

---

## 3. Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Frontend (Svelte, in webview)                                 │
│  - host switcher, containers, logs/stats/exec, compose          │
│  - images/volumes/networks (+ prune preview), disk, history     │
│  - calls generated bindings; subscribes to typed events         │
└───────────────▲────────────────────────────────┬──────────────┘
                │ generated bindings              │ typed events (stats/logs/events)
┌───────────────┴────────────────────────────────▼──────────────┐
│  app/  — Wails binding layer (ONLY place importing wails)       │
│  - thin: UI calls → core service calls                          │
│  - map core errors → typed DTOs; core events → UI events        │
│  - NO business logic                                            │
└───────────────▲────────────────────────────────┬──────────────┘
┌───────────────┴────────────────────────────────▼──────────────┐
│  internal/core/  — headless engine (no UI imports)              │
│  - hosts     : connection lifecycle, observe-mode, multi-host   │
│  - engine    : container/image/volume/network/compose services  │
│  - prune     : prune-impact calculation (preview-and-confirm)   │
│  - options   : operation option catalog + builder + rules       │
│  - history   : operation/resource history queries               │
│  - audit     : append-only, hash-chained log                    │
│  - store     : Store port + query layer                         │
│  - domain    : Host/Container/Image/Volume/Network/Operation... │
│  - ports     : Engine, Dialer (transport), Store, SecretStore   │
└───────────────▲────────────────────────────────┬──────────────┘
┌───────────────┴────────────────────────────────▼──────────────┐
│  internal/adapters/                                             │
│  - dockerengine: Docker Go SDK impl of Engine                   │
│  - sshdialer   : SSH transport (dials the remote engine)        │
│  - tlsdialer   : optional mTLS-over-TCP transport               │
│  - sqlitestore : SQLite impl of Store (migrations)              │
│  - keyring     : OS keyring impl of SecretStore                 │
└──────────────────────────────────────────────────────────────┘
```

Defining property: **`internal/core` is a library you could ship without a GUI.** You could put a CLI or a local HTTP server in front of it tomorrow. The Wails layer is one consumer, not the system. The engine is reached through a `Dialer` port so the core neither knows nor cares whether a host is local, SSH-tunnelled, or TLS-TCP.

---

## 4. Tech stack

| Concern | Choice | Notes |
|---|---|---|
| Desktop shell | **Wails v2 (stable)** | ADR-0001. Runtime abstracted behind a `shell` package so v3 migration is contained. |
| Core language | **Go 1.23+** | Pin the toolchain in `go.mod`. |
| Frontend | **Svelte + TypeScript + Vite** | ADR-0002. |
| Frontend state | Typed Svelte stores | Live stats/logs/events stream + view state kept distinct. |
| Styling | Plain CSS / CSS modules + design tokens | Consult the `frontend-design` skill when building the UI. |
| Engine client | **Docker Go SDK (`github.com/docker/docker/client`)** | ADR-0003. API version negotiation; streaming logs/stats/events. Also reaches API-compatible engines (e.g. a Podman socket). |
| Remote transport | **SSH (`golang.org/x/crypto/ssh`)**, optional mTLS-over-TCP | ADR-0005. SSH-first; reuse existing keys/agent. Never an exposed unauthenticated socket. |
| Datastore | **SQLite via `modernc.org/sqlite` (pure Go, no CGO)** | ADR-0007. Host profiles, audit log, short-horizon resource history. Forward-only embedded migrations. |
| At-rest encryption | **XChaCha20-Poly1305 (`golang.org/x/crypto/chacha20poly1305`)** | ADR-0009. Sensitive saved values sealed app-side with a per-install data key. |
| Secrets | **OS keyring (`github.com/zalando/go-keyring`)** | ADR-0009. macOS Keychain / Linux Secret Service. Holds the data key; SSH auth prefers the agent. |
| Logging | `log/slog` | JSON to rotating file + pretty in dev. Distinct from the audit log. |
| Config | TOML via a single typed loader | XDG-respecting paths (§4.1). |
| Tests | stdlib `testing` + `testify` (assert/require) | No BDD frameworks. |
| Lint | `golangci-lint`, `gofumpt`, `govulncheck` | Committed config (§2.6). |
| Task runner | `Taskfile` (go-task) | |

### 4.1 Filesystem & config
XDG-style resolver:
- Config: `$XDG_CONFIG_HOME/drydock/config.toml` (Linux `~/.config/drydock/`, macOS `~/Library/Application Support/drydock/`).
- Data (SQLite DB, audit log): platform data dir, created with restrictive permissions (`0700` dir, `0600` files).
- **Never** write into the app bundle or CWD. Drydock does not create or modify `~/.ssh` or `~/.docker` contents.

---

## 5. Repo layout & conventions

> **Amended by ADR-0014:** the composition root is `main.go` at the repository
> root (not `cmd/drydock/main.go`), because the Wails v2 toolchain expects the
> `main` package at the module root. It stays thin; the rule is unchanged.

```
drydock/
├── main.go                   # composition root: build deps, wire, run. Thin. (ADR-0014)
├── app/                      # Wails binding layer (only place importing wails)
│   ├── app.go
│   ├── events.go             # typed event names + emit helpers
│   └── errors.go             # core error -> DTO mapping (uses §8.4 catalog)
├── shell/                    # Wails runtime behind our own interface (ADR-0001)
├── internal/
│   ├── core/
│   │   ├── domain/           # Host/Container/Image/Volume/Network/Operation/... + validation
│   │   ├── hosts/            # connection lifecycle, observe-mode, multi-host registry
│   │   ├── engine/           # container/image/volume/network/compose services
│   │   ├── prune/            # prune-impact calculation (preview-and-confirm)
│   │   ├── options/          # operation option catalog, builder, impact-rule engine
│   │   ├── history/          # operation + resource history queries
│   │   ├── audit/            # append-only hash-chained audit log
│   │   ├── store/            # Store port + query helpers (no SQL dialect here)
│   │   └── ports/            # Engine, Dialer, Store, SecretStore, Clock...
│   └── adapters/
│       ├── dockerengine/     # Docker Go SDK impl of Engine
│       ├── sshdialer/        # SSH transport
│       ├── tlsdialer/        # optional mTLS-over-TCP transport
│       ├── sqlitestore/      # SQLite impl of Store (+ embedded migrations)
│       └── keyring/          # OS keyring impl of SecretStore
├── catalogs/                 # versioned operation option catalog (embedded) — see §7.5
│   └── docker@<apiversion>.toml
├── migrations/               # forward-only SQL migrations (embedded)
├── frontend/
│   └── src/{lib/api,lib/stores,lib/components,routes}
├── testdata/api/             # sanitized captured Engine API JSON fixtures (§2.10)
├── .golangci.yml
├── Taskfile.yml
├── docs/adr/
├── CONTRIBUTING.md
├── SECURITY.md               # vulnerability disclosure policy (§10)
├── LICENSE
├── NOTICE                    # third-party license notices (§10)
├── CHANGELOG.md              # keepachangelog format, semver
└── README.md
```

Conventions:
- Package names are nouns, lower-case, no stutter (`hosts.Registry`, not `hosts.HostRegistry`).
- The root `main.go` is the only composition root (ADR-0014). Business logic there is a defect.
- Generated code is marked generated and reviewed when regenerated.

> **Shared-kit note:** the slog setup, XDG resolver, error-DTO mapping, the `Store`/`SecretStore` ports, and the SQLite migration harness are generic and identical to the sibling projects (Conductor, Recon Deck). If a shared module already exists, depend on it; otherwise build cleanly here and extract on second use.

---

## 6. Architecture Decision Records

Commit these into `docs/adr/` as the opening commits.

### ADR-0001 — Wails v2 now, v3 as a contained migration
**Context.** v3 is the framework's future (GTK4 on Linux, better bindings) but is still alpha as of mid-2026, not a tagged stable release. v2 is stable and battle-tested.
**Decision.** Build on v2. Confine all Wails runtime calls behind a thin internal `shell` package depending on *our* interface. v3 migration then touches `app/` and `shell/`, not the core.
**Consequences.** A little indirection; near-zero core churn when v3 stabilizes. Revisit when v3 cuts a stable tag.

### ADR-0002 — Svelte for the frontend
**Context.** A dense control surface (tables, live logs/stats), not a content site. We want minimal webview runtime overhead and low ceremony; the reactivity model fits live streams well.
**Decision.** Svelte + TS + Vite.
**Consequences.** Smaller bundle and simpler state than React.

### ADR-0003 — Talk to the Engine API via the official Go SDK; do not shell out to the `docker` CLI
**Context.** Docker exposes a complete HTTP Engine API; the first-party Go SDK negotiates API versions and gives typed access to containers, images, volumes, networks, `system df`, logs, stats, and the event stream. Shelling the CLI means parsing human-formatted text, losing streaming, and adding a binary dependency.
**Decision.** Use the Docker Go SDK behind an `Engine` port. The same client speaks to any API-compatible engine reachable through the `Dialer` (including a Podman socket). Compose actions use the Compose-over-API path where available, falling back to documented project semantics — never string-built shell.
**Consequences.** Typed, streaming, version-negotiated access; no text scraping. Cost: we track the SDK and a minimum supported API version (ADR-0008). The `Engine` adapter is the main thing to test against fixtures.

### ADR-0004 — Any subprocess is spawned argv-style; never through a shell
**Context.** Drydock is mostly SDK + library calls, but `exec` into a container, an SSH helper, or a compose shell-out must never interpolate operator input into a shell.
**Decision.** Any process is launched with explicit argv (`exec.CommandContext(ctx, bin, args...)`); container `exec` uses the Engine API exec endpoints with argv, not a shell string. No `sh -c`, no concatenation, no operator input reaching a shell.
**Consequences.** Eliminates a command-injection class. Enforced by `gosec` + review.

### ADR-0005 — Remote access is SSH-first; the daemon API is treated as root-equivalent; the tool never exposes or encourages exposing the socket
**Context.** The Docker daemon runs as root, so access to its API (the `/var/run/docker.sock` Unix socket) is equivalent to unrestricted root on the host. Exposing an unauthenticated TCP socket (`tcp://0.0.0.0:2375`) is an internet-scannable path to a root shell; even TLS-over-TCP grants root to any holder of the client certificate. SSH transport reuses existing keys and the host's own access controls.
**Decision.** Drydock reaches remote engines by **dialing the remote Unix socket over SSH** (reusing the operator's keys/agent), as the default and recommended path. **mTLS-over-TCP** is supported only for hosts already configured that way, with the client key treated as a root-equivalent secret (ADR-0009). Drydock **never opens, configures, or suggests opening an unauthenticated TCP socket**, and surfaces a warning if it detects it is talking to one. A connected host is shown with its transport and trust level so the operator always knows how root-equivalent access is travelling.
**Consequences.** Safe-by-default remote management with no new attack surface on the host. Cost: SSH connection/tunnel management must be robust (the precise area where existing TUIs are unreliable) — it gets dedicated supervision and tests.

### ADR-0006 — No telemetry
**Decision.** Zero analytics, crash reporting, or network calls the user didn't initiate. Logs are local. Stated in the README as a feature.

### ADR-0007 — Embedded SQLite (pure-Go) for host profiles, audit log, and short-horizon resource history
**Context.** Connection details for multiple hosts, an audit trail, and a rolling window of resource samples (to catch a leak or a restart loop) must persist and be queryable; live container state is ephemeral and stays with the engine. Cross-platform Wails builds favour a CGO-free toolchain.
**Decision.** SQLite via the **pure-Go `modernc.org/sqlite` driver** — a single file in the platform data dir — with **versioned, embedded, forward-only migrations** behind a schema-version check. The core depends on a `Store` port; SQL lives only in `sqlitestore`. Resource history is a bounded, rolling retention; full live state is never persisted.
**Consequences.** Queryable history without a server; CGO-free keeps cross-compilation trivial. Cost: pure-Go SQLite has no transparent at-rest encryption, so sensitive values are sealed at the app layer (ADR-0009). Alternatives (flat files, bbolt) rejected for a history/query workload.

### ADR-0008 — Engine API version negotiation and a declared minimum supported engine
**Context.** Docker engines in the field run a range of API versions; the SDK supports negotiation. There is no single pinned external binary to verify (we link the SDK), so the risk is API drift, not a tampered binary.
**Decision.** The `Engine` adapter negotiates the API version per host and records the engine + API version on every operation. Drydock declares a **minimum supported API version**; a host below it connects in a clearly-labelled reduced-capability mode rather than failing opaquely. Capability differences are surfaced in the UI, never guessed.
**Consequences.** Works across a heterogeneous fleet; behaviour differences are explicit. Cost: capability checks per feature.

### ADR-0009 — Secrets in the OS keyring; sensitive persisted data encrypted at the application layer
**Context.** SSH passphrases and TLS client keys are root-equivalent; pure-Go SQLite (ADR-0007) offers no transparent encryption. Drydock should not become a second store of host credentials.
**Decision.** SSH authentication **prefers the agent and on-disk keys the operator already manages** — Drydock references them, and does not copy or store private keys. Where a secret must be retained (e.g. an optional SSH passphrase or a TLS client-key passphrase), it is stored in the **OS keyring**, never in the DB or config. A per-install **data key** in the keyring seals any sensitive persisted field via **AEAD (XChaCha20-Poly1305)** before disk. The data dir is OS-permission-restricted.
**Consequences.** No new plaintext credential store; secrets never on disk or in logs. Cost: a keyring round-trip, unit-tested.

### ADR-0010 — Operation capture and an append-only, hash-chained audit log
**Context.** A tool with root-equivalent reach that can delete data must be able to answer, durably and verifiably, what it changed and what authorized it.
**Decision.** Every mutating operation's target, host, resolved parameters, and result are persisted (ADR-0007/0009). Every consequential action (start/stop/restart/remove, **prune with its exact impact**, exec, compose up/down, host connect/disconnect, observe-mode changes) is recorded as an **append-only audit entry whose hash chains to the previous entry** (`hash = SHA256(prev_hash || canonical(entry))`). The chain is verifiable and exportable.
**Consequences.** A complete, tamper-evident record. Cost: storage, managed by retention + encryption.

### ADR-0011 — Operations are a typed catalog with impact warnings, and destruction is preview-and-confirm
**Context.** Docker operations carry a large, evolving set of options, and several are destructive or risky — `prune` (especially `--volumes`), `rm -f` on a running container, `down -v`, `exec` as root. Cleanup is a leading cause of accidental data loss; volumes are excluded from default prune precisely because removing the wrong one is permanent.
**Decision.** A versioned **catalog** describes each operation's options as typed metadata (name, type, default, category, help, risk, `affects_data`, `conflicts_with`, `requires`, `impacts`). The UI is generated from it; a **builder** assembles validated API parameters (ADR-0003/0004). An **impact-rule engine** plus a **prune-impact calculator** (§7.4) show, before execution, exactly what will be affected — for prune, the precise objects and reclaimable bytes by category (containers/images/build cache/volumes), with **named volumes never bulk-deleted**: each must be individually confirmed. Destructive operations require an explicit acknowledgement recorded in the audit log, and are blocked entirely on observe-mode hosts (ADR-0013).
**Consequences.** Cleanup stops being a guessing game; the data-loss footgun is removed. Cost: the catalog is maintained per API version and tested against the engine's actual options.

### ADR-0012 — Signed/notarized macOS, reproducibly packaged Linux
**Context.** A tool that holds root-equivalent access to your servers must be trustworthy to install. An unsigned binary that fights Gatekeeper undermines that from the first launch.
**Decision.** macOS builds are **signed with a Developer ID and notarized** (hardened runtime, minimal entitlements, stapled). Linux ships as a versioned **AppImage** plus a **`.deb`**, built **reproducibly in CI** from pinned toolchains. Every release is a signed semver git tag with a maintained `CHANGELOG.md` and **published SHA-256 checksums**.
**Consequences.** Installs cleanly with verifiable provenance. Cost: a signing identity + notarization step. No silent auto-update.

### ADR-0013 — Per-host observe (read-only) mode
**Context.** Some hosts (production) should be inspectable but not mutable from a desktop tool; a single mis-click should not be able to stop a prod container or prune a prod volume.
**Decision.** A host can be marked **observe-only**. In observe mode the core **rejects every mutating operation** with `ErrObserveMode` before it reaches the engine — enforced in the core, not just hidden in the UI — and the UI presents the host as read-only. Leaving observe mode is an explicit, audited action.
**Consequences.** A hard, testable guardrail for sensitive hosts. Cost: a mode check on the mutating path (cheap, and unit-tested).

---

The following ADRs govern **Phase 2** (the post-1.0 roadmap, §7.12). They are recorded now so the decisions are fixed before that work begins; the build order in §0 still holds — none of this is started until the v1 plan (§7.10) is green.

### ADR-0016 — Compose changes are plan-first: `up` is preview-and-confirm, not a black box
**Context.** `compose up` against a changed project silently recreates containers, recreates or removes networks, and can drop anonymous volumes — the operator learns what happened only afterwards. This is the same surprise-and-data-loss class the book already guards on the *delete* path (ADR-0011, §7.4), now on the *apply* path. Compose decides what to recreate by comparing a per-service config hash stored on each container as the `com.docker.compose.config-hash` label; reproducing that hash from outside Compose is version-fragile — the stored label and the `compose config --hash` subcommand are documented to diverge (Compose mutates the service, e.g. injecting the resolved image digest, before hashing for the label), and a separate `com.docker.compose.config-hash-dependencies` label was later added.
**Decision.** Before applying, Drydock computes a **`ComposePlan`**: the desired project (parsed with the pinned `compose-go`) diffed against observed engine state, classifying each service as create / recreate / start / no-op and each network and volume as create / recreate / remove, and flagging recreations that interrupt a running container or drop an anonymous volume. Convergence detection uses **the same Compose library version Drydock applies with** — never the `config --hash` CLI — and where a hash cannot be matched with confidence the plan **degrades to a coarser, clearly-labelled diff** (image/tag/digest, published ports, env and mount presence) rather than asserting precision it does not have. The plan is computed through the SDK / `compose-go` path, never a shell (ADR-0003/0004); Compose's own `up` remains the authority that actually converges state. Destructive plan elements route through the §7.4 acknowledgement-and-audit path. **A precise plan requires the project source.** Drydock locates it from the running containers' `com.docker.compose.project.config_files` / `working_dir` labels; it computes a **full plan when that source is accessible to it** (a local host, or compose files the operator points it at). It does **not** read arbitrary files off a remote host — the transport is a Docker-socket dialer, not a filesystem (ADR-0005) — so when the source is not locally accessible the plan enters an explicit **`source-unavailable` degraded state**: a labelled best-effort diff built from container labels and image/digest/port/env/mount inspection, never a false "no changes." (A remote-source-read capability, if ever wanted, would be a separate, deliberate transport decision with its own ADR.)
**Consequences.** `compose up` becomes legible before it runs. Cost: the diff must be computed with the pinned Compose library and **re-verified on every Compose upgrade** — the hash derivation is out-of-spec and has changed between versions — and remote stacks without locally-accessible source get the honest degraded plan, not a precise one.

### ADR-0017 — Published-port exposure is computed and surfaced; loopback vs all-interfaces is first-class
**Context.** The book's spine is that the daemon is root-equivalent and must never be needlessly exposed (ADR-0005). The same risk re-appears one layer up: a container that publishes a port to `0.0.0.0`/`::` is reachable on every interface, and on a public-IP host that is the public internet — usually unintentionally, since `-p 5432:5432` binds all interfaces by default while `-p 127.0.0.1:5432:5432` does not. Operators have no single place to see what is reachable from outside, across a fleet.
**Decision.** Drydock computes a per-host, fleet-aggregable **exposure map** from container port bindings (host IP, host port, container port, protocol) and classifies each binding by reach: **loopback** (`127.0.0.1`/`::1`), **private/LAN**, or **all-interfaces** (`0.0.0.0`/`::`/empty host IP). All-interfaces bindings on a host reached over a non-loopback transport are flagged prominently. This is **read-only insight, not enforcement**: Drydock never edits a firewall or rebinds a port. It makes accidental exposure legible the way the prune preview makes deletion legible.
**Consequences.** "What have I exposed?" gets one answer. Cost: reach is classified at the daemon layer only — Drydock cannot see an upstream cloud security group or host firewall, so the UI states its scope honestly: it reports the *binding*, not the full network path. **A container using `network_mode: host` publishes no bindings yet shares the host's network namespace, so its exposure is not derivable from bindings;** such containers are listed explicitly as "host network — exposure not derivable from port bindings" rather than shown as exposing nothing (the dangerous false negative). The map reports the *binding*, not the host's actual reachable addresses.

### ADR-0018 — A host timeline merges the engine event stream with Drydock's audit log, without weakening the audit log
**Context.** Drydock records what *it* changed (ADR-0010), but most of what happens to a host happens outside Drydock — CLI actions, other tools, and the daemon itself: a container `die` with an `exitCode`, an `oom`, a `health_status` flip. The Engine exposes all of this on the event stream (with the exit code and the container's labels in the actor attributes), already consumed for live updates and restart-loop detection (§7.3/§7.6). Reconstructing "why did this die at 03:00, and who changed it" is currently manual.
**Decision.** Drydock persists a bounded, rolling **host timeline** that interleaves mapped engine events with references to its own audit entries, correlated by host / container / time, as the explain-and-detect-drift surface. The **audit log stays the separate, append-only, hash-chained record of Drydock-authored mutations** (ADR-0010): engine events are **untrusted input** — typed-mapped, fixture-tested, and **never written into the hash chain**. The timeline reuses the §7.7 rolling-retention approach.
**Consequences.** A single "what happened on this host" view, including out-of-band change. Cost: the event stream can gap across a reconnect, so the timeline is best-effort and **labelled as such**, explicitly distinct from the audit log's completeness guarantee for Drydock's own actions.

### ADR-0019 — Image provenance and staleness in core; vulnerability scanning, if ever added, is an optional adapter behind a port
**Context.** A security-minded operator wants to know a running container is on a stale or drifted image — the tag has moved since the container started, or the running digest no longer matches the tag. The Engine exposes the running image's `RepoDigests`; comparing against the tag's current registry digest answers "is this stale." A full CVE scan is a different weight class — a large, fast-moving vulnerability database and network egress — and must not become a core dependency or quietly violate the no-telemetry posture (ADR-0006).
**Decision.** Drydock surfaces **lightweight provenance** in the core: image age, **tag-vs-digest drift** (running digest ≠ the tag's current registry digest), and untagged / `:latest` ambiguity — from the Engine's image data plus an **explicit, operator-initiated** registry digest check (no background phone-home). **The digest check runs through the daemon, not the desktop** — via the Engine's distribution-inspect endpoint — so it reflects **the host's** registry reachability and credentials, not the desktop's: a private registry reachable only from the host gives a correct answer, and the desktop's own credentials are never silently substituted. **Vulnerability scanning stays out of core:** if it is ever added it is a `Scanner` **port** with an optional adapter wrapping a scanner the operator already trusts, invoked explicitly, results never silently transmitted — and it gets its own ADR then, not now.
**Consequences.** A useful staleness signal with no scope creep and no telemetry footprint, correct even for host-only-reachable registries. Cost: the check is an operator-initiated call through the host's engine (consistent with ADR-0006); any auth is the operator's existing Docker credentials, encoded per-call and **referenced, not copied or stored** (ADR-0009); if the host's daemon cannot reach the registry the result is a clear "registry unreachable from host," not a desktop-side guess.

### ADR-0020 — A volume may be snapshotted before destruction; snapshots are explicit, never automatic, and never a precondition
**Context.** The book already makes volume deletion preview-and-confirm and never bulk (ADR-0011, §7.4). The natural completion of that property is an escape hatch: let the operator capture a volume's contents *before* removing it, so a wrong confirm is recoverable. But silent automatic backups of unknown-size volumes would be their own surprise — blown disk, stalled operations.
**Decision.** Drydock offers an **explicit, operator-initiated volume snapshot**: a streamed `tar` of the volume's contents to an operator-chosen destination, taken via a throwaway helper container mounting the volume **read-only** (Engine API, argv — ADR-0004), with destination, size estimate, and expected duration shown first. It is **never automatic** and **never blocks deletion**; it is an *offered* safeguard at the confirm step for a volume that holds data. The helper is a **digest-pinned minimal image** (recorded in the versioned, embedded catalog alongside the operation options, §7.5); Drydock prefers an image **already present on the host** and otherwise pulls the pinned digest as an audited, operator-visible step, **failing closed with a clear error on an air-gapped host** rather than silently doing nothing. Because a snapshot **starts a container**, it is a mutating side effect and is therefore **blocked on observe-mode hosts like every other mutation** (ADR-0013) — there is no observe-mode exception, and nothing is lost, since deletion is already blocked there too. Snapshot and restore are audited operations (ADR-0010).
**Consequences.** The data-loss footgun gains an undo. Cost: snapshotting a large volume is slow and consumes disk (size/time stated up front; the operation is cancellable per §2.3); a snapshot is a plaintext archive **outside** Drydock's sealed store, so the UI states plainly that the operator must protect the destination; restore is a separate, deliberately-confirmed action; the pinned helper image is a supply-chain input that is verified by digest and re-pinned through the normal catalog-versioning process.

---

The following ADRs are **gap-review refinements to v1 mechanics** — not Phase 2. They make explicit the contracts behind promises the charter already makes (supervised streams, reliable remote exec, a tool that doesn't leak the secrets it handles, durable persistence, a trustworthy audit log). Several are **v1-blocking** and are scheduled into the v1 phases (§7.10), not deferred.

### ADR-0021 — The streaming & event-binding contract: subscriptions, backpressure, cancellation, reconnect-resync
**Context.** Logs, stats, the engine event stream, and the Phase 2 timeline all push from the headless core to the UI across the one Wails binding layer (`app/events.go`). §2.3 requires every goroutine to have an owner and forbids leaked streams, but the *contract* that makes that true — how a view subscribes and unsubscribes, what happens under backpressure, how a cancel crosses the boundary, how live state is reconciled after a reconnect — was implicit.
**Decision.** (1) **Subscriptions are correlated:** a frontend opens a stream through a bound method that returns a core-issued **correlation ID**; typed events arrive on `app/events.go` channels keyed by that ID; the view **unsubscribes on unmount**, which cancels the owning `context` in the core and tears down the underlying engine stream — no subscription outlives its view. (2) **Backpressure is explicit per stream:** logs and the event stream **coalesce and drop with a visible "stream fell behind" marker** rather than growing an unbounded buffer; stats are **fixed-cadence sampled** (§7.6) and so naturally rate-limited; nothing buffers without bound. (3) **Cancellation crosses by ID:** a bound `Cancel(id)` cancels the core context and tears down the SDK request and connection (the existing P4 no-leak gate). (4) **Reconnect means resync, not resume:** after a supervised transport reconnect (ADR-0005), live views **refetch authoritative state** (object lists, re-established stats subscriptions) rather than resuming a stale stream; the timeline marks the gap (ADR-0018); resync is debounced and bounded. (5) The streaming surface sits behind **one internal interface** so the Wails v2 event API — and a future v3 migration (the open question in §3) — touches only `app/events.go`, never the core.
**Consequences.** The robustness claim becomes testable beyond "no leak": backpressure and resync have defined, tested behavior. Cost: a small subscription registry in the binding layer and a resync path per live view.

### ADR-0022 — Interactive exec is a first-class terminal contract, not just argv
**Context.** §7.11.4 promises a reliable in-app terminal over SSH-tunnelled remotes — exactly the area competing tools handle badly. ADR-0004/§7.3 pin exec to argv via the API exec endpoints, but the terminal mechanics that *determine* reliability were unspecified; argv-without-a-terminal-contract is the easy 80%, and the differentiator lives in the other 20%.
**Decision.** The exec terminal uses the Engine API exec/attach endpoints with a **PTY when interactive** (`Tty: true`), **bidirectional**: stdin is streamed to the exec and stdout/stderr stream back over an ADR-0021 subscription. **Terminal resize is propagated** via the engine's resize endpoint on every UI resize event; **stdin half-close** and **detach** are handled explicitly; a non-interactive exec (`Tty: false`) keeps stdout and stderr separated for argv-style command runs. The session is a supervised stream (ADR-0021): closing the pane cancels the exec context and the attach connection. Over a reconnect an exec session is **dropped, not silently resumed**, and the operator is told.
**Consequences.** The headline differentiator is specified where it actually matters. Cost: PTY + resize handling and a tested teardown path; exec sessions do not survive a reconnect by design.

### ADR-0023 — Captured parameters are redacted/sealed: the tool never persists in cleartext the secrets it handles
**Context.** Operation capture and the audit log persist each operation's **resolved parameters** (ADR-0010, §7.7/§7.8), and the option builder governs `run`/`create`/`exec` options including `-e` / `--env` / `--env-file` and registry auth. A container env secret is therefore a "resolved parameter" that would otherwise be written to `operations.option_set` JSON and the audit `detail` **in cleartext**. ADR-0009's sealing covered connection secrets (SSH/TLS keys, passphrases, the data key) but not this capture path — a latent way for the tool to leak the very secrets it was used to set.
**Decision.** The option catalog (§7.5) marks options that carry secret material with a **`secret` flag** (env values, `--env-file` contents, registry credentials, secret build args). On capture, secret-flagged values are **recorded as present but redacted** (stable key, value `‹redacted›`) in `operations.option_set` and the audit `detail`, and **AEAD-sealed (ADR-0009) only if a value must be retained at all**; they never appear in persisted rows, the audit log, or slog output. The existing logging-redaction boundary (§2.4) is **extended to the persistence path**, with a test that a known env secret never appears in any persisted row or log line (mirroring the §2.4 logs test).
**Consequences.** Drydock cannot, through its own history or audit, leak a secret it was used to set. Cost: a `secret` classification per catalog option and a redaction step on the capture path; one more boundary test.

### ADR-0024 — SQLite durability: WAL mode, a real backup path, and downgrade refusal
**Context.** §7.7 described backup as a "file-copy of the data dir while quiesced." Under SQLite **WAL mode** (the right default for a responsive app doing concurrent reads), a bare copy of the main DB file omits uncheckpointed WAL/SHM and can be inconsistent; the journal mode was unstated. Forward-only migrations also left "an older app opens a newer DB" undefined.
**Decision.** The database runs in **WAL mode** with `synchronous=NORMAL`. **Backup uses the SQLite online backup API (or `VACUUM INTO`)** to produce a single consistent file — never a raw file copy, and the docs no longer instruct one. On open, the app checks `schema_migrations`: a DB **newer than the app refuses to open** with a clear typed error (`ERR_STORE_SCHEMA_NEWER`) rather than risk corrupting it; an older DB migrates forward as today.
**Consequences.** Backups are consistent and downgrades fail safe. Cost: a backup routine using the driver's backup support and a one-line version-guard on open.

### ADR-0025 — The audit chain is HMAC-keyed and truncation-aware, with honestly-stated guarantees
**Context.** ADR-0010's chain (`SHA256(prev_hash || canonical(entry))`) is tamper-*evident* against in-place edits — and is correctly never claimed as tamper-*proof*. But a plain hash chain is **recomputable** by anyone who can write the DB and reproduce the canonicalization, and **tail-truncation** is undetectable without an external anchor. For a security tool the bar should be both raised and documented.
**Decision.** The chain is **keyed**: `mac = HMAC-SHA256(K, prev_mac || canonical(entry))`, where **K is a per-install audit key in the OS keyring** (ADR-0009) — so recomputing the chain also requires the keyring secret, not merely DB write access. The **latest `(seq, mac)` is persisted as an external high-water mark** (keyring / a small separate file) so **tail truncation is detected** on verify. The verification indicator (§7.11.8) distinguishes **intact / in-place-tampered / truncated / key-unavailable**. The book states plainly what this defends against — accidental corruption, casual and in-place tampering, tail truncation — and what it does **not**: an attacker with simultaneous DB write **and** keyring access can still forge a consistent chain. This is a single-operator desktop trust model, stated as such.
**Consequences.** A materially stronger, honestly-scoped audit guarantee. Cost: an HMAC key in the keyring and a high-water-mark check on verify; with the key unavailable the chain still verifies structurally but is flagged **key-unavailable** rather than silently "intact."

---

## 7. Drydock — specification

### 7.1 Domain model (`internal/core/domain`)
- `Host{ ID, Name, Transport (local|ssh|tls), Endpoint, ObserveMode, EngineVersion, APIVersion, Status }`
- `Container{ ID, HostRef, Name, Image, State, Status, Ports, ComposeProject, Created }`
- `Image{ ID, HostRef, Repo, Tag, Size, Dangling, InUse }`
- `Volume{ Name, HostRef, Driver, Size, InUse, Mountpoint }`
- `Network{ ID, HostRef, Name, Driver, InUse }`
- `ComposeStack{ Project, HostRef, Services []Container, Status }`
- `Operation{ ID, HostRef, Kind, Target, OptionSet, Result, BytesReclaimed, StartedAt, EndedAt }`
- `ResourceSample{ HostRef, ContainerID, At, CPUPct, MemBytes, NetRx, NetTx, BlkRead, BlkWrite }` — rolling history
- `PruneImpact{ Kind, Objects []ObjectRef, ReclaimableBytes, NamedVolumes []VolumeRef }` (§7.4)
- `AuditEntry{ Seq, At, Action, HostRef, Subject, Detail, PrevHash, Hash }` (§7.8)

Validation lives with these types: a destructive op requires an explicit confirm flag; an op against an observe-mode host is rejected (§7.4, ADR-0013).

**Phase 2 additions** (introduced with §7.12; kept out of the v1 surface until §7.10 is green):
- `ComposePlan{ Project, HostRef, Services []ServiceChange, Networks []ResourceChange, Volumes []ResourceChange, Degraded bool, Destructive bool }` (§7.12.2, ADR-0016)
- `ServiceChange{ Service, Action (create|recreate|start|noop), Reasons []string, DropsAnonymousVolumes bool, InterruptsRunning bool }`; `ResourceChange{ Name, Kind (network|volume), Action (create|recreate|remove) }`
- `PortBinding{ HostRef, ContainerID, HostIP, HostPort, ContainerPort, Protocol, Reach (loopback|private|all_interfaces) }` and the aggregate `ExposureMap{ HostRef, Bindings []PortBinding }` (§7.12.3, ADR-0017)
- `TimelineEntry{ HostRef, At, Source (engine|audit), Kind, Subject, ExitCode *int, Detail }` — engine events interleaved with audit references, rolling retention (§7.12.4, ADR-0018)
- `ImageProvenance{ HostRef, ImageRef, RunningDigest, RegistryDigest, TagDrifted bool, Age, Untagged bool }` (§7.12.5, ADR-0019)
- `VolumeSnapshot{ ID, HostRef, Volume, Destination, SizeBytes, CreatedAt }` (§7.12.6, ADR-0020)

### 7.2 Host connections & transport (`internal/core/hosts` + `sshdialer`/`tlsdialer`)
- **SSH-first (ADR-0005):** a remote host is reached by dialing its Unix socket over an SSH connection using the operator's existing keys/agent. Connection setup is robust — explicit timeouts, health-check, reconnect-with-backoff, and clean teardown (no leaked tunnels), because this is exactly where existing tools are unreliable.
- **Local:** the local Unix socket directly.
- **mTLS-over-TCP:** supported for hosts already configured for it; the client key is a root-equivalent secret (ADR-0009).
- **Never** an unauthenticated TCP socket; if one is detected, Drydock warns prominently and treats the host as untrusted.
- **Add-host wizard:** pick transport, select an SSH key/agent identity, **test the connection** (and report engine/API version) before saving — turning the fiddly key/`DOCKER_HOST`/context dance into one guided flow.
- **Multi-host:** a host switcher; each host shows transport + trust + observe state. Observe-mode (ADR-0013) is set here.
- Connection profiles persist (§7.7); **private keys are never copied into Drydock's store** — only references.

### 7.3 Engine API integration (`internal/adapters/dockerengine`)
Per ADR-0003, the Docker Go SDK behind the `Engine` port:
- Containers: list/inspect, start/stop/restart/kill, remove, **exec**, **logs** (streamed tail + search), **stats** (streamed; sampled into `ResourceSample` history).
- **Exec terminal (ADR-0022):** interactive exec uses the API exec/attach endpoints with a **PTY** (`Tty: true`), bidirectional — stdin streamed in, stdout/stderr streamed back; **terminal resize is propagated** on every UI resize; stdin half-close and detach are explicit; non-interactive exec (`Tty: false`) keeps stdout/stderr separated for argv runs. Always argv, never a shell (ADR-0004). A session is a supervised stream; closing the pane cancels it; a reconnect drops it (not silently resumed) and tells the operator.
- Images/volumes/networks: list/inspect/remove, and prune via the impact path (§7.4).
- Compose: discover projects (by container labels), view as a unit, per-service status/logs, up/down.
- **Registry distribution-inspect** for image provenance (§7.12.5): the digest check runs **through the daemon** so it reflects the host's reachability and credentials (ADR-0019).
- `system df` mapped to the disk view (§7.6); the **event stream** drives live UI updates, restart-loop detection, and the Phase 2 timeline (§7.12.4) — scoped to `local` events (swarm-scope events are filtered on a swarm-manager host).
- **Streaming & subscription contract (ADR-0021):** every stream (logs, stats, events, exec) is opened by a bound method returning a core-issued **correlation ID**, delivered on typed `app/events.go` channels, and **torn down on view unmount / `Cancel(id)`** — the owning `context` is cancelled and the SDK stream closed (no leaks, §2.3). Backpressure is explicit per stream (logs/events coalesce-and-drop with a "fell behind" marker; stats are fixed-cadence). After a transport reconnect, live views **resync** (refetch authoritative state), they do not resume a stale stream.
- API version negotiated per host (ADR-0008); every response shape gets a typed mapping and a sanitized fixture under `testdata/api/`.

### 7.4 Destructive-operation safety & prune impact preview (the central safety property)
Grounded in the leading cause of Docker data loss — cleanup gone wrong.
- **Preview before delete.** Any prune/remove/`down -v` first computes a `PruneImpact`: the exact objects that would be removed and the reclaimable bytes, **broken out by category** (stopped containers, dangling vs unused images, **build cache** — the commonly-overlooked big consumer — and volumes). The operator sees precisely what disappears before confirming.
- **Volumes are never bulk-deleted.** Named volumes hold persistent data; the impact view lists each candidate volume with its size and in-use status, and **each must be individually confirmed**. There is no one-click "prune all volumes."
- **Explicit confirm + acknowledgement.** Destructive operations require a typed confirmation; the acknowledgement and the computed impact are written to the audit log (§7.8).
- **Observe-mode hosts refuse all mutation** in the core (ADR-0013).
- **In-use protection:** removing a running container, a volume in use, or a network with attachments surfaces the dependency and requires explicit override.

### 7.5 Operation options & impact warnings (`internal/core/options`)
Implements ADR-0011 — choose operation options safely, with explanations and combination-impact alerts.
- **Catalog format** (`catalogs/docker@<apiversion>.toml`, embedded): each option = `{ name, type, default, category, summary, description, risk (read|mutating|destructive), affects_data (bool), secret (bool), conflicts_with[], requires[], impacts[] }`. The catalog also pins the snapshot helper image by digest (§7.5 / ADR-0020).
- **Builder:** validates types/`conflicts_with`/`requires` and assembles API parameters (never a shell). Examples it governs: container `run`/`create` options, `exec` (user, working dir, tty), log filters, prune filters.
- **Secret handling on capture (ADR-0023):** `secret`-flagged values (`-e`/`--env`/`--env-file` contents, registry credentials, secret build args) are **never persisted in cleartext** — they are recorded as present-but-`‹redacted›` in `operations.option_set` and the audit `detail`, AEAD-sealed only if a value must be retained, and excluded from slog. A boundary test asserts a known env secret appears in no persisted row or log line.
- **Impact-rule engine:** declarative, pure, table-tested rules producing `warn` / `require_ack` / `block`. Examples:
  - prune with volumes selected → "named volumes hold persistent data; each will be confirmed individually." (routes to §7.4)
  - `rm -f` on a running container → "force-removes a running container; in-flight work is lost." (`require_ack`)
  - compose `down -v` → "removes the stack's volumes; persistent data is deleted." (`require_ack`)
  - `exec` as root / privileged → "runs as root inside the container." (`warn`)
  - any mutation on an observe-mode host → `block` (ADR-0013).
- **Risk badges:** every option and the resolved operation show `read` / `mutating` / `destructive`.

### 7.6 Disk & resource insight (`internal/core/engine` + `history`)
- **Disk view:** `system df` made legible — reclaimable space **by category** (images, containers, local volumes, **build cache**), each with a one-click route into the preview-and-confirm flow (§7.4). The build-cache line is first-class, since it is frequently the largest and most overlooked consumer.
- **Resource history:** container stats are sampled into a bounded rolling `ResourceSample` window so the operator can see a memory leak or a CPU spike *after* it happened, not just live — answering the "`docker stats` is ephemeral" complaint.
- **Restart-loop detection:** the event stream flags a container as looping on a configurable threshold (default: **≥ 3 restarts within 60 s**), surfaced on the dashboard with the last exit code; a container with an explicit `restart: always`/`unless-stopped` policy is evaluated against the same threshold but labelled as policy-driven so an intentional always-restart is not mistaken for a crash loop.

### 7.7 Persistence & history (`internal/core/store` + `sqlitestore`)
The SQLite schema (ADR-0007). Tables (forward-only migrations, embedded):
- `hosts` — id, name, transport, endpoint, observe_mode, key_ref, last_engine_version, last_api_version.
- `operations` — id, host_id, kind, target, option_set (JSON; `secret`-flagged values redacted before insert, ADR-0023), result, bytes_reclaimed, started_at, ended_at.
- `prune_impacts` — operation_id, category, object_count, reclaimable_bytes (the recorded preview that was confirmed).
- `resource_samples` — host_id, container_id, at, cpu_pct, mem_bytes, net_rx, net_tx, blk_read, blk_write (rolling retention).
- `sealed_values` — id, scope, nonce, sealed_bytes (AEAD) — for any sensitive saved value.
- `audit_log` — seq, at, action, host_id, subject, detail (JSON; secrets redacted), prev_mac, mac (HMAC-keyed, ADR-0025); the latest `(seq, mac)` high-water mark lives outside the table (keyring / small separate file) for truncation detection (§7.8).
- `schema_migrations` — version, applied_at.

Rules: all writes go through transactions; sensitive values sealed before insert (ADR-0009) and `secret`-flagged parameters redacted (ADR-0023); the `Store` port exposes intention-revealing queries (`HostsWithStatus`, `OperationsForHost`, `RecentResourceSamples`, `DestructiveOperations`) — the UI never composes SQL. Resource history has a configurable rolling retention. **Durability (ADR-0024):** the DB runs in **WAL mode** (`synchronous=NORMAL`); **backup uses the SQLite online backup API / `VACUUM INTO`** to produce one consistent file — never a raw file copy; on open, a DB whose `schema_migrations` version is **newer than the app refuses to open** (`ERR_STORE_SCHEMA_NEWER`) rather than risk corruption.

### 7.8 Capture & audit log (`internal/core/audit`)
- **Capture:** each operation's result (and, for prune, the exact `PruneImpact` that was confirmed and the bytes reclaimed) is persisted with the operation.
- **Audit log:** append-only and **HMAC-keyed-chained** (ADR-0025): `mac = HMAC-SHA256(K, prev_mac || canonical(entry))`, where `K` is a per-install audit key in the OS keyring — so the chain cannot be silently recomputed by DB write access alone. The latest `(seq, mac)` is persisted as an **external high-water mark** so tail-truncation is detectable. Recorded actions include host connect/disconnect, observe-mode changes, container lifecycle, **every destructive operation with its impact and acknowledgement**, exec, compose actions, and snapshot/restore. The chain is verifiable, exportable, and surfaced in the UI (§7.11.8) as **intact / in-place-tampered / truncated / key-unavailable**.
- **Stated guarantee:** this defends against accidental corruption, casual or in-place tampering, and tail truncation; it does **not** defend against an attacker holding **both** DB write access **and** the keyring key — a single-operator desktop trust model, stated plainly rather than oversold as tamper-proof.

### 7.9 Secrets & data-at-rest (`internal/adapters/keyring`)
Implements ADR-0009.
- SSH auth prefers the agent and existing on-disk keys; Drydock references, never copies, private keys.
- A per-install data key in the OS keyring seals any sensitive saved value; seal/open round-trip is unit-tested. A **separate per-install audit HMAC key** in the keyring keys the audit chain (ADR-0025).
- mTLS client keys are referenced by path; any passphrase that must be retained goes to the keyring, never the DB or config.
- **Captured parameters never leak secrets (ADR-0023):** `secret`-flagged option values (container env, `--env-file` contents, registry credentials, secret build args) are redacted/sealed before they reach `operations.option_set` or the audit `detail`; the logging-redaction boundary extends to the persistence path, with a test that a known env secret appears in no persisted row or log line.

### 7.10 Phased build plan (gates are commands)
Tagged integration tests run against a real daemon in CI — a Docker-in-Docker service for the local-engine gates and a containerized SSH-to-daemon sidecar for the transport gates — so "against a real local daemon / over SSH" is a reproducible CI step, not a manual one.
- **P0 — skeleton + charter scaffolding.** Repo layout, `.golangci.yml`, Taskfile, CI (incl. `govulncheck`), slog, XDG config, ADRs, governance files committed. *Gate:* `task lint test build` green on an app that opens a window and logs startup; `govulncheck` clean.
- **P1 — persistence foundation.** `Store` port + `sqlitestore` + WAL-mode migrations; **HMAC-keyed, truncation-aware** audit log; keyring `SecretStore` + per-install data key + per-install audit key + AEAD seal/open. *Gate:* migrations up + schema-version check against a real temp SQLite file; **a newer-than-app DB refuses to open (`ERR_STORE_SCHEMA_NEWER`)**; audit chain verify distinguishes intact / in-place-tampered / **truncated** / key-unavailable (ADR-0025); **backup via the SQLite backup API produces a consistent file under concurrent writes** (ADR-0024); keyring round-trip (tagged); AEAD seal/open round-trip.
- **P2 — local engine + Engine port (read-only).** Docker SDK adapter, API-version negotiation, list/inspect containers/images/volumes/networks, typed mappings, sanitized fixtures, mock-based unit tests. *Gate:* mapper fixture tests; tagged integration test lists objects against a real local daemon and asserts no leaked connections.
- **P3 — SSH transport + add-host wizard + multi-host + observe-mode.** `Dialer` port, `sshdialer`, robust connect/health/reconnect/teardown; observe-mode enforced in core. *Gate:* tagged integration test connects to a host over SSH, lists containers, disconnects, asserts no leaked tunnel; unit test that a mutating call on an observe-mode host returns `ErrObserveMode` before reaching the engine; unauthenticated-TCP detection warns.
- **P4 — container control + logs + stats + exec terminal + streaming contract.** Start/stop/restart/kill/remove (mutating, with confirm), streamed log tail + search, fixed-cadence stats into history, interactive **PTY exec** (resize, stdin half-close, detach — ADR-0022), and the **subscription/backpressure/cancellation/resync contract** (ADR-0021). *Gate:* exec uses argv (no shell) and a PTY with working resize (tagged); cancel-by-correlation-ID tears down the SDK request + connection (no leaked streams); a slow consumer triggers coalesce-and-drop with a "fell behind" marker, never an unbounded buffer; a simulated reconnect drives a **resync** (refetch), not a stale resume, and drops any exec session; a known secret never appears in logs **or in any persisted `option_set`/audit row** (capture-redaction test, ADR-0023).
- **P5 — prune impact preview + destructive-op safety.** `PruneImpact` calculator, per-category reclaimable, per-volume confirm, impact-rule engine, option builder/catalog (incl. the `secret` flag, ADR-0023). *Gate:* impact-calculator tests against `system df` fixtures (incl. build cache); test that volumes are never bulk-deleted (each requires confirm); impact-rule tests (`rm -f` running → require_ack; `down -v` → require_ack; observe-mode → block); a `secret`-flagged option value is redacted in the captured `option_set` and audit detail (test); destructive ops write impact + ack to the audit log.
- **P6 — disk & resource dashboard + restart-loop detection.** Disk view, rolling resource history, event-stream-driven updates. *Gate:* resource-history retention test; restart-loop detection unit test against an event fixture.
- **P7 — compose stacks.** Discover by labels, view as a unit, per-service logs/status, up/down (down -v gated). *Gate:* stack-grouping unit test against fixtures; `down -v` requires acknowledgement (test).
- **P8 — operation history & audit view + export.** History browser + queries, audit-chain verification view, export. *Gate:* query tests; audit chain verifies/export; export round-trip.
- **P9 — packaging, signing & release.** macOS sign+notarize, Linux AppImage + `.deb`, reproducible CI build, published checksums, semver tag + changelog (§9). *Gate:* CI produces a signed+notarized macOS artifact and Linux packages; checksums published; reproducible-build check matches across two runs.

### 7.11 UI/UX specification

This app is a GUI; the front end carries half the value. Governing principle: **the interface makes the consequential things visible — how root-equivalent access is travelling, and exactly what a delete will destroy — and the resolved operation, not raw flags, is what the operator confirms.** Visual direction follows the `frontend-design` skill; this section specifies structure, state, and behavior.

#### 7.11.1 Layout & navigation
A **host switcher** is always visible and always shows the active host's **transport (local/SSH/TLS), trust, and observe state** — the operator can never lose track of which machine, reached how, they're about to act on. Primary views per host: **Containers**, **Compose**, **Images**, **Volumes**, **Networks**, **Disk**, **History**, **Audit**. The window has defined minimum dimensions.

#### 7.11.2 Hosts & add-host wizard
- Host list with status, transport, engine/API version, and observe badge.
- **Add-host wizard:** choose transport → pick SSH key/agent identity → **test connection** (reports engine/API version) → save. Reduced-capability mode (ADR-0008) and unauthenticated-TCP warnings (ADR-0005) are shown here, not buried.
- Observe-mode toggle is an explicit, confirmed, audited action.

#### 7.11.3 Containers
- Sortable/filterable list: name, image, state, ports, compose project, uptime. State is conveyed with a non-colour cue + WCAG-AA contrast (§8.5).
- Row actions (start/stop/restart/remove) respect observe-mode (disabled with reason) and route destructive actions through confirm.
- Detail: inspect, env (redacted where sensitive), mounts, the **live stats** sparkline (from history), and a **logs** pane (streamed tail + search). The classic debugging flow — hop between a container's logs and stats — is one view, and switching containers is instant.

#### 7.11.4 Logs, stats & exec
- Logs: streamed tail, follow toggle, search/filter, copy/export; multi-container log view for tracing a request across services.
- Stats: live + the rolling history window so a past spike/leak is visible.
- **Exec:** a reliable in-app PTY terminal into a container via the API exec/attach endpoints (argv), working over SSH-tunnelled remotes — the thing the terminal tools do unreliably. **Window resize, stdin half-close, and detach are handled explicitly** (ADR-0022); a transport reconnect drops the session with a clear notice rather than silently resuming a dead one.

#### 7.11.5 Images, volumes, networks & the prune preview
- List/inspect/remove, with in-use indicators (dangling vs named vs in-use volumes are clearly distinguished).
- **Prune is preview-first (§7.4):** selecting prune opens an impact panel showing per-category reclaimable space (containers / dangling images / unused images / **build cache** / volumes) and the exact objects. **Volumes are listed individually with size + in-use status and confirmed one by one** — never bulk. The confirm step states totals; the result is recorded.

#### 7.11.6 Compose
- Stacks grouped by project: per-service status and logs, the stack viewed as a unit, up/down. `down -v` is gated behind an explicit acknowledgement (it deletes the stack's volumes).

#### 7.11.7 Disk dashboard
- `system df` made legible: a breakdown of used vs reclaimable by category with the **build-cache line first-class**, each category routing into the preview-and-confirm cleanup flow. Restart-loop and unhealthy-container callouts surface here.

#### 7.11.8 Operation history & audit view
- History: past operations per host with target, options, result, and (for prune) what was reclaimed; queries by host/kind/time; destructive-only filter.
- Audit: append-only entries with a visible **chain-verification indicator** distinguishing **intact / in-place-tampered / truncated / key-unavailable** (ADR-0025), exportable. The indicator states its guarantee honestly (it is tamper-*evident*, keyed and truncation-aware — not a claim of tamper-proofness).

#### 7.11.9 Empty, loading, error, and degraded states
Every view defines all four.
- **Empty:** "No hosts yet — add a host to begin"; per-view empties for no containers/images/etc.
- **Loading/streaming:** live data with a streaming indicator, not a blocking spinner; partial host lists render as they connect.
- **Error:** typed, human-readable messages from the error DTO (§2.2/§8.4) shown in context — never a raw Go string. A host that fails to connect shows why (auth, unreachable, version) with a remediation hint.
- **Degraded:** a below-minimum API version connects in labelled reduced-capability mode; an unauthenticated-TCP host is flagged untrusted; a disconnected host is clearly distinguished from an empty one; a stream that **fell behind** shows the coalesce-and-drop marker; a **reconnect** shows a resync indicator (refetching, not resuming — ADR-0021) and any open exec session is shown as dropped.

#### 7.11.10 Front-end structure (maps to §2.8)
- Generated typed bindings only; live stats/logs/events/exec arrive as typed events on correlation-ID-keyed channels, **subscribed on mount and unsubscribed on unmount** (which cancels the core context — ADR-0021). No subscription outlives its view; backpressure and reconnect-resync are handled in the streaming layer, not per component.
- Stores split by concern: `hosts` (registry + active + transport/observe state), `containers`, `logs`, `stats` (live + history window), `objects` (images/volumes/networks + prune impact), `options` (selection + impact), `history`, `audit` (entries + chain status). Runtime and view state are not commingled.
- Components are small and role-named: `HostSwitcher`, `AddHostWizard`, `ContainerList`, `ContainerDetail`, `LogPane`, `StatsSparkline`, `ExecTerminal`, `PruneImpactPanel`, `VolumeConfirmRow`, `ComposeStackView`, `DiskDashboard`, `OperationHistory`, `AuditLogView`. No monolithic `App.svelte`.

### 7.12 Phase 2 — post-1.0 roadmap

Phase 2 extends the two properties that define v1 rather than adding unrelated surface area: it pushes **preview-before-mutate** onto the `compose up` path (§7.12.2) and into volume destruction (§7.12.6), and it makes more of the host's root-equivalent reality **legible** — what is exposed to the network (§7.12.3), what actually happened and who did it (§7.12.4), and whether a running image is stale (§7.12.5). Decisions are fixed in ADR-0016..0020.

#### 7.12.1 Scope & ordering
Phase 2 is **not started until the v1 plan (§7.10) is green** — the §0 build discipline is unchanged: no working ahead, every phase ends on a command gate, each feature ships under the existing release rules (§9). Phase 2 introduces **no new bypass** of the two central safety properties; where it adds a destructive path (compose recreation, volume snapshot/restore) it reuses the §7.4 acknowledgement-and-audit machinery and the observe-mode block (ADR-0013). All read-only insight surfaces (exposure, timeline, provenance) state their own scope honestly and never imply enforcement they do not perform.

#### 7.12.2 Compose plan — preview-and-confirm on the apply path (`internal/core/engine` + `compose`)
Implements ADR-0016. The completion of the preview-before-mutate property: `up` stops being a black box.
- **Plan, then apply.** Selecting *up* on a stack first computes a `ComposePlan`: the desired project (parsed with the pinned `compose-go`) diffed against observed engine state, classifying each service create / recreate / start / no-op and each network and volume create / recreate / remove.
- **Surface the consequential elements.** A recreation that **interrupts a running container** or **drops an anonymous volume** is flagged; those are the elements that route through `require_ack` and into the audit log (§7.4/§7.8). A no-op plan says so plainly (this also answers the common "why did it recreate everything?" confusion).
- **Honest precision.** Convergence detection uses the **same Compose library version Drydock applies with**, never the `config --hash` CLI (which is documented to diverge from the stored label). When a service hash cannot be matched confidently, the plan sets `Degraded` and shows a **coarser, labelled diff** (image/tag/digest, published ports, env and mount presence) instead of asserting false precision.
- **Compose stays the authority.** The plan is a preview; Compose's own `up` performs the actual convergence. The plan is computed via the SDK / `compose-go` path, never a shell (ADR-0003/0004).
- **Source locality (ADR-0016).** A precise plan needs the project source, located from the running containers' `com.docker.compose.project.config_files` / `working_dir` labels. Drydock computes a **full plan when that source is accessible to it** (local host, or operator-supplied compose files) and does **not** read arbitrary files off a remote host (the transport is a socket dialer, not a filesystem). When the source is unreachable the plan enters an explicit **`source-unavailable` degraded state** — a labelled best-effort diff from labels and image/digest/port/env/mount inspection — never a false "no changes."

#### 7.12.3 Exposure map — what is reachable from outside (`internal/core/expose`)
Implements ADR-0017. Read-only insight, one layer up from the daemon-exposure posture (ADR-0005).
- **Compute reach per binding.** From each container's port bindings, build a per-host `ExposureMap` and classify every binding as **loopback** (`127.0.0.1`/`::1`), **private/LAN**, or **all-interfaces** (`0.0.0.0`/`::`/empty host IP).
- **Flag the dangerous default.** All-interfaces bindings are called out, and called out *prominently* on a host Drydock reached over a non-loopback transport (the binding is plausibly internet-reachable). The fleet view aggregates "what is exposed where" across hosts.
- **Host-network blind spot, named (ADR-0017).** A `network_mode: host` container publishes no bindings yet shares the host's namespace; it is listed explicitly as "host network — exposure not derivable from port bindings" rather than shown as exposing nothing (the dangerous false negative).
- **No enforcement, stated scope.** Drydock never edits a firewall or rebinds a port. The UI states that reach is classified **at the daemon layer only** — an upstream cloud security group or host firewall is invisible to it — so it reports the binding, not the full network path.

#### 7.12.4 Host timeline — engine events interleaved with the audit log (`internal/core/history` + `timeline`)
Implements ADR-0018. The explain-and-detect-drift surface.
- **Consume the event stream.** The per-host Engine event stream (already wired for live updates and restart-loop detection, §7.3/§7.6) is mapped to typed `TimelineEntry`s — including `die` with its `exitCode`, `oom`, and `health_status` transitions — and persisted with bounded rolling retention (§7.7).
- **Interleave, do not merge.** The timeline interleaves engine events with **references** to Drydock's own audit entries, correlated by host / container / time. The hash-chained **audit log is not merged into or weakened by the timeline** (ADR-0010); engine events are untrusted input and never enter the chain.
- **Host time, not desktop time.** Engine events carry the **host's** timestamp while audit entries carry the desktop's; correlation uses the host event time as authoritative for engine rows and surfaces any host-vs-desktop **clock skew** rather than silently misordering. The `Clock` port stays for test injection; it does not paper over skew.
- **Local scope only.** On a swarm-manager host the event stream also carries swarm-scoped events; the mapper **filters to `scope: local`** so the timeline reflects this host, not the cluster.
- **Best-effort, labelled.** The event stream can gap across a reconnect; the timeline marks such gaps and is presented as best-effort, explicitly distinct from the audit log's completeness guarantee for Drydock-authored actions. Restart-loop detection (§7.6) reads from this layer and the dashboard links into it.

#### 7.12.5 Image provenance & staleness (`internal/core/engine` + `images`)
Implements ADR-0019. A staleness signal without a scanner's weight or telemetry.
- **In-core provenance.** For each running container, surface image **age**, **tag-vs-digest drift** (running digest ≠ the tag's current registry digest), and **untagged / `:latest` ambiguity**, from the Engine's image data plus an **operator-initiated** registry digest check — no background phone-home (ADR-0006).
- **Checked through the daemon (ADR-0019).** The digest check runs via the Engine's **distribution-inspect** endpoint, so it reflects the **host's** registry reachability and credentials, not the desktop's — a registry reachable only from the host gives a correct answer, and a host that cannot reach the registry returns a clear "unreachable from host," not a desktop-side guess.
- **Credentials referenced, not copied.** Any auth is the operator's existing Docker credentials, encoded per-call, not stored (ADR-0009).
- **Scanning stays out.** Vulnerability scanning is explicitly **not** in core. If ever added it is a `Scanner` port + optional adapter, invoked explicitly, results never silently transmitted — and a separate ADR at that time.

#### 7.12.6 Volume snapshots — the undo for destruction (`internal/core/engine` + `snapshot`)
Implements ADR-0020. The escape hatch that completes §7.4.
- **Explicit, offered, never automatic.** At the confirm step for removing a data-bearing volume, Drydock *offers* a snapshot: a streamed `tar` of the volume to an operator-chosen destination, via a throwaway helper container mounting the volume **read-only** (Engine API, argv — ADR-0004). It never runs automatically and never blocks the deletion.
- **Pinned helper image (ADR-0020).** The helper is a **digest-pinned minimal image** recorded in the versioned catalog (§7.5); Drydock prefers an image already on the host and otherwise pulls the pinned digest as an audited, visible step, **failing closed on an air-gapped host** rather than hanging silently.
- **State the cost first.** Destination, size estimate, and expected duration are shown before it starts; the operation is cancellable (§2.3).
- **Observe-mode blocks it too.** Because a snapshot **starts a container** it is a mutation, so it is **blocked on observe-mode hosts** like every other mutation (ADR-0013) — no exception, and nothing lost, since deletion is already blocked there.
- **Audited; protect the output.** Snapshot and restore are audited operations (ADR-0010). The archive is plaintext **outside** Drydock's sealed store — the UI states plainly that the operator must protect the destination. Restore is a separate, deliberately-confirmed action, also observe-mode-blocked.

#### 7.12.7 Phase 2 build plan (gates are commands)
Continues §7.10's numbering and discipline. Begins only after **P9** is green.
- **P10 — compose plan.** `ComposePlan` diff engine over `compose-go`; recreation/anonymous-volume flagging; `require_ack` on destructive elements; degraded-mode fallback. *Gate:* plan-classification table tests against project + running-state fixtures, **including a config-hash mismatch case that forces `Degraded`** and a **`source-unavailable` case** that yields a labelled best-effort diff (never a false "no changes"); a recreation that drops an anonymous volume requires acknowledgement (test); plan is computed with no shell (review + `gosec`); destructive applies write impact + ack to the audit log.
- **P11 — exposure map.** Reach classification + per-host and fleet aggregation. *Gate:* classification table tests for `127.0.0.1` / `::1` / `0.0.0.0` / `::` / empty-host-IP / LAN; all-interfaces on a non-loopback-transport host is flagged (test); **a `network_mode: host` container is listed as "exposure not derivable," not as exposing nothing** (test); no code path edits a firewall or rebinds a port (review).
- **P12 — host timeline.** Engine-event ingestion → typed `TimelineEntry`; interleave with audit references; rolling retention. *Gate:* event-mapper fixture tests including `die`(`exitCode`)/`oom`/`health_status`; **swarm-scope events are filtered (test)**; **host-vs-desktop clock skew is surfaced, not silently reordered (test)**; retention test; **a test proving engine events never enter the audit chain** (separateness); reconnect gaps are marked.
- **P13 — image provenance & staleness.** Age, tag-vs-digest drift, `:latest`/untagged; operator-initiated registry check **via daemon distribution-inspect**. *Gate:* drift unit test (running-digest ≠ registry-digest → `TagDrifted`); the check runs through the host's engine, not the desktop network (test/mapper); a test that **no provenance/registry call happens without explicit operator action** (no background network); registry credentials referenced, not copied (review).
- **P14 — volume snapshot & restore.** Read-only-mount helper (digest-pinned image), streamed `tar`, size/time preview, cancellable; audited; restore separately confirmed. *Gate:* snapshot uses a read-only mount + argv (no shell); helper image is **digest-pinned and fails closed when unpullable** (air-gapped test); size/time shown before start; cancellation is clean (no orphaned helper container); snapshot and restore both write audit entries; **both are blocked on observe-mode** (test); snapshot is never a precondition for delete (test).

Release for each Phase 2 feature follows §9 (signed/notarized macOS, AppImage + `.deb`, published checksums, changelog) — no separate packaging phase.

#### 7.12.8 Phase 2 UI surfaces (extends §7.11)
- New per-host views: **Exposure** (the reach map, all-interfaces bindings surfaced first) and **Timeline** (engine events + audit references, with restart-loop and `oom`/`die`/`health` callouts linking from the Disk dashboard, §7.11.7).
- Compose (§7.11.6) gains a **plan step**: selecting *up* opens a `ComposePlanPanel` showing per-service create/recreate/start/no-op and per-resource changes, destructive elements gated, `Degraded` plans clearly labelled — the operator confirms the plan, not raw `up`.
- Images (§7.11.5) gains an `ImageProvenanceBadge` (stale / drifted / `:latest`) with an explicit "check registry" action.
- The volume confirm flow (`VolumeConfirmRow`, §7.11.5) gains an **offered** `VolumeSnapshotDialog` stating destination, size, and duration before any snapshot runs.
- New components: `ComposePlanPanel`, `ExposureMap`, `HostTimeline`, `ImageProvenanceBadge`, `VolumeSnapshotDialog`; new stores `exposure`, `timeline`, `provenance` (runtime/view state kept distinct, §2.8). Every new view defines its empty / loading / error / degraded states (§7.11.9) — the timeline's "stream gapped" and the plan's `Degraded` are first-class degraded states.

---

## 8. Security & threat model

A tool that holds root-equivalent access to one or more hosts must reason about its own security explicitly. Mitigations live in the ADRs and spec and are cross-referenced.

### 8.1 Assets
The integrity and confidentiality of every connected host (the Docker API is root-equivalent); the operator's SSH keys and any mTLS client keys; the per-install data key (OS keyring); host connection profiles; the integrity of the audit log; the operator's container data and volumes.

### 8.2 Trust boundaries
The webview ↔ Go bridge (local IPC, not a network port); Go ↔ SSH connection ↔ remote daemon socket; Go ↔ local daemon socket; the SDK ↔ engine (per-host, version-negotiated); the app ↔ OS keyring and filesystem; the app ↔ the SSH agent/keys it references.

### 8.3 Abuse cases & mitigations
| Threat | Mitigation |
|---|---|
| Pushing the operator toward exposing the root-equivalent daemon | SSH-first by design; never opens/encourages an unauthenticated TCP socket; warns and marks untrusted if one is detected (ADR-0005). |
| Command injection via container/exec/option input | Engine API + argv exec; options validated against a typed catalog; no shell anywhere (ADR-0003/0004, `gosec`). |
| Accidental data loss via prune/remove | Preview-and-confirm with per-category impact; volumes never bulk-deleted; in-use protection; observe-mode (ADR-0011/0013, §7.4). |
| Acting on the wrong (e.g. production) host | Always-visible host + transport + observe indicator; observe-mode rejects mutation in the core (§7.11.1, ADR-0013). |
| Credential leakage via logs/DB/export | SSH keys referenced not copied; secrets in keyring; sensitive values AEAD-sealed; `redact()` at the logging boundary (ADR-0009, §2.4). |
| The tool persisting, in cleartext, a secret it was used to set | `secret`-flagged option values (container env, `--env-file`, registry creds, secret build args) are redacted/sealed before reaching `operations.option_set`, the audit `detail`, or slog; tested on the persistence boundary (ADR-0023, §7.5/§7.9). |
| Audit-log tampering, recompute, or tail-truncation | **HMAC-keyed** chain (key in the keyring, so DB-write alone can't recompute it) + an **external high-water mark** for truncation detection; the UI distinguishes intact / in-place-tampered / truncated / key-unavailable; guarantee stated honestly, not as tamper-proof (ADR-0025, §7.8). |
| Leaked SSH tunnels / open streams | Supervised connection lifecycle; no-leak integration tests (§2.3, §7.2). |
| Vulnerable Go dependency | `govulncheck` in CI; pinned `go.sum` (§2.6). |
| Sensitive infrastructure data committed to the repo | `testdata` sanitization rule + pre-commit/test scan (§2.10). |
| Silent container recreation / volume loss via `compose up` | Plan-first preview; recreations that interrupt a running container or drop an anonymous volume are flagged and require acknowledgement; Compose stays authoritative; degraded plans are labelled, not faked (ADR-0016, §7.12.2). |
| Service unintentionally exposed to all interfaces / the public internet | Exposure map classifies loopback vs all-interfaces and surfaces all-interfaces bindings prominently on non-loopback-transport hosts; read-only insight with stated daemon-layer scope, no false enforcement claim (ADR-0017, §7.12.3). |
| Spoofed/poisoned engine events corrupting the trusted record | Engine events are untrusted input — typed-mapped, fixture-tested, and never written into the hash-chained audit log; the timeline is separate and labelled best-effort (ADR-0018, §7.12.4). |
| Stale/drifted image silently in production | Provenance surfaces age and tag-vs-digest drift; the digest check runs **through the daemon** (host's view/creds), is operator-initiated only with no background phone-home, and credentials are referenced not copied (ADR-0019/0006, §7.12.5). |
| Volume snapshot leaking data, running on a read-only host, or pulling an unverified helper | Snapshot is explicit and audited, destination operator-chosen, output stated as plaintext outside the sealed store; the helper is a **digest-pinned** image that fails closed when unpullable; because it starts a container it is **observe-mode-blocked** like restore; never automatic, never a delete precondition (ADR-0020, §7.12.6). |
| Inconsistent backup or downgrade corruption of the local DB | WAL mode with a **consistent backup via the SQLite backup API** (never a raw file copy); a newer-than-app DB **refuses to open** rather than risk corruption (ADR-0024, §7.7). |

### 8.4 Error-code catalog
The typed error DTO (§2.2) draws `code` from a single enumerated catalog (typed constants), e.g. `ERR_HOST_UNREACHABLE`, `ERR_HOST_AUTH`, `ERR_API_VERSION_UNSUPPORTED`, `ERR_OBSERVE_MODE`, `ERR_INSECURE_TRANSPORT`, `ERR_OPTION_CONFLICT`, `ERR_DESTRUCTIVE_NOT_CONFIRMED`, `ERR_OBJECT_IN_USE`, `ERR_EXEC_FAILED`, `ERR_STORE_MIGRATION`, `ERR_SECRET_UNAVAILABLE`, `ERR_AUDIT_CHAIN_BROKEN`. Each maps to a stable UI message and a `retryable` flag. The frontend switches on `code`, never on message text. The gap-review refinements (ADR-0021..0025) add `ERR_STORE_SCHEMA_NEWER` (a newer DB; non-retryable), `ERR_BACKUP_FAILED`, `ERR_AUDIT_KEY_UNAVAILABLE`, and `ERR_AUDIT_TRUNCATED`. Phase 2 (§7.12) adds `ERR_COMPOSE_PLAN_FAILED`, `ERR_COMPOSE_SOURCE_UNAVAILABLE` (informational — drives the degraded plan, not a failure), `ERR_COMPOSE_RECREATE_NOT_CONFIRMED`, `ERR_EVENT_STREAM_INTERRUPTED` (non-fatal, retryable — the timeline marks the gap), `ERR_REGISTRY_UNREACHABLE_FROM_HOST`, `ERR_SNAPSHOT_HELPER_UNAVAILABLE` (air-gapped, fail-closed), `ERR_SNAPSHOT_FAILED`, and `ERR_SNAPSHOT_DEST_UNWRITABLE`.

### 8.5 Accessibility & performance targets
- **Accessibility:** keyboard operability for core flows (host switching, container actions, the exec terminal); focus management on view changes; ARIA labels; container/host state conveyed by label/shape in addition to colour, with WCAG-AA contrast — meaning is never colour-only.
- **Performance:** container/image lists and the log pane are virtualized and stay responsive at fleet scale — target ≥5,000 containers across hosts and high-rate log streams without UI stall; live stat/log updates are batched/coalesced so a chatty container doesn't thrash rendering.

---

## 9. Distribution, signing & release

Implements ADR-0012; for a tool with root-equivalent reach, trustworthy distribution is essential.
- **macOS:** sign with a Developer ID, hardened runtime with minimal entitlements, notarize and staple. Installs and launches without Gatekeeper friction.
- **Linux:** ship a versioned **AppImage** and a **`.deb`** with a desktop entry. Note the system-tray caveat (freedesktop `StatusNotifierItem`; GNOME needs an extension) if a tray presence is added.
- **Reproducible builds:** build in CI from pinned Go and frontend toolchains; a reproducibility check compares artifacts across two runs. Optionally publish an SBOM.
- **Releases:** signed semver git tags, `CHANGELOG.md` (keepachangelog), and **published SHA-256 checksums** for every artifact.
- **Updates:** operator-initiated and integrity-checked; no silent auto-update (consistent with ADR-0006).

---

## 10. Legal & project governance

> This section is engineering guidance, not legal advice; confirm licensing specifics for the actual jurisdiction.

- **Licensing.** Choose a license for the app — **Apache-2.0** is recommended (permissive, explicit patent grant; the Docker Go SDK is Apache-2.0, so this aligns cleanly); MIT is the lighter alternative. Ship a `NOTICE`/`THIRD-PARTY-LICENSES` file with the SDK's and other dependencies' notices.
- **Vulnerability disclosure.** `SECURITY.md` defines how to report a vulnerability *in Drydock itself* — important for a tool with root-equivalent access: contact channel, scope, and response expectations.
- **Governance files** (also in §5): `LICENSE`, `NOTICE`, `SECURITY.md`, `CONTRIBUTING.md` (§11), `CHANGELOG.md`, `README.md` (what it is, the SSH-first/no-exposed-socket posture, verified install steps, quick start, the no-telemetry statement), plus issue/PR templates and optionally `CODEOWNERS`.

---

## 11. `CONTRIBUTING.md` template (commit to the repo)

The operating rules live in a standard `CONTRIBUTING.md` so the repository reads as a normally-governed project. Anyone (or anything) implementing against it follows the same standards; tooling that wants these rules loaded can be pointed at this file and `PROJECT-BOOK.md` directly.

```markdown
# Contributing to Drydock

`PROJECT-BOOK.md` is the specification and these standards are binding. Work in
the phases defined in PROJECT-BOOK.md §7.10, in order; a phase is not started
until the previous phase's gate is green.

## Before writing code
- Read PROJECT-BOOK.md §2 (Engineering Charter) and §7 (specification).
- Two properties have NO bypass: remote access is SSH-first and we never expose
  the daemon socket (§7.2/ADR-0005); destruction is preview-and-confirm and
  volumes are never bulk-deleted (§7.4/ADR-0011). Do not weaken either.

## Every change
- context.Context is the first parameter of any I/O / connection / API call.
- Errors wrapped with %w + context; typed sentinels for branched-on errors
  (e.g. ErrObserveMode); error codes from the §8.4 catalog; no string matching
  on error text; no panic outside main/tests.
- Structured logging via slog; secrets are never logged (use redact()). The
  operational log is not the audit log.
- No global mutable state; dependencies constructed and injected in
  cmd/drydock/main.go.
- Engine access via the Docker SDK behind the Engine port; container exec and
  any subprocess use argv. Never a shell.
- SSH private keys are referenced, never copied into our store; secrets live in
  the keyring; sensitive saved values are AEAD-sealed before disk.
- Mutations are rejected in the core for observe-mode hosts.
- No wails import outside app/ and shell/ (enforced by depguard).
- No authorship/tooling fingerprints in any file, commit, or metadata.

## Definition of a finished phase
Run and record the output of:
    task lint               # gofumpt + golangci-lint + go vet + govulncheck
    task test               # go test ./... green on a bare machine (no daemon)
    task test:integration   # tagged; runs against a real daemon/keyring if present
    task build              # runnable app for darwin + linux
A phase is not done until its gate (PROJECT-BOOK.md §7.10) passes. "It runs" is
not the gate.

## Not acceptable
- Placeholder packages, stub tests that assert true, or empty TODOs.
- Business logic in app/ or main.go; SQL composed outside the store adapter.
- A dependency added without pinning it and recording why.
- Any code path that exposes/encourages exposing the daemon socket, bulk-deletes
  volumes, or lets a mutation reach an observe-mode host.
- Real infrastructure data (hosts, IPs, private image refs) committed anywhere.

## When the book is wrong
Open the disagreement as a proposed ADR and amend the book. Do not diverge silently.
```

---

## 12. Definition of Done — review checklist

A change/phase is done only when **all** hold. Maps 1:1 to §2 and the security model.

**Boundaries**
- [ ] No `wails` import outside `app/` and `shell/` (depguard green).
- [ ] `go test ./internal/core/...` passes with no daemon, no webview.
- [ ] Domain core has no knowledge of SDK types, SSH, SQL, or the keyring.

**Errors & lifecycle**
- [ ] No `panic` outside `main`/tests. Errors wrapped with `%w` + context.
- [ ] Branched-on errors are typed (incl. `ErrObserveMode`); error codes from the §8.4 catalog; no `err.Error()` string matching.
- [ ] `context` first param on all I/O; cancellation reaches the SDK request and the connection, including a frontend `Cancel(id)` by correlation ID (ADR-0021).
- [ ] No goroutine without a defined owner/exit; no subscription outlives its view (unsubscribe-on-unmount). No leaked SSH tunnels or open streams after disconnect/quit (integration-tested).
- [ ] Streams have explicit backpressure (coalesce-and-drop with a marker, not unbounded buffers); a reconnect drives resync (refetch), not a stale resume (ADR-0021, tested).
- [ ] No package-level mutable state.

**Transport & host safety (a central safety property)**
- [ ] Remote access is SSH-first; the app never opens/encourages an unauthenticated TCP socket; insecure transport is detected and flagged.
- [ ] Observe-mode hosts reject all mutation in the core (not just hidden in the UI), tested.
- [ ] The active host's transport/trust/observe state is always visible.

**Destruction safety (a central safety property)**
- [ ] Prune/remove/down compute and show a per-category impact before execution.
- [ ] Named volumes are never bulk-deleted; each is individually confirmed (tested).
- [ ] Destructive ops require confirm + acknowledgement, recorded with their impact in the audit log.
- [ ] In-use objects surface their dependency and require explicit override.

**Options & impact**
- [ ] Operation options come from the catalog + builder; API params assembled, never a shell.
- [ ] Catalog references only real options for the negotiated API version (drift test where feasible).
- [ ] Impact rules tested: `rm -f` running → require_ack; `down -v` → require_ack; observe-mode → block.

**Data, history & audit**
- [ ] DB in WAL mode; migrations run forward against a real SQLite file; a newer-than-app DB refuses to open (`ERR_STORE_SCHEMA_NEWER`); backup uses the SQLite backup API and is consistent under concurrent writes (ADR-0024, tested).
- [ ] Sensitive values AEAD-sealed before insert; data key and audit HMAC key only in the keyring (round-trip tested); SSH private keys never stored.
- [ ] `secret`-flagged parameters are redacted/sealed before persistence — a known env secret appears in no `option_set`/audit row or log line (ADR-0023, tested).
- [ ] Each mutating operation persists target/host/params/result; prune persists its confirmed impact.
- [ ] Audit log is append-only and **HMAC-keyed-chained** with an external truncation high-water mark; verify distinguishes intact / in-place-tampered / truncated / key-unavailable (ADR-0025, tested).

**Tests**
- [ ] Tests assert behavior; no coverage theater.
- [ ] Mappers tested table-driven against sanitized API fixtures.
- [ ] Integration tests build-tagged; skip cleanly when daemon/keyring absent.

**Observability & security**
- [ ] slog only; no stray `fmt.Println`/`log.Printf`; operational log distinct from audit log.
- [ ] A redaction test proves SSH/TLS secrets and `secret`-flagged parameters never appear in logs or persisted rows.
- [ ] Container exec and any subprocess spawned argv-style (gosec green); no shell.
- [ ] `govulncheck` clean.
- [ ] Threat-model mitigations for touched areas (§8.3) hold.

**Distribution**
- [ ] Release artifacts: macOS signed+notarized, Linux AppImage+.deb, checksums published, reproducible-build check passes.

**Tooling & hygiene**
- [ ] `gofumpt`, `golangci-lint` (curated), `go vet`, `tsc --noEmit` clean in CI.
- [ ] Every exported symbol has a godoc comment.
- [ ] No commented-out code; no bare TODO/FIXME; no authorship/tooling fingerprints.
- [ ] Conventional Commits; every commit builds.
- [ ] Relevant ADR written/updated.

**Frontend**
- [ ] Frontend calls generated typed bindings only; event payloads typed/validated.
- [ ] State in defined stores (§7.11.10); components small/role-named; no unjustified `any`.
- [ ] Host switcher always shows transport/trust/observe state.
- [ ] Prune is preview-first with per-category reclaimable and per-volume confirmation.
- [ ] Reliable in-app PTY exec works over SSH-tunnelled remotes: resize propagates, stdin half-close/detach handled, a reconnect drops the session cleanly (ADR-0022).
- [ ] Container/host state conveyed with non-colour cue + WCAG-AA contrast; lists and logs virtualized.
- [ ] History and Audit views exist; audit view shows chain-verification status.
- [ ] Every view defines empty / loading / error / degraded states; errors render from the typed DTO.

**Phase 2 (apply only when building §7.12; P9 must be green first)**
- [ ] Compose `up` is plan-first; recreations that interrupt a running container or drop an anonymous volume require acknowledgement and are audited; a config-hash mismatch forces a labelled `Degraded` plan and an unreachable source forces a labelled `source-unavailable` plan (never false precision or a false "no changes"); plan computed via `compose-go`, no shell.
- [ ] Exposure reach is classified (loopback / private / all-interfaces); all-interfaces on a non-loopback-transport host is flagged; `network_mode: host` is listed as "not derivable," not as exposing nothing; no code path edits a firewall or rebinds a port; UI states its daemon-layer scope.
- [ ] Engine events are typed-mapped from fixtures, filtered to `scope: local`, and **never** enter the audit chain (tested); host-vs-desktop clock skew is surfaced; the timeline marks reconnect gaps and is labelled best-effort.
- [ ] Image provenance/registry checks happen only on explicit operator action (no background network, tested), run **through the daemon** (host's view/creds), and never copy credentials.
- [ ] Volume snapshot uses a read-only mount + argv with a digest-pinned helper that fails closed when unpullable; states size/time first; is cancellable with no orphaned helper; is audited; is never automatic and never blocks delete; **snapshot and restore are both observe-mode-blocked** and restore is separately confirmed.

---

## 13. Getting started
1. Execute **P0**; record the green gate.
2. Write ADR-0001..0025 into `docs/adr/` and `CONTRIBUTING.md` (from §11) as the opening commits, alongside `LICENSE`, `NOTICE`, `SECURITY.md`, `CHANGELOG.md`, and `README.md`. (ADR-0016..0020 fix the Phase 2 decisions up front, with their *work* deferred to §7.12; ADR-0021..0025 are gap-review refinements to v1 mechanics and are scheduled **into** the v1 phases, not deferred — streaming/exec in P4, secret-capture in P5, durability and the keyed audit chain in P1.)
3. Build the **persistence foundation (P1)**, then the **local read-only engine (P2)**, before adding remote transport or any mutating operation.
4. Proceed phase by phase. Do not skip gates.
5. **Phase 2 (P10–P14, §7.12) starts only after P9 is green** — the v1 build is shipped and solid before the apply-path safety, exposure, timeline, provenance, and snapshot features begin.
