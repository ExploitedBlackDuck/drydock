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
- Containers: list/inspect, start/stop/restart/kill, remove, **exec** (argv, via the API exec endpoints), **logs** (streamed tail + search), **stats** (streamed; sampled into `ResourceSample` history).
- Images/volumes/networks: list/inspect/remove, and prune via the impact path (§7.4).
- Compose: discover projects (by container labels), view as a unit, per-service status/logs, up/down.
- `system df` mapped to the disk view (§7.6); the **event stream** drives live UI updates and restart-loop detection.
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
- **Catalog format** (`catalogs/docker@<apiversion>.toml`, embedded): each option = `{ name, type, default, category, summary, description, risk (read|mutating|destructive), affects_data (bool), conflicts_with[], requires[], impacts[] }`.
- **Builder:** validates types/`conflicts_with`/`requires` and assembles API parameters (never a shell). Examples it governs: container `run`/`create` options, `exec` (user, working dir, tty), log filters, prune filters.
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
- **Restart-loop detection:** the event stream flags containers restarting repeatedly, surfaced on the dashboard.

### 7.7 Persistence & history (`internal/core/store` + `sqlitestore`)
The SQLite schema (ADR-0007). Tables (forward-only migrations, embedded):
- `hosts` — id, name, transport, endpoint, observe_mode, key_ref, last_engine_version, last_api_version.
- `operations` — id, host_id, kind, target, option_set (JSON), result, bytes_reclaimed, started_at, ended_at.
- `prune_impacts` — operation_id, category, object_count, reclaimable_bytes (the recorded preview that was confirmed).
- `resource_samples` — host_id, container_id, at, cpu_pct, mem_bytes, net_rx, net_tx, blk_read, blk_write (rolling retention).
- `sealed_values` — id, scope, nonce, sealed_bytes (AEAD) — for any sensitive saved value.
- `audit_log` — seq, at, action, host_id, subject, detail (JSON), prev_hash, hash (§7.8).
- `schema_migrations` — version, applied_at.

Rules: all writes go through transactions; sensitive values sealed before insert (ADR-0009); the `Store` port exposes intention-revealing queries (`HostsWithStatus`, `OperationsForHost`, `RecentResourceSamples`, `DestructiveOperations`) — the UI never composes SQL. Resource history has a configurable rolling retention. Backup is a documented file-copy of the data dir while the app is quiesced.

### 7.8 Capture & audit log (`internal/core/audit`)
- **Capture:** each operation's result (and, for prune, the exact `PruneImpact` that was confirmed and the bytes reclaimed) is persisted with the operation.
- **Audit log:** append-only, hash-chained (`hash = SHA256(prev_hash || canonical(entry))`). Recorded actions include host connect/disconnect, observe-mode changes, container lifecycle, **every destructive operation with its impact and acknowledgement**, exec, and compose actions. The chain is verifiable and surfaced in the UI (§7.11.8), and exportable.

### 7.9 Secrets & data-at-rest (`internal/adapters/keyring`)
Implements ADR-0009.
- SSH auth prefers the agent and existing on-disk keys; Drydock references, never copies, private keys.
- A per-install data key in the OS keyring seals any sensitive saved value; seal/open round-trip is unit-tested.
- mTLS client keys are referenced by path; any passphrase that must be retained goes to the keyring, never the DB or config.

### 7.10 Phased build plan (gates are commands)
- **P0 — skeleton + charter scaffolding.** Repo layout, `.golangci.yml`, Taskfile, CI (incl. `govulncheck`), slog, XDG config, ADRs, governance files committed. *Gate:* `task lint test build` green on an app that opens a window and logs startup; `govulncheck` clean.
- **P1 — persistence foundation.** `Store` port + `sqlitestore` + migrations; append-only hash-chained audit log; keyring `SecretStore` + per-install data key + AEAD seal/open. *Gate:* migrations up + schema-version check against a real temp SQLite file; audit chain verify (tamper detection); keyring round-trip (tagged); AEAD seal/open round-trip.
- **P2 — local engine + Engine port (read-only).** Docker SDK adapter, API-version negotiation, list/inspect containers/images/volumes/networks, typed mappings, sanitized fixtures, mock-based unit tests. *Gate:* mapper fixture tests; tagged integration test lists objects against a real local daemon and asserts no leaked connections.
- **P3 — SSH transport + add-host wizard + multi-host + observe-mode.** `Dialer` port, `sshdialer`, robust connect/health/reconnect/teardown; observe-mode enforced in core. *Gate:* tagged integration test connects to a host over SSH, lists containers, disconnects, asserts no leaked tunnel; unit test that a mutating call on an observe-mode host returns `ErrObserveMode` before reaching the engine; unauthenticated-TCP detection warns.
- **P4 — container control + logs + stats + exec.** Start/stop/restart/kill/remove (mutating, with confirm), streamed log tail + search, streamed stats sampled into history, API-exec (argv). *Gate:* cancel propagates to SDK request + connection (no leaked streams); exec uses argv (no shell); a known secret never appears in logs (redaction test).
- **P5 — prune impact preview + destructive-op safety.** `PruneImpact` calculator, per-category reclaimable, per-volume confirm, impact-rule engine, option builder/catalog. *Gate:* impact-calculator tests against `system df` fixtures (incl. build cache); test that volumes are never bulk-deleted (each requires confirm); impact-rule tests (`rm -f` running → require_ack; `down -v` → require_ack; observe-mode → block); destructive ops write impact + ack to the audit log.
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
- **Exec:** a reliable in-app terminal into a container via the API exec endpoints (argv), working over SSH-tunnelled remotes — the thing the terminal tools do unreliably.

