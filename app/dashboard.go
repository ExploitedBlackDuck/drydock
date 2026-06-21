package app

import (
	"context"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/restartloop"
)

// Restart-loop detection defaults: a container that dies 3+ times within two
// minutes is flagged (PROJECT-BOOK §7.6).
const (
	restartThreshold = 3
	restartWindow    = 2 * time.Minute
)

// RestartLoopAlert is the typed payload pushed on the "restart-loop:<hostID>"
// event when a container is detected restarting repeatedly.
type RestartLoopAlert struct {
	HostID        string `json:"hostId"`
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Deaths        int    `json:"deaths"`
}

// Event-stream supervision (ADR-0021): when a live event stream drops, the UI is
// told to resync (refetch authoritative state, not trust stale data) and a
// bounded reconnect is attempted, so a transport blip resyncs against a live
// connection instead of resuming a stale one.
const (
	eventReconnectMin = 1 * time.Second
	eventReconnectMax = 30 * time.Second
)

// ResyncDTO is pushed on "resync:<hostID>" when the live stream is interrupted or
// re-established, so the frontend refetches authoritative state.
type ResyncDTO struct {
	HostID string `json:"hostId"`
	// Reason is "stream-interrupted" or "reconnected".
	Reason string `json:"reason"`
}

// SubscribeEvents streams a host's engine events, emitting "events:<hostID>" for
// live refresh, "restart-loop:<hostID>" alerts, and "resync:<hostID>" when the
// stream drops or is re-established. Calling it again, or UnsubscribeEvents,
// cancels the prior subscription.
func (a *App) SubscribeEvents(hostID string) error {
	if _, err := a.registry.Engine(hostID); err != nil {
		return err
	}
	key := "events:" + hostID
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.setStream(key, cancel)
	go a.superviseEvents(ctx, key, hostID)
	return nil
}

// superviseEvents runs the event stream and, on a drop, signals a resync and
// retries the connection with bounded backoff until the subscription is
// cancelled. The owning context ties the whole loop (including an in-flight
// reconnect) to UnsubscribeEvents / quit — nothing is leaked.
func (a *App) superviseEvents(ctx context.Context, key, hostID string) {
	defer a.clearStream(key)
	detector := restartloop.New(restartThreshold, restartWindow)

	//nolint:contextcheck // resync emits on the app-lifetime event bus, not the stream ctx
	resync := func(reason string) {
		a.runtime.EmitEvent(a.baseCtx(), "resync:"+hostID, ResyncDTO{HostID: hostID, Reason: reason})
	}

	backoff := eventReconnectMin
	for {
		if ctx.Err() != nil {
			return
		}
		if eng, err := a.registry.Engine(hostID); err == nil {
			backoff = eventReconnectMin
			//nolint:contextcheck // the sink emits on the app-lifetime event bus, not ctx
			_ = eng.StreamEvents(ctx, func(e domain.EngineEvent) {
				a.runtime.EmitEvent(a.baseCtx(), key, e.Action)
				if alert, looping := detector.Observe(e); looping {
					a.runtime.EmitEvent(a.baseCtx(), "restart-loop:"+hostID, RestartLoopAlert{
						HostID:        hostID,
						ContainerID:   alert.ContainerID,
						ContainerName: alert.ContainerName,
						Deaths:        alert.Deaths,
					})
				}
			})
			if ctx.Err() != nil {
				return
			}
			// The stream returned without a cancel: the connection dropped.
			resync("stream-interrupted")
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if _, err := a.registry.Reconnect(ctx, hostID); err == nil {
			resync("reconnected")
			backoff = eventReconnectMin
		} else {
			backoff *= 2
			if backoff > eventReconnectMax {
				backoff = eventReconnectMax
			}
		}
	}
}

// UnsubscribeEvents stops a host's event subscription.
func (a *App) UnsubscribeEvents(hostID string) {
	a.stopStream("events:" + hostID)
}

// GetResourceHistory returns the recent rolling resource samples for a container
// (PROJECT-BOOK §7.6), oldest first, ready to plot.
func (a *App) GetResourceHistory(hostID, containerID string, limit int) ([]domain.ResourceSample, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	if limit <= 0 || limit > 1000 {
		limit = 120
	}
	return a.samples.RecentResourceSamples(ctx, hostID, containerID, limit)
}
