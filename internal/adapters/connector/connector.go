// Package connector builds engine connections for host profiles, choosing the
// transport (local socket, SSH tunnel, or mTLS-over-TCP) and wiring the Docker
// SDK client to it. It is the adapter that satisfies hosts.Connector, keeping
// the core registry free of transport details.
package connector

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"

	"github.com/drydock/drydock/internal/adapters/dockerengine"
	"github.com/drydock/drydock/internal/adapters/sshdialer"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// remoteSocket is the engine socket Drydock dials on remote hosts.
const remoteSocket = "/var/run/docker.sock"

// Connector implements hosts.Connector.
type Connector struct{}

// New returns a connector.
func New() *Connector { return &Connector{} }

// Connect opens an engine for the host using its transport.
func (c *Connector) Connect(ctx context.Context, h domain.Host) (engine.Engine, error) {
	switch h.Transport {
	case domain.TransportLocal:
		return dockerengine.Open(h.ID)

	case domain.TransportSSH:
		dialer, err := sshdialer.New(ctx, sshdialer.Config{
			Endpoint:     h.Endpoint,
			RemoteSocket: remoteSocket,
			UseAgent:     true,
		})
		if err != nil {
			return nil, fmt.Errorf("establishing SSH tunnel to %q: %w", h.Name, err)
		}
		eng, err := dockerengine.Open(h.ID,
			dockerengine.WithClientOptions(
				client.WithHost("unix://"+remoteSocket),
				client.WithDialContext(dialer.DialContext),
			),
			dockerengine.WithCloser(dialer),
		)
		if err != nil {
			_ = dialer.Close()
			return nil, err
		}
		return eng, nil

	case domain.TransportTLS:
		return nil, fmt.Errorf("mTLS transport for host %q is not yet implemented", h.Name)

	default:
		return nil, fmt.Errorf("unknown transport %q for host %q", h.Transport, h.Name)
	}
}

// Compile-time check that Connector satisfies the port consumed by the registry.
var _ interface {
	Connect(ctx context.Context, h domain.Host) (engine.Engine, error)
} = (*Connector)(nil)
