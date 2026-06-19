# ADR-0010 — Operation capture and an append-only, hash-chained audit log

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

A tool with root-equivalent reach that can delete data must be able to answer,
durably and verifiably, what it changed and what authorized it.

## Decision

Every mutating operation's target, host, resolved parameters, and result are
persisted (ADR-0007/0009). Every consequential action (start/stop/restart/
remove, **prune with its exact impact**, exec, compose up/down, host
connect/disconnect, observe-mode changes) is recorded as an **append-only audit
entry whose hash chains to the previous entry**:

```
hash = SHA256(prev_hash || canonical(entry))
```

The chain is verifiable and exportable.

## Consequences

A complete, tamper-evident record. Cost: storage, managed by retention and
encryption.
