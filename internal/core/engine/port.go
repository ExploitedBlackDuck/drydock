// Package engine defines the Engine port — Drydock's interface to a Docker (or
// API-compatible) engine — and the read-only services over it. The port is
// declared here, by its consumer (PROJECT-BOOK §2.1); the Docker Go SDK
// implementation lives in the dockerengine adapter (ADR-0003). The core knows
// nothing of the SDK's types.
package engine

import (
	"context"

	"github.com/drydock/drydock/internal/core/domain"
)

// MinAPIVersion is the lowest Docker Engine API version Drydock fully supports.
// A host below it connects in a clearly-labelled reduced-capability mode rather
// than failing opaquely (ADR-0008).
const MinAPIVersion = "1.41"

// Engine is the port for a single connected engine. All methods take a context
// so cancellation (UI "stop"/"disconnect") propagates to the request and the
// underlying connection (PROJECT-BOOK §2.3). Read-only in this phase; mutating
// methods are added in P4.
type Engine interface {
	// Info returns the engine and negotiated API version (ADR-0008).
	Info(ctx context.Context) (domain.EngineInfo, error)
	// ListContainers lists all containers, running and stopped.
	ListContainers(ctx context.Context) ([]domain.Container, error)
	// ListImages lists images held on the engine.
	ListImages(ctx context.Context) ([]domain.Image, error)
	// ListVolumes lists volumes known to the engine.
	ListVolumes(ctx context.Context) ([]domain.Volume, error)
	// ListNetworks lists networks known to the engine.
	ListNetworks(ctx context.Context) ([]domain.Network, error)
	// Close releases the engine connection and any associated resources.
	Close() error
}
