package app

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
	"github.com/drydock/drydock/internal/core/stream"
)

// maxLogLine bounds a single scanned log line so a pathological stream cannot
// exhaust memory.
const maxLogLine = 1024 * 1024

// Log-stream backpressure (ADR-0021): the reader fills a bounded buffer that a
// ticker drains to the event bus, so a flood coalesces and drops with a marker
// rather than growing without limit.
const (
	logBufferLines   = 1000
	logFlushInterval = 100 * time.Millisecond
)

// SampleStore persists and reads rolling resource-history samples
// (PROJECT-BOOK §7.6). It is consumer-defined here and satisfied by the SQLite
// store.
type SampleStore interface {
	SaveResourceSample(ctx context.Context, s domain.ResourceSample) error
	RecentResourceSamples(ctx context.Context, hostID, containerID string, limit int) ([]domain.ResourceSample, error)
}

// StatsDTO is the typed shape pushed to the frontend on each stats sample.
type StatsDTO struct {
	ContainerID string  `json:"containerId"`
	CPUPct      float64 `json:"cpuPct"`
	MemBytes    int64   `json:"memBytes"`
	NetRx       int64   `json:"netRx"`
	NetTx       int64   `json:"netTx"`
}

// StreamContainerLogs follows a container's logs, emitting each line on the
// "logs:<id>" event. Calling it again, or StopContainerLogs, cancels the prior
// stream — no reader or goroutine is leaked.
func (a *App) StreamContainerLogs(hostID, containerID string) error {
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return err
	}

	key := logsEvent(containerID)
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.setStream(key, cancel)

	rc, err := eng.ContainerLogs(ctx, containerID, engine.LogOptions{Follow: true, Tail: 300})
	if err != nil {
		cancel()
		a.clearStream(key)
		return err
	}

	// The reader fills a bounded buffer; a separate drainer emits to the event bus
	// on a ticker, coalescing bursts and marking drops (ADR-0021). Both goroutines
	// stop on ctx cancellation (StopContainerLogs / quit) — no stream is leaked.
	buf := stream.NewLineBuffer(logBufferLines)
	done := make(chan struct{})

	go func() {
		defer func() { _ = rc.Close() }()
		defer close(done)
		scanner := bufio.NewScanner(rc)
		scanner.Buffer(make([]byte, 0, 64*1024), maxLogLine)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			buf.Push(scanner.Text())
		}
	}()

	go func() {
		defer a.clearStream(key)
		ticker := time.NewTicker(logFlushInterval)
		defer ticker.Stop()
		// flush drains the buffer to the log event, prefixing a "fell behind"
		// marker when lines were dropped. It emits on the long-lived app context,
		// not the cancellable stream ctx, so the bus stays usable for the app life.
		//nolint:contextcheck // emits on the app-lifetime event bus, not the stream ctx
		flush := func() {
			lines, dropped := buf.Drain()
			if dropped > 0 {
				a.runtime.EmitEvent(a.baseCtx(), key,
					fmt.Sprintf("⟪ %d line(s) dropped — stream fell behind ⟫", dropped))
			}
			for _, line := range lines {
				a.runtime.EmitEvent(a.baseCtx(), key, line)
			}
		}
		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case <-done:
				flush()
				return
			case <-ticker.C:
				flush()
			}
		}
	}()
	return nil
}

// StopContainerLogs cancels a container's log stream.
func (a *App) StopContainerLogs(containerID string) {
	a.stopStream(logsEvent(containerID))
}

// StreamContainerStats follows a container's stats, emitting a StatsDTO on the
// "stats:<id>" event and persisting each sample to the rolling history.
func (a *App) StreamContainerStats(hostID, containerID string) error {
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return err
	}

	key := statsEvent(containerID)
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.setStream(key, cancel)

	go func() {
		defer a.clearStream(key)
		_ = eng.StreamStats(ctx, containerID, func(s domain.ResourceSample) {
			//nolint:contextcheck // emit on the app-lifetime event bus, not the stream ctx
			a.runtime.EmitEvent(a.baseCtx(), key, StatsDTO{
				ContainerID: containerID,
				CPUPct:      s.CPUPct,
				MemBytes:    s.MemBytes,
				NetRx:       s.NetRx,
				NetTx:       s.NetTx,
			})
			if a.samples != nil {
				_ = a.samples.SaveResourceSample(ctx, s)
			}
		})
	}()
	return nil
}

// StopContainerStats cancels a container's stats stream.
func (a *App) StopContainerStats(containerID string) {
	a.stopStream(statsEvent(containerID))
}

func (a *App) baseCtx() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) setStream(key string, cancel context.CancelFunc) {
	a.streamMu.Lock()
	defer a.streamMu.Unlock()
	if existing, ok := a.streams[key]; ok {
		existing()
	}
	a.streams[key] = cancel
}

func (a *App) clearStream(key string) {
	a.streamMu.Lock()
	defer a.streamMu.Unlock()
	delete(a.streams, key)
}

func (a *App) stopStream(key string) {
	a.streamMu.Lock()
	defer a.streamMu.Unlock()
	if cancel, ok := a.streams[key]; ok {
		cancel()
		delete(a.streams, key)
	}
}
