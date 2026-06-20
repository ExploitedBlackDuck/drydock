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

// SubscribeEvents streams a host's engine events, emitting "events:<hostID>" for
// live refresh and "restart-loop:<hostID>" alerts. Calling it again, or
// UnsubscribeEvents, cancels the prior subscription.
func (a *App) SubscribeEvents(hostID string) error {
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return err
	}

	key := "events:" + hostID
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.setStream(key, cancel)

	detector := restartloop.New(restartThreshold, restartWindow)
	go func() {
		defer a.clearStream(key)
		//nolint:contextcheck // the sink emits on the app-lifetime event bus, not ctx
		_ = eng.StreamEvents(ctx, func(e domain.EngineEvent) {
			a.runtime.EmitEvent(a.baseCtx(), key, e.Action) //nolint:contextcheck // app-lifetime event bus
			if alert, looping := detector.Observe(e); looping {
				a.runtime.EmitEvent(a.baseCtx(), "restart-loop:"+hostID, RestartLoopAlert{ //nolint:contextcheck // app-lifetime event bus
					HostID:        hostID,
					ContainerID:   alert.ContainerID,
					ContainerName: alert.ContainerName,
					Deaths:        alert.Deaths,
				})
			}
		})
	}()
	return nil
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
