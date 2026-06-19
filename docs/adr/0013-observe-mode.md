# ADR-0013 — Per-host observe (read-only) mode

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Some hosts (production) should be inspectable but not mutable from a desktop
tool; a single mis-click should not be able to stop a prod container or prune a
prod volume.

## Decision

A host can be marked **observe-only**. In observe mode the core **rejects every
mutating operation** with `ErrObserveMode` before it reaches the engine —
enforced in the core, not just hidden in the UI — and the UI presents the host as
read-only. Leaving observe mode is an explicit, audited action.

## Consequences

A hard, testable guardrail for sensitive hosts. Cost: a mode check on the
mutating path (cheap, and unit-tested).
