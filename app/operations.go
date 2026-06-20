package app

import "github.com/drydock/drydock/internal/core/engine"

// Container lifecycle bindings. Each routes through the operations service, which
// enforces observe-mode in the core, records the operation, and audits it.
// Destructive actions require ack=true (ADR-0011/0013).

// StartContainer starts a stopped container.
func (a *App) StartContainer(hostID, containerID string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.Start(ctx, hostID, containerID)
}

// StopContainer gracefully stops a container.
func (a *App) StopContainer(hostID, containerID string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.Stop(ctx, hostID, containerID)
}

// RestartContainer restarts a container.
func (a *App) RestartContainer(hostID, containerID string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.Restart(ctx, hostID, containerID)
}

// KillContainer force-kills a container (destructive; ack required).
func (a *App) KillContainer(hostID, containerID string, ack bool) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.Kill(ctx, hostID, containerID, ack)
}

// RemoveContainer removes a container (destructive; ack required).
func (a *App) RemoveContainer(hostID, containerID string, force, volumes, ack bool) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.Remove(ctx, hostID, containerID, engine.RemoveOptions{Force: force, Volumes: volumes}, ack)
}
