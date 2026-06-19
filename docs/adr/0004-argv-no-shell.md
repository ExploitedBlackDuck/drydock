# ADR-0004 — Any subprocess is spawned argv-style; never through a shell

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Drydock is mostly SDK and library calls, but `exec` into a container, an SSH
helper, or a compose shell-out must never interpolate operator input into a
shell.

## Decision

Any process is launched with explicit argv
(`exec.CommandContext(ctx, bin, args...)`); container `exec` uses the Engine API
exec endpoints with argv, not a shell string. No `sh -c`, no concatenation, no
operator input reaching a shell.

## Consequences

Eliminates a command-injection class. Enforced by `gosec` and review.
