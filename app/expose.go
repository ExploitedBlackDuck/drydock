package app

import (
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/expose"
)

// HostExposure computes the host's exposure map (PROJECT-BOOK §7.12.3, ADR-0017):
// what its containers publish and how far each binding reaches. It is a read —
// Drydock never edits a firewall or rebinds a port. All-interfaces bindings are
// flagged when the host is reached over a non-loopback transport.
func (a *App) HostExposure(hostID string) (domain.ExposureMap, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return domain.ExposureMap{}, err
	}
	containers, err := eng.ListContainers(ctx)
	if err != nil {
		return domain.ExposureMap{}, err
	}
	return expose.Compute(hostID, a.remoteTransport(hostID), containers), nil
}

// remoteTransport reports whether a host is reached over a non-loopback
// transport (SSH/TLS), making its all-interfaces bindings plausibly reachable
// from outside the host.
func (a *App) remoteTransport(hostID string) bool {
	for _, status := range a.registry.List() {
		if status.Host.ID == hostID {
			return status.Host.Transport != domain.TransportLocal
		}
	}
	return false
}
