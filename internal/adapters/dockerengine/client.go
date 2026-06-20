// Package dockerengine implements the engine.Engine port using the official
// Docker Go SDK (ADR-0003). It negotiates the API version per host (ADR-0008),
// streams nothing in this read-only phase, and translates SDK types to domain
// types via the pure mappers in this package. The same client speaks to any
// API-compatible engine (e.g. a Podman socket) reached through the connection it
// is given.
package dockerengine

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	dockerversions "github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// Client is a Docker-SDK-backed engine bound to a single host.
type Client struct {
	cli     *client.Client
	hostRef string
}

type options struct {
	host       string
	clientOpts []client.Opt
}

// Option configures the engine client.
type Option func(*options)

// WithHost sets an explicit engine host (e.g. "unix:///var/run/docker.sock").
// When unset, the environment (DOCKER_HOST and friends) is used.
func WithHost(host string) Option {
	return func(o *options) { o.host = host }
}

// WithClientOptions injects additional SDK options, used by the transport
// adapters (P3) to supply an SSH-dialed connection.
func WithClientOptions(opts ...client.Opt) Option {
	return func(o *options) { o.clientOpts = append(o.clientOpts, opts...) }
}

// Open constructs an engine client for hostRef. API-version negotiation happens
// lazily on the first request.
func Open(hostRef string, opts ...Option) (*Client, error) {
	var cfg options
	for _, opt := range opts {
		opt(&cfg)
	}

	clientOpts := []client.Opt{client.WithAPIVersionNegotiation()}
	if cfg.host != "" {
		clientOpts = append(clientOpts, client.WithHost(cfg.host))
	} else {
		clientOpts = append(clientOpts, client.FromEnv)
	}
	clientOpts = append(clientOpts, cfg.clientOpts...)

	cli, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating docker client for host %q: %w", hostRef, err)
	}
	return &Client{cli: cli, hostRef: hostRef}, nil
}

// Info returns the engine and negotiated API version, flagging reduced
// capability when the engine is below the minimum supported version (ADR-0008).
func (c *Client) Info(ctx context.Context) (domain.EngineInfo, error) {
	v, err := c.cli.ServerVersion(ctx)
	if err != nil {
		return domain.EngineInfo{}, fmt.Errorf("querying engine version on host %q: %w", c.hostRef, err)
	}
	return domain.EngineInfo{
		EngineVersion: v.Version,
		APIVersion:    v.APIVersion,
		OS:            v.Os,
		Arch:          v.Arch,
		Degraded:      dockerversions.LessThan(v.APIVersion, engine.MinAPIVersion),
	}, nil
}

// ListContainers lists all containers, running and stopped.
func (c *Client) ListContainers(ctx context.Context) ([]domain.Container, error) {
	summaries, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("listing containers on host %q: %w", c.hostRef, err)
	}
	out := make([]domain.Container, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, mapContainer(c.hostRef, s))
	}
	return out, nil
}

// ListImages lists images held on the engine.
func (c *Client) ListImages(ctx context.Context) ([]domain.Image, error) {
	summaries, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing images on host %q: %w", c.hostRef, err)
	}
	out := make([]domain.Image, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, mapImage(c.hostRef, s))
	}
	return out, nil
}

// ListVolumes lists volumes known to the engine.
func (c *Client) ListVolumes(ctx context.Context) ([]domain.Volume, error) {
	resp, err := c.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing volumes on host %q: %w", c.hostRef, err)
	}
	out := make([]domain.Volume, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		out = append(out, mapVolume(c.hostRef, v))
	}
	return out, nil
}

// ListNetworks lists networks known to the engine.
func (c *Client) ListNetworks(ctx context.Context) ([]domain.Network, error) {
	summaries, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing networks on host %q: %w", c.hostRef, err)
	}
	out := make([]domain.Network, 0, len(summaries))
	for _, n := range summaries {
		out = append(out, mapNetwork(c.hostRef, n))
	}
	return out, nil
}

// Close releases the underlying connection and idle transport connections.
func (c *Client) Close() error {
	if err := c.cli.Close(); err != nil {
		return fmt.Errorf("closing docker client for host %q: %w", c.hostRef, err)
	}
	return nil
}

// Compile-time assertion that Client satisfies the port.
var _ engine.Engine = (*Client)(nil)
