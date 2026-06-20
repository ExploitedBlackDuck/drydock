package dockerengine

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

// composeProjectFilter builds the label filter that selects a Compose project's
// objects. The project name is a structured label value passed to the SDK —
// never interpolated into a shell or a command line (ADR-0004).
func composeProjectFilter(project string) filters.Args {
	return filters.NewArgs(filters.Arg("label", composeProjectLabel+"="+project))
}

// ComposeUp brings a stack up by starting every container in the project that is
// not already running. Drydock acts on the containers Compose created; it does
// not recreate them from a compose file (which it need not have), so "up" here
// means start the stack's existing containers.
func (c *Client) ComposeUp(ctx context.Context, project string) error {
	summaries, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: composeProjectFilter(project),
	})
	if err != nil {
		return fmt.Errorf("listing containers for project %q on host %q: %w", project, c.hostRef, err)
	}
	for _, s := range summaries {
		if s.State == "running" {
			continue
		}
		if err := c.cli.ContainerStart(ctx, s.ID, container.StartOptions{}); err != nil {
			return fmt.Errorf("starting %q in project %q: %w", s.ID, project, err)
		}
	}
	return nil
}

// ComposeDown stops and removes the project's containers. When volumes is true
// it then removes the project's named volumes (compose `down -v`); that branch
// is reached only after the operation was acknowledged upstream (§7.5/§7.11.6).
func (c *Client) ComposeDown(ctx context.Context, project string, volumes bool) error {
	summaries, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: composeProjectFilter(project),
	})
	if err != nil {
		return fmt.Errorf("listing containers for project %q on host %q: %w", project, c.hostRef, err)
	}
	for _, s := range summaries {
		if err := c.cli.ContainerStop(ctx, s.ID, container.StopOptions{}); err != nil {
			return fmt.Errorf("stopping %q in project %q: %w", s.ID, project, err)
		}
		if err := c.cli.ContainerRemove(ctx, s.ID, container.RemoveOptions{}); err != nil {
			return fmt.Errorf("removing %q in project %q: %w", s.ID, project, err)
		}
	}
	if !volumes {
		return nil
	}

	listed, err := c.cli.VolumeList(ctx, volume.ListOptions{Filters: composeProjectFilter(project)})
	if err != nil {
		return fmt.Errorf("listing volumes for project %q on host %q: %w", project, c.hostRef, err)
	}
	for _, v := range listed.Volumes {
		if err := c.cli.VolumeRemove(ctx, v.Name, false); err != nil {
			return fmt.Errorf("removing volume %q in project %q: %w", v.Name, project, err)
		}
	}
	return nil
}
