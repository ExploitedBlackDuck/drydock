package app

import (
	"context"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/timeline"
)

// clockSkewThreshold is how far apart the host and desktop clocks must be before
// the timeline warns that ordering is uncertain.
const clockSkewThreshold = 5 * time.Second

// TimelineStore persists mapped engine events and reads them back with the audit
// log, for the host timeline (ADR-0018). It is consumer-defined here and
// satisfied by the SQLite store.
type TimelineStore interface {
	SaveTimelineEntry(ctx context.Context, e domain.TimelineEntry) error
	RecentTimelineEntries(ctx context.Context, hostID string, limit int) ([]domain.TimelineEntry, error)
	AuditEntries(ctx context.Context) ([]domain.AuditEntry, error)
}

// HostTimelineDTO is the timeline view's payload: the interleaved entries plus
// the observed clock skew (when significant) so the UI can flag uncertain
// ordering rather than hide it.
type HostTimelineDTO struct {
	Entries     []domain.TimelineEntry `json:"entries"`
	SkewSeconds float64                `json:"skewSeconds"`
}

// recordEngineEvent persists a local-scope engine event to the host timeline and
// updates the observed clock skew. Swarm-scope events are dropped (ADR-0018).
func (a *App) recordEngineEvent(hostID string, e domain.EngineEvent) {
	entry, ok := timeline.FromEvent(hostID, e)
	if !ok {
		return
	}
	skew := timeline.Skew(e.At, time.Now().UTC())
	a.skewMu.Lock()
	a.hostSkew[hostID] = skew
	a.skewMu.Unlock()

	if a.timeline != nil {
		_ = a.timeline.SaveTimelineEntry(a.baseCtx(), entry)
	}
}

// recordTimelineGap marks a break in the event stream (a reconnect), so the
// timeline is honest that it may have missed events while disconnected.
func (a *App) recordTimelineGap(hostID string) {
	if a.timeline == nil {
		return
	}
	_ = a.timeline.SaveTimelineEntry(a.baseCtx(), domain.TimelineEntry{
		HostRef: hostID,
		At:      time.Now().UTC(),
		Source:  domain.TimelineEngine,
		Kind:    "stream-gap",
		Subject: "event stream interrupted — some events may be missing",
	})
}

// GetHostTimeline returns the host's timeline: persisted engine events
// interleaved with this host's audit entries, plus the observed clock skew.
func (a *App) GetHostTimeline(hostID string, limit int) (HostTimelineDTO, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	engine, err := a.timeline.RecentTimelineEntries(ctx, hostID, limit)
	if err != nil {
		return HostTimelineDTO{}, err
	}
	allAudit, err := a.timeline.AuditEntries(ctx)
	if err != nil {
		return HostTimelineDTO{}, err
	}
	hostAudit := make([]domain.AuditEntry, 0, len(allAudit))
	for _, entry := range allAudit {
		if entry.HostRef == hostID {
			hostAudit = append(hostAudit, entry)
		}
	}

	a.skewMu.Lock()
	skew := a.hostSkew[hostID]
	a.skewMu.Unlock()
	if skew < 0 {
		skew = -skew
	}
	skewSeconds := 0.0
	if skew >= clockSkewThreshold {
		skewSeconds = skew.Seconds()
	}

	return HostTimelineDTO{Entries: timeline.Merge(engine, hostAudit), SkewSeconds: skewSeconds}, nil
}
