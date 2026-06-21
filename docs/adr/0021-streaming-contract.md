# ADR-0021 — The streaming & event-binding contract: subscriptions, backpressure, cancellation, reconnect-resync

- **Status:** Accepted (implemented incrementally)
- **Date:** 2026-06-21

## Context

Logs, stats, the engine event stream, and the Phase 2 timeline all push from the
headless core to the UI across the one Wails binding layer (`app/events.go`).
§2.3 requires every goroutine to have an owner and forbids leaked streams, but the
*contract* that makes that true — how a view subscribes and unsubscribes, what
happens under backpressure, how a cancel crosses the boundary, how live state is
reconciled after a reconnect — was implicit.

This ADR is numbered 0021 (the project book numbers it ADR-0019); the repository
keeps ADR-0014/0015 for the composition-root and govulncheck-allowlist
decisions, so the book's gap-review ADRs are recorded here shifted by two.

## Decision

(1) **Subscriptions are correlated:** a stream is opened by a bound method that
returns (or is keyed by) a correlation id; typed events arrive on `app/events.go`
channels keyed by that id; the view **unsubscribes on unmount**, which cancels the
owning `context` in the core and tears down the underlying engine stream — no
subscription outlives its view. (2) **Backpressure is explicit per stream:** logs
and the event stream **coalesce and drop with a visible "stream fell behind"
marker** rather than growing an unbounded buffer; stats are **fixed-cadence
sampled** and so naturally rate-limited. (3) **Cancellation crosses by id:** a
bound stop/cancel cancels the core context and tears down the SDK request and
connection (the P4 no-leak gate). (4) **Reconnect means resync, not resume:**
after a supervised transport reconnect, live views **refetch authoritative state**
rather than resuming a stale stream; the timeline marks the gap; resync is
debounced and bounded. (5) The streaming surface sits behind one internal
interface so the Wails event API touches only `app/events.go`, never the core.

## Consequences

The robustness claim becomes testable beyond "no leak": backpressure and resync
have defined behavior. Cost: a small subscription registry in the binding layer
and a resync path per live view.

**Implementation status.** Backpressure (2) is implemented: a bounded
`internal/core/stream.LineBuffer` coalesces the log stream and a ticker drains it,
emitting a "fell behind" marker on drops — never an unbounded buffer (unit
tested). Cancellation (3) and unsubscribe-on-unmount (1) hold for the existing
per-stream stop bindings. Reconnect-driven resync (4) is implemented: the app's
event supervisor, on a dropped stream, emits `resync:<hostID>` and retries the
connection with bounded backoff via `registry.Reconnect` (unit tested); the
frontend `ResyncWatcher` refetches host status and object lists on the signal
rather than resuming stale data. The opaque correlation-id registry unification
(replacing the per-resource keys with a single `Cancel(id)`) remains a tracked
follow-up — behaviour is correct today, the refactor is cosmetic.
