# ADR-0009 — Secrets in the OS keyring; sensitive persisted data encrypted at the application layer

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

SSH passphrases and TLS client keys are root-equivalent; pure-Go SQLite
(ADR-0007) offers no transparent encryption. Drydock should not become a second
store of host credentials.

## Decision

SSH authentication **prefers the agent and the on-disk keys the operator already
manages** — Drydock references them and does not copy or store private keys.
Where a secret must be retained (e.g. an optional SSH passphrase or a TLS
client-key passphrase), it is stored in the **OS keyring**, never in the DB or
config. A per-install **data key** in the keyring seals any sensitive persisted
field via **AEAD (XChaCha20-Poly1305)** before it touches disk. The data
directory is OS-permission-restricted (`0700`/`0600`).

## Consequences

No new plaintext credential store; secrets never on disk or in logs. Cost: a
keyring round-trip, unit-tested.
