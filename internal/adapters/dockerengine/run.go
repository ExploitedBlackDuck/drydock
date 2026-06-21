package dockerengine

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// RunContainer creates and starts a container from a resolved RunSpec. All
// parameters are typed and assembled into the SDK config — nothing is
// interpolated into a shell (ADR-0004). The helper that builds the config is
// pure and unit-tested.
func (c *Client) RunContainer(ctx context.Context, spec domain.RunSpec) (string, error) {
	config, hostConfig, err := buildRunConfig(spec)
	if err != nil {
		return "", err
	}
	created, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, spec.Name)
	if err != nil {
		return "", fmt.Errorf("creating container from image %q on host %q: %w", spec.Image, c.hostRef, err)
	}
	if err := c.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		// Leave the created-but-unstarted container for the operator to inspect.
		return created.ID, fmt.Errorf("starting container %q: %w", created.ID, err)
	}
	return created.ID, nil
}

// buildRunConfig assembles the SDK config from a RunSpec. Pure (no client), so it
// is table-tested independently of a daemon.
func buildRunConfig(spec domain.RunSpec) (*container.Config, *container.HostConfig, error) {
	exposed, bindings, err := nat.ParsePortSpecs(spec.Publish)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing published ports: %w", err)
	}

	config := &container.Config{
		Image:        spec.Image,
		Cmd:          spec.Command,
		Env:          spec.Env,
		User:         spec.User,
		WorkingDir:   spec.WorkingDir,
		ExposedPorts: exposed,
	}
	hostConfig := &container.HostConfig{
		Binds:        spec.Volumes,
		PortBindings: bindings,
	}
	if spec.Restart != "" && spec.Restart != "no" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(spec.Restart)}
	}
	if spec.NetworkHost {
		hostConfig.NetworkMode = "host"
	}
	return config, hostConfig, nil
}

// Compile-time assertion the engine still satisfies the port after the addition.
var _ engine.Engine = (*Client)(nil)
