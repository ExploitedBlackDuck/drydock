// Package restartloop detects containers that are restarting repeatedly — a
// crash loop — from the engine's event stream (PROJECT-BOOK §7.6). It counts
// "die" events per container within a sliding window and raises an alert when a
// container crosses the threshold.
package restartloop

import (
	"sync"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// Alert reports a container that is restarting too often.
type Alert struct {
	ContainerID   string
	ContainerName string
	Deaths        int
	WindowStart   time.Time
	WindowEnd     time.Time
}

// Detector tracks recent container deaths and flags restart loops. It is safe
// for concurrent use. Timing is taken from each event, so it is deterministic
// under test.
type Detector struct {
	threshold int
	window    time.Duration

	mu     sync.Mutex
	deaths map[string][]time.Time
	names  map[string]string
}

// New returns a detector that alerts when a container dies threshold or more
// times within window.
func New(threshold int, window time.Duration) *Detector {
	if threshold < 1 {
		threshold = 1
	}
	return &Detector{
		threshold: threshold,
		window:    window,
		deaths:    map[string][]time.Time{},
		names:     map[string]string{},
	}
}

// Observe feeds one event to the detector. It returns an Alert and true when the
// event causes the container to reach the restart-loop threshold. A "destroy"
// event clears the container's history.
func (d *Detector) Observe(e domain.EngineEvent) (Alert, bool) {
	if e.Type != domain.EventTypeContainer || e.ContainerID == "" {
		return Alert{}, false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if e.ContainerName != "" {
		d.names[e.ContainerID] = e.ContainerName
	}

	if e.Action == domain.EventActionDestroy {
		delete(d.deaths, e.ContainerID)
		delete(d.names, e.ContainerID)
		return Alert{}, false
	}
	if e.Action != domain.EventActionDie {
		return Alert{}, false
	}

	cutoff := e.At.Add(-d.window)
	recent := append(trimBefore(d.deaths[e.ContainerID], cutoff), e.At)
	d.deaths[e.ContainerID] = recent

	if len(recent) >= d.threshold {
		return Alert{
			ContainerID:   e.ContainerID,
			ContainerName: d.names[e.ContainerID],
			Deaths:        len(recent),
			WindowStart:   recent[0],
			WindowEnd:     e.At,
		}, true
	}
	return Alert{}, false
}

// trimBefore drops timestamps at or before cutoff (outside the window).
func trimBefore(times []time.Time, cutoff time.Time) []time.Time {
	kept := times[:0:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	return kept
}
