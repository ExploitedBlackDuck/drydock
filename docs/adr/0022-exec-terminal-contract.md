# ADR-0022 — Interactive exec is a first-class terminal contract, not just argv

- **Status:** Accepted (implemented incrementally)
- **Date:** 2026-06-21

## Context

§7.11.4 promises a reliable in-app terminal over SSH-tunnelled remotes — exactly
the area competing tools handle badly. ADR-0004/§7.3 pin exec to argv via the API
exec endpoints, but the terminal mechanics that *determine* reliability were
unspecified; argv-without-a-terminal-contract is the easy 80%, and the
differentiator lives in the other 20%.

This ADR is numbered 0022 (the project book numbers it ADR-0020); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's gap-review ADRs are recorded here shifted by two.

## Decision

The exec terminal uses the Engine API exec/attach endpoints with a **PTY when
interactive** (`Tty: true`), **bidirectional**: stdin is streamed to the exec and
stdout/stderr stream back over an ADR-0021 subscription. **Terminal resize is
propagated** via the engine's resize endpoint on every UI resize event; **stdin
half-close** and **detach** are handled explicitly; a non-interactive exec
(`Tty: false`) keeps stdout and stderr separated for argv-style command runs. The
session is a supervised stream (ADR-0021): closing the pane cancels the exec
context and the attach connection. Over a reconnect an exec session is **dropped,
not silently resumed**, and the operator is told.

## Consequences

The headline differentiator is specified where it actually matters. Cost: PTY +
resize handling and a tested teardown path; exec sessions do not survive a
reconnect by design.

**Implementation status.** PTY, bidirectional streaming, resize, and supervised
teardown were delivered with the exec terminal; **stdin half-close** is now
implemented (`ExecStream.CloseStdin` / the `CloseExecInput` binding, integration
tested). A reconnect already drops the session: the attach connection breaks, the
output pump errors, and the operator sees the session end. Explicit detach (leave
the command running while detaching output) and non-interactive stdout/stderr
separation remain to be implemented and are tracked as follow-up work.
