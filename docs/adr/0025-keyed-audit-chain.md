# ADR-0025 — The audit chain is HMAC-keyed and truncation-aware, with honestly-stated guarantees

- **Status:** Accepted
- **Date:** 2026-06-21

## Context

ADR-0010's chain (`hash = SHA256(prev_hash || canonical(entry))`) is
tamper-*evident* against in-place edits — and is correctly never claimed as
tamper-*proof*. But a plain hash chain is **recomputable** by anyone who can
write the DB and reproduce the canonicalization, and **tail-truncation** is
undetectable without an external anchor. For a security tool the bar should be
both raised and documented.

This ADR is numbered 0025 (the project book numbers it ADR-0023); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist decisions,
so the book's gap-review ADRs are recorded here shifted by two.

## Decision

The chain is **keyed**: `mac = HMAC-SHA256(K, prev_mac || canonical(entry))`,
where **K is a per-install audit key in the OS keyring** (ADR-0009), separate
from the data-sealing key — so recomputing the chain also requires the keyring
secret, not merely DB write access. The latest `(seq, mac)` is persisted as an
**external high-water mark** (a small JSON file outside the SQLite database) so
**tail truncation is detected** on verify: a chain whose last row is behind the
mark has lost entries.

Verification classifies the chain into one of four states, surfaced in the UI
(§7.11.8):

- **intact** — the keyed chain validates end to end and its tail matches the mark.
- **in_place_tampered** — an entry's content no longer matches its MAC, or a
  sequence/back-link is broken.
- **truncated** — the keyed chain is internally valid but shorter than the mark.
- **key_unavailable** — the audit key is absent (keyring locked/unreachable), so
  authenticity cannot be confirmed; only structural consistency is checked. The
  log degrades to a plain SHA-256 chain in this mode rather than blocking startup.

The book states plainly what this defends against — accidental corruption,
casual and in-place tampering, tail truncation — and what it does **not**: an
attacker with simultaneous DB write **and** keyring access can still forge a
consistent chain. This is a single-operator desktop trust model, stated as such.

## Consequences

A materially stronger, honestly-scoped audit guarantee. Cost: an HMAC key in the
keyring and a high-water-mark check on verify; with the key unavailable the chain
still verifies structurally but is flagged **key-unavailable** rather than
silently "intact." Upgrade note: entries written by an earlier build under the
unkeyed SHA-256 scheme do not match the keyed MAC, so a chain that predates this
change verifies as `in_place_tampered` until re-based; fresh installs are keyed
from the first entry. Enforced by unit tests for all four states (including a
real-store truncation test) and the keyring round-trip.
