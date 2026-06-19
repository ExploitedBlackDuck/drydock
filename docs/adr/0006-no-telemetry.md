# ADR-0006 — No telemetry

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

A tool with root-equivalent reach into the operator's servers must not exfiltrate
anything the operator did not initiate.

## Decision

Zero analytics, crash reporting, or network calls the user didn't initiate. Logs
are local. This is stated in the README as a feature.

## Consequences

Trust by default. Any future opt-in diagnostic would require its own ADR and an
explicit, off-by-default consent flow.
