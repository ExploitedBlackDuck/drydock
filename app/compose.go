package app

import (
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
