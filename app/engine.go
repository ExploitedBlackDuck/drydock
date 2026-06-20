package app

import (
	"context"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// requestTimeout bounds a single engine request initiated from the UI.
const requestTimeout = 15 * time.Second

// ListContainers returns the containers on the given connected host.
func (a *App) ListContainers(hostID string) ([]domain.Container, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return nil, err
	}
	return eng.ListContainers(ctx)
}

// ListImages returns the images on the given connected host.
func (a *App) ListImages(hostID string) ([]domain.Image, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return nil, err
	}
	return eng.ListImages(ctx)
}

// ListVolumes returns the volumes on the given connected host.
func (a *App) ListVolumes(hostID string) ([]domain.Volume, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return nil, err
	}
	return eng.ListVolumes(ctx)
}

// ListNetworks returns the networks on the given connected host.
func (a *App) ListNetworks(hostID string) ([]domain.Network, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return nil, err
	}
	return eng.ListNetworks(ctx)
}

func (a *App) requestCtx() (context.Context, context.CancelFunc) {
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	return context.WithTimeout(base, requestTimeout)
}
