package app

// Event names emitted toward the frontend. They are typed constants in one
// place (PROJECT-BOOK §2.7) so the Go and TypeScript sides never disagree on a
// magic string. The frontend subscribes to these names.
const (
	// EventAppReady is emitted once at startup, carrying the running version.
	EventAppReady = "app:ready"
)
