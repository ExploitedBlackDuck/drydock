# ADR-0011 — Operations are a typed catalog with impact warnings, and destruction is preview-and-confirm

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Docker operations carry a large, evolving set of options, and several are
destructive or risky — `prune` (especially `--volumes`), `rm -f` on a running
container, `down -v`, `exec` as root. Cleanup is a leading cause of accidental
data loss; volumes are excluded from default prune precisely because removing the
wrong one is permanent.

## Decision

A versioned **catalog** describes each operation's options as typed metadata
(name, type, default, category, help, risk, `affects_data`, `conflicts_with`,
`requires`, `impacts`). The UI is generated from it; a **builder** assembles
validated API parameters (ADR-0003/0004). An **impact-rule engine** plus a
**prune-impact calculator** show, before execution, exactly what will be affected
— for prune, the precise objects and reclaimable bytes by category
(containers / images / build cache / volumes), with **named volumes never
bulk-deleted**: each must be individually confirmed. Destructive operations
require an explicit acknowledgement recorded in the audit log, and are blocked
entirely on observe-mode hosts (ADR-0013).

## Consequences

Cleanup stops being a guessing game; the data-loss footgun is removed. Cost: the
catalog is maintained per API version and tested against the engine's actual
options.
