package app

import (
	"context"
	"os"
	"strings"

	"github.com/drydock/drydock/internal/adapters/composeparse"
	"github.com/drydock/drydock/internal/core/compose"
	"github.com/drydock/drydock/internal/core/domain"
)

// Compose bindings (PROJECT-BOOK §7.11.6). Listing is a read that fetches the
// host's containers and groups them in the core; up/down route through the
// operations service, which enforces observe-mode, records, and audits.

// ListStacks groups the host's containers into Compose stacks. The grouping is
// pure core logic (internal/core/compose); this binding only fetches and forwards.
func (a *App) ListStacks(hostID string) ([]domain.Stack, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return nil, err
	}
	containers, err := eng.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	return compose.Group(containers), nil
}

// ComputeComposePlan previews what bringing a stack up would do (ADR-0016,
// §7.12.2): it reads the project's containers, builds the observed state, parses
// the project's compose source when it is locally accessible, and returns the
// classified plan. A remote (or otherwise unreadable) source yields the explicit
// source-unavailable degraded plan rather than a false "no changes".
func (a *App) ComputeComposePlan(hostID, project string) (domain.ComposePlan, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return domain.ComposePlan{}, err
	}
	containers, err := eng.ListContainers(ctx)
	if err != nil {
		return domain.ComposePlan{}, err
	}

	members := make([]domain.Container, 0)
	for _, c := range containers {
		if c.ComposeProject == project {
			members = append(members, c)
		}
	}
	return compose.Plan(project, hostID, desiredStack(ctx, project, members), observedStack(members)), nil
}

// observedStack derives the engine-observed state from a project's containers.
func observedStack(members []domain.Container) compose.ObservedStack {
	byService := map[string]*compose.ObservedService{}
	var order []string
	for _, c := range members {
		svc := c.ComposeService
		if svc == "" {
			svc = "(unknown)"
		}
		obs, ok := byService[svc]
		if !ok {
			obs = &compose.ObservedService{Name: svc, ConfigHash: c.ComposeConfigHash, Image: c.Image}
			byService[svc] = obs
			order = append(order, svc)
		}
		if c.State == "running" {
			obs.Running = true
		}
	}
	stack := compose.ObservedStack{}
	for _, svc := range order {
		stack.Services = append(stack.Services, *byService[svc])
	}
	return stack
}

// desiredStack parses the project's compose source when it is locally
// accessible, returning nil (source-unavailable) otherwise — Drydock does not
// read files off a remote host (ADR-0005/0016).
func desiredStack(ctx context.Context, project string, members []domain.Container) *compose.DesiredStack {
	var files []string
	var workingDir string
	for _, c := range members {
		if c.ComposeConfigFiles != "" {
			for _, f := range strings.Split(c.ComposeConfigFiles, ",") {
				if trimmed := strings.TrimSpace(f); trimmed != "" {
					files = append(files, trimmed)
				}
			}
			workingDir = c.ComposeWorkingDir
			break
		}
	}
	if len(files) == 0 {
		return nil
	}
	// Local accessibility: every config file must be readable on this machine.
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return nil
		}
	}
	stack, err := composeparse.Parse(ctx, project, workingDir, files)
	if err != nil {
		return nil
	}
	return &stack
}

// ApplyComposePlan recomputes the stack's plan server-side and applies it; a
// destructive plan requires ack=true. Recomputing ensures the impact recorded
// and audited matches what is applied (ADR-0016).
func (a *App) ApplyComposePlan(hostID, project string, ack bool) error {
	plan, err := a.ComputeComposePlan(hostID, project)
	if err != nil {
		return err
	}
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.ComposeApply(ctx, hostID, project, plan, ack)
}

// ComposeUp brings a stack up by starting its containers (observe-aware, recorded).
func (a *App) ComposeUp(hostID, project string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.ComposeUp(ctx, hostID, project)
}

// ComposeDown takes a stack down (destructive; ack required). When volumes is
// true it removes the stack's volumes too — compose `down -v`.
func (a *App) ComposeDown(hostID, project string, volumes, ack bool) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.ComposeDown(ctx, hostID, project, volumes, ack)
}
