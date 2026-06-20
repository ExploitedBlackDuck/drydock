# ADR-0015 — Reviewed govulncheck allowlist for unfixable, unreachable advisories

- **Status:** Accepted
- **Date:** 2026-06-20
- **Relates to:** ADR-0003 (Docker Go SDK), PROJECT-BOOK §2.6 (govulncheck must be clean)

## Context

The Engineering Charter (§2.6) requires `govulncheck` to be clean — a tool with
root-equivalent access must not ship a known-vulnerable dependency. ADR-0003
mandates the first-party Docker Go SDK (`github.com/docker/docker`); there is no
alternative typed client, and shelling out to the `docker` CLI is rejected.

That module is the Moby monorepo, published as `+incompatible` (no semver
modules). As of June 2026 it carries two advisories with **no fixed release**:

- **GO-2026-4883** — off-by-one in plugin **privilege validation** (daemon).
- **GO-2026-4887** — **authorization-plugin** bypass on oversized request bodies
  (daemon).

Both are **daemon-side**. Drydock is a Docker *client*: it never runs the daemon,
installs plugins, or implements authorization plugins, so neither vulnerable code
path is reachable from our list/inspect usage. `govulncheck` nonetheless reports
them as "called" because the monorepo's package `init` graph is reachable from
importing the client — the call traces show only `init()` and client list calls,
never the vulnerable daemon functions.

The book anticipated exactly this: *"Re-verify before locking dependency
versions."*

## Decision

Keep `govulncheck` as a CI gate, but run it through `scripts/govulncheck.sh`,
which triages results against a reviewed allowlist
(`.govulncheck-allowlist.txt`). A vulnerability that affects our code fails the
build **unless** its OSV id is listed with a written justification. Any new or
unreviewed vulnerability still fails.

The allowlist currently contains only GO-2026-4883 and GO-2026-4887, justified
above. We track the Docker SDK and remove entries as fixes ship or the advisories
are withdrawn.

## Consequences

The gate stays meaningful: it catches every reachable, unreviewed vulnerability,
including any newly disclosed one in the Docker chain. The cost is a small triage
script and a periodic review of the allowlist. Each exception is auditable in
version control with its rationale.
