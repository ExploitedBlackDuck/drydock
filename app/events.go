package app

// Event names emitted toward the frontend. They are typed constants in one
// place (PROJECT-BOOK §2.7) so the Go and TypeScript sides never disagree on a
// magic string. The frontend subscribes to these names.
const (
	// EventAppReady is emitted once at startup, carrying the running version.
	EventAppReady = "app:ready"
)

// logsEvent and statsEvent name the per-container streaming channels the frontend
// subscribes to (e.g. "logs:<id>", "stats:<id>").
func logsEvent(containerID string) string  { return "logs:" + containerID }
func statsEvent(containerID string) string { return "stats:" + containerID }
