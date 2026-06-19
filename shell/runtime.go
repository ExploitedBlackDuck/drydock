// Package shell isolates the desktop framework's runtime behind Drydock's own
// interface (ADR-0001). Only this package and app/ import Wails; confining the
// runtime here keeps the eventual Wails v2 -> v3 migration contained to two
// packages and lets the binding layer be tested against a fake runtime.
package shell

import "context"

// Runtime is the subset of the desktop shell runtime that Drydock uses to push
// data toward the frontend. It is implemented by WailsRuntime in production and
// is trivially fakeable in tests.
type Runtime interface {
	// EmitEvent sends a named, typed event to the frontend. The context is the
	// one delivered to the application at startup.
	EmitEvent(ctx context.Context, name string, data ...any)
}
