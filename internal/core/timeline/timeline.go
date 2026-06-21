// Package timeline builds a host's explain-and-detect-drift view (PROJECT-BOOK
// §7.12.4, ADR-0018): it interleaves mapped engine events with references to
// Drydock's own audit entries, correlated by time. The hash-chained audit log is
// never merged into or weakened by the timeline — engine events are untrusted
// input and are mapped here, never written into the chain. The timeline is
// best-effort and labelled; the audit log keeps its completeness guarantee for
// Drydock-authored actions.
package timeline

import (
	"sort"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// FromEvent maps an engine event to a timeline entry, keeping only local-scope
// events so the timeline reflects this host, not a swarm cluster (ADR-0018). It
// returns ok=false for a non-local event, which the caller drops.
func FromEvent(hostRef string, e domain.EngineEvent) (domain.TimelineEntry, bool) {
	if e.Scope != "" && e.Scope != "local" {
		return domain.TimelineEntry{}, false
	}
	subject := e.ContainerName
	if subject == "" {
		subject = e.ContainerID
	}
	return domain.TimelineEntry{
		HostRef:      hostRef,
		At:           e.At,
		Source:       domain.TimelineEngine,
		Kind:         e.Action,
		Subject:      subject,
		ExitCode:     e.ExitCode,
		HealthStatus: e.HealthStatus,
	}, true
}

// Merge interleaves engine timeline entries with audit entries (as references),
// sorted by each row's own authoritative timestamp. Engine rows keep host time
// and audit rows keep desktop time — neither is adjusted, so a clock skew is
// surfaced (via Skew) rather than hidden by reordering. The audit slice is read
// only; it is never modified.
func Merge(engine []domain.TimelineEntry, audit []domain.AuditEntry) []domain.TimelineEntry {
	out := make([]domain.TimelineEntry, 0, len(engine)+len(audit))
	out = append(out, engine...)
	for _, a := range audit {
		out = append(out, domain.TimelineEntry{
			HostRef: a.HostRef,
			At:      a.At,
			Source:  domain.TimelineAudit,
			Kind:    string(a.Action),
			Subject: a.Subject,
			Detail:  a.Detail,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

// Skew returns how far an engine event's host clock is from the desktop clock at
// receipt (desktop − host). A large magnitude means engine and audit rows cannot
// be globally ordered by timestamp with confidence; the UI surfaces it rather
// than silently misordering (ADR-0018).
func Skew(hostEventTime, desktopReceiptTime time.Time) time.Duration {
	return desktopReceiptTime.Sub(hostEventTime)
}
