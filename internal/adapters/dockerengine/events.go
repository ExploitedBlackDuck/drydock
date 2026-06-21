package dockerengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"

	"github.com/drydock/drydock/internal/core/domain"
)

// StreamEvents subscribes to the engine event stream, mapping each message and
// delivering it to sink until ctx is cancelled. No goroutine outlives the call.
func (c *Client) StreamEvents(ctx context.Context, sink func(domain.EngineEvent)) error {
	messages, errs := c.cli.Events(ctx, events.ListOptions{})
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			// EOF or a cancelled context is a clean end of stream, not a failure.
			if err == nil || errors.Is(err, io.EOF) || ctx.Err() != nil {
				return nil //nolint:nilerr // graceful shutdown on EOF/cancel
			}
			return fmt.Errorf("streaming events on host %q: %w", c.hostRef, err)
		case msg := <-messages:
			sink(mapEvent(msg))
		}
	}
}

// mapEvent converts a Docker event message to the domain type. Pure and
// fixture-tested. Compound actions (e.g. "health_status: healthy") are reduced
// to their base action.
func mapEvent(m events.Message) domain.EngineEvent {
	at := time.Unix(0, m.TimeNano).UTC()
	if m.TimeNano == 0 {
		at = time.Unix(m.Time, 0).UTC()
	}
	event := domain.EngineEvent{
		Type:          string(m.Type),
		Action:        baseAction(string(m.Action)),
		ContainerID:   m.Actor.ID,
		ContainerName: m.Actor.Attributes["name"],
		Scope:         string(m.Scope),
		At:            at,
	}
	// A die event carries the exit code in its actor attributes (§7.12.4).
	if code, ok := m.Actor.Attributes["exitCode"]; ok {
		if n, err := strconv.Atoi(code); err == nil {
			event.ExitCode = &n
		}
	}
	// A health_status event's status is the suffix after the colon.
	if event.Action == domain.EventActionHealth {
		if _, status, found := strings.Cut(string(m.Action), ":"); found {
			event.HealthStatus = strings.TrimSpace(status)
		}
	}
	return event
}

func baseAction(action string) string {
	if idx := strings.IndexByte(action, ':'); idx >= 0 {
		return strings.TrimSpace(action[:idx])
	}
	return action
}