#### 7.11.5 Images, volumes, networks & the prune preview
- List/inspect/remove, with in-use indicators (dangling vs named vs in-use volumes are clearly distinguished).
- **Prune is preview-first (§7.4):** selecting prune opens an impact panel showing per-category reclaimable space (containers / dangling images / unused images / **build cache** / volumes) and the exact objects. **Volumes are listed individually with size + in-use status and confirmed one by one** — never bulk. The confirm step states totals; the result is recorded.

#### 7.11.6 Compose
- Stacks grouped by project: per-service status and logs, the stack viewed as a unit, up/down. `down -v` is gated behind an explicit acknowledgement (it deletes the stack's volumes).

#### 7.11.7 Disk dashboard
- `system df` made legible: a breakdown of used vs reclaimable by category with the **build-cache line first-class**, each category routing into the preview-and-confirm cleanup flow. Restart-loop and unhealthy-container callouts surface here.

#### 7.11.8 Operation history & audit view
- History: past operations per host with target, options, result, and (for prune) what was reclaimed; queries by host/kind/time; destructive-only filter.
- Audit: append-only entries with a visible **chain-verification indicator** (green = intact, red = tampering detected), exportable.

#### 7.11.9 Empty, loading, error, and degraded states
Every view defines all four.
- **Empty:** "No hosts yet — add a host to begin"; per-view empties for no containers/images/etc.
- **Loading/streaming:** live data with a streaming indicator, not a blocking spinner; partial host lists render as they connect.
- **Error:** typed, human-readable messages from the error DTO (§2.2/§8.4) shown in context — never a raw Go string. A host that fails to connect shows why (auth, unreachable, version) with a remediation hint.
- **Degraded:** a below-minimum API version connects in labelled reduced-capability mode; an unauthenticated-TCP host is flagged untrusted; a disconnected host is clearly distinguished from an empty one.

#### 7.11.10 Front-end structure (maps to §2.8)
- Generated typed bindings only; live stats/logs/events arrive as typed events validated at the boundary.
- Stores split by concern: `hosts` (registry + active + transport/observe state), `containers`, `logs`, `stats` (live + history window), `objects` (images/volumes/networks + prune impact), `options` (selection + impact), `history`, `audit` (entries + chain status). Runtime and view state are not commingled.
- Components are small and role-named: `HostSwitcher`, `AddHostWizard`, `ContainerList`, `ContainerDetail`, `LogPane`, `StatsSparkline`, `ExecTerminal`, `PruneImpactPanel`, `VolumeConfirmRow`, `ComposeStackView`, `DiskDashboard`, `OperationHistory`, `AuditLogView`. No monolithic `App.svelte`.

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
| Audit-log tampering | Append-only, hash-chained entries; verification surfaced in the UI (ADR-0010, §7.8). |
| Leaked SSH tunnels / open streams | Supervised connection lifecycle; no-leak integration tests (§2.3, §7.2). |
| Vulnerable Go dependency | `govulncheck` in CI; pinned `go.sum` (§2.6). |
| Sensitive infrastructure data committed to the repo | `testdata` sanitization rule + pre-commit/test scan (§2.10). |

### 8.4 Error-code catalog
The typed error DTO (§2.2) draws `code` from a single enumerated catalog (typed constants), e.g. `ERR_HOST_UNREACHABLE`, `ERR_HOST_AUTH`, `ERR_API_VERSION_UNSUPPORTED`, `ERR_OBSERVE_MODE`, `ERR_INSECURE_TRANSPORT`, `ERR_OPTION_CONFLICT`, `ERR_DESTRUCTIVE_NOT_CONFIRMED`, `ERR_OBJECT_IN_USE`, `ERR_EXEC_FAILED`, `ERR_STORE_MIGRATION`, `ERR_SECRET_UNAVAILABLE`, `ERR_AUDIT_CHAIN_BROKEN`. Each maps to a stable UI message and a `retryable` flag. The frontend switches on `code`, never on message text.

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
- [ ] `context` first param on all I/O; cancellation reaches the SDK request and the connection.
- [ ] No goroutine without a defined owner/exit. No leaked SSH tunnels or open streams after disconnect/quit (integration-tested).
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
- [ ] Migrations run forward against a real SQLite file; schema-version checked.
- [ ] Sensitive values AEAD-sealed before insert; data key only in the keyring (round-trip tested); SSH private keys never stored.
- [ ] Each mutating operation persists target/host/params/result; prune persists its confirmed impact.
- [ ] Audit log is append-only and hash-chained; tamper detection tested.

**Tests**
- [ ] Tests assert behavior; no coverage theater.
- [ ] Mappers tested table-driven against sanitized API fixtures.
- [ ] Integration tests build-tagged; skip cleanly when daemon/keyring absent.

**Observability & security**
- [ ] slog only; no stray `fmt.Println`/`log.Printf`; operational log distinct from audit log.
- [ ] A redaction test proves SSH/TLS secrets never appear in logs.
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
- [ ] Reliable in-app exec works over SSH-tunnelled remotes.
- [ ] Container/host state conveyed with non-colour cue + WCAG-AA contrast; lists and logs virtualized.
- [ ] History and Audit views exist; audit view shows chain-verification status.
- [ ] Every view defines empty / loading / error / degraded states; errors render from the typed DTO.

---

## 13. Getting started
1. Execute **P0**; record the green gate.
2. Write ADR-0001..0014 into `docs/adr/` and `CONTRIBUTING.md` (from §11) as the opening commits, alongside `LICENSE`, `NOTICE`, `SECURITY.md`, `CHANGELOG.md`, and `README.md`.
3. Build the **persistence foundation (P1)**, then the **local read-only engine (P2)**, before adding remote transport or any mutating operation.
4. Proceed phase by phase. Do not skip gates.
