# ADR-0016 — Compose changes are plan-first: `up` is preview-and-confirm, not a black box

- **Status:** Accepted (implemented incrementally)
- **Date:** 2026-06-21

## Context

`compose up` against a changed project silently recreates containers, recreates
or removes networks, and can drop anonymous volumes — the operator learns what
happened only afterwards. This is the same surprise-and-data-loss class the book
already guards on the *delete* path (ADR-0011, §7.4), now on the *apply* path.
Compose decides what to recreate by comparing a per-service config hash stored on
each container as the `com.docker.compose.config-hash` label; reproducing that
hash from outside Compose is version-fragile.

This ADR is numbered 0016 (the project book numbers it ADR-0014); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's Phase 2 ADRs are recorded here shifted by two.

## Decision

Before applying, Drydock computes a **`ComposePlan`**: the desired project
(parsed with the pinned `compose-go`) diffed against observed engine state,
classifying each service as create / recreate / start / no-op and each network
and volume as create / recreate / remove, and flagging recreations that interrupt
a running container or drop an anonymous volume. Convergence detection uses **the
same Compose library version Drydock applies with** — never the `config --hash`
CLI — and where a hash cannot be matched with confidence the plan **degrades to a
coarser, clearly-labelled diff** (image/tag/digest, ports, env/mount presence)
rather than asserting precision it does not have. The plan is computed through the
SDK / `compose-go` path, never a shell (ADR-0003/0004); Compose's own `up`
remains the authority that actually converges state. Destructive plan elements
route through the §7.4 acknowledgement-and-audit path.

**A precise plan requires the project source.** Drydock locates it from the
running containers' `com.docker.compose.project.config_files` / `working_dir`
labels; it computes a full plan when that source is accessible to it (a local
host, or compose files the operator points it at). It does not read arbitrary
files off a remote host, so when the source is not locally accessible the plan
enters an explicit **`source-unavailable` degraded state**: a labelled best-effort
diff built from container labels, never a false "no changes."

## Consequences

`compose up` becomes legible before it runs. Cost: the diff must be computed with
the pinned Compose library and **re-verified on every Compose upgrade** — the hash
derivation is out-of-spec and has changed between versions — and remote stacks
without locally-accessible source get the honest degraded plan, not a precise one.

**Implementation status.** The pure plan classifier (`internal/core/compose.Plan`)
and the `domain.ComposePlan` types are implemented and table-tested against
fixtures, including the config-hash-mismatch → `Degraded` case, the
source-unavailable degraded case (never a false no-change), and the
anonymous-volume-dropping recreate → `Destructive` case. The `compose-go` parsing
adapter (desired state from compose files), the engine apply path, the
acknowledgement-and-audit wiring, and the GUI `ComposePlanPanel` are the
remaining P10 work.
