# ADR-0008 — Engine API version negotiation and a declared minimum supported engine

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Docker engines in the field run a range of API versions; the SDK supports
negotiation. There is no single pinned external binary to verify (we link the
SDK), so the risk is API drift, not a tampered binary.

## Decision

The `Engine` adapter negotiates the API version per host and records the engine
and API version on every operation. Drydock declares a **minimum supported API
version**; a host below it connects in a clearly-labelled reduced-capability
mode rather than failing opaquely. Capability differences are surfaced in the
UI, never guessed.

## Consequences

Works across a heterogeneous fleet; behaviour differences are explicit. Cost:
capability checks per feature.
