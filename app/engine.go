package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// requestTimeout bounds a single engine request initiated from the UI.
const requestTimeout = 15 * time.Second

// ErrUnknownHost is returned when the frontend asks for a host that is not a
// connected engine. In this phase only the local engine is connected; the hosts
// registry (P3) generalizes this.
var ErrUnknownHost = errors.New("host is not connected")

// LocalEngineStatus reports whether the local Docker engine is reachable and,
// if so, its negotiated version. It is the typed shape delivered to the
// frontend so the local host can be presented in the host switcher.
type LocalEngineStatus struct {
	Available     bool   `json:"available"`
	HostID        string `json:"hostId"`
	EngineVersion string `json:"engineVersion"`
	APIVersion    string `json:"apiVersion"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Degraded      bool   `json:"degraded"`
}

// LocalEngine probes the local engine and reports its status. An unreachable
// engine is a normal, non-error outcome (the UI shows "no hosts").
func (a *App) LocalEngine() LocalEngineStatus {
	ctx, cancel := a.requestCtx()
	defer cancel()

	info, err := a.engine.Info(ctx)
	if err != nil {
		a.log.Info("local engine unavailable", slog.Any("error", err))
		return LocalEngineStatus{Available: false, HostID: engine.LocalHostID}
	}
	return LocalEngineStatus{
		Available:     true,
		HostID:        engine.LocalHostID,
		EngineVersion: info.EngineVersion,
		APIVersion:    info.APIVersion,
		OS:            info.OS,
		Arch:          info.Arch,
		Degraded:      info.Degraded,
	}
}

// ListContainers returns the containers on the given host.
func (a *App) ListContainers(hostID string) ([]domain.Container, error) {
	if err := requireLocal(hostID); err != nil {
		return nil, err
	}
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.engine.ListContainers(ctx)
}

// ListImages returns the images on the given host.
func (a *App) ListImages(hostID string) ([]domain.Image, error) {
	if err := requireLocal(hostID); err != nil {
		return nil, err
	}
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.engine.ListImages(ctx)
}

// ListVolumes returns the volumes on the given host.
func (a *App) ListVolumes(hostID string) ([]domain.Volume, error) {
	if err := requireLocal(hostID); err != nil {
		return nil, err
	}
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.engine.ListVolumes(ctx)
}

// ListNetworks returns the networks on the given host.
func (a *App) ListNetworks(hostID string) ([]domain.Network, error) {
	if err := requireLocal(hostID); err != nil {
		return nil, err
	}
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.engine.ListNetworks(ctx)
}

func requireLocal(hostID string) error {
	if hostID != engine.LocalHostID {
		return ErrUnknownHost
	}
	return nil
}

func (a *App) requestCtx() (context.Context, context.CancelFunc) {
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	return context.WithTimeout(base, requestTimeout)
}
