// Package stream provides backpressure primitives for the headless core's live
// streams (logs, events) as they cross to the UI event bus (PROJECT-BOOK §2.3,
// ADR-0021). A fast engine producer must never grow an unbounded buffer behind
// a slower consumer; instead it coalesces and drops, reporting how far it fell
// behind so the UI can say so rather than silently lose data.
package stream

import "sync"

// LineBuffer is a bounded, coalescing buffer between a fast producer and a slower
// consumer. Push never blocks; when the buffer is full the oldest line is
// dropped and counted, so memory is bounded by the capacity and Drain reports
// how many lines were lost since the last drain — the "stream fell behind"
// signal — instead of the buffer growing without limit.
type LineBuffer struct {
	mu      sync.Mutex
	cap     int
	lines   []string
	dropped int
}

// NewLineBuffer returns a buffer that holds at most capacity lines (minimum 1).
func NewLineBuffer(capacity int) *LineBuffer {
	if capacity < 1 {
		capacity = 1
	}
	return &LineBuffer{cap: capacity}
}

// Push adds a line, dropping the oldest (and counting it) when full.
func (b *LineBuffer) Push(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.lines) >= b.cap {
		b.lines = b.lines[1:]
		b.dropped++
	}
	b.lines = append(b.lines, line)
}

// Drain returns the buffered lines (oldest first) and the number dropped since
// the previous drain, resetting both to empty.
func (b *LineBuffer) Drain() (lines []string, dropped int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	lines, dropped = b.lines, b.dropped
	b.lines, b.dropped = nil, 0
	return lines, dropped
}
