// Package expose computes a host's exposure map — what its containers publish
// and how far each binding reaches (PROJECT-BOOK §7.12.3, ADR-0017). It is pure
// (no engine, no I/O, no shell) and read-only insight: it never edits a firewall
// or rebinds a port. Reach is classified at the daemon layer only — the binding,
// not the host's actual reachable addresses.
package expose

import (
	"sort"
	"strings"

	"github.com/drydock/drydock/internal/core/domain"
)

// Compute builds the exposure map for a host from its containers. remoteTransport
// is true when the host is reached over a non-loopback transport (SSH/TLS), in
// which case all-interfaces bindings are flagged as plausibly reachable from
// outside the host. It is table-tested against fixtures (the P11 gate).
func Compute(hostRef string, remoteTransport bool, containers []domain.Container) domain.ExposureMap {
	exposure := domain.ExposureMap{HostRef: hostRef, RemoteTransport: remoteTransport}

	for _, c := range containers {
		// A host-network container publishes no bindings yet shares the host's
		// namespace; surface it explicitly rather than as exposing nothing.
		if strings.EqualFold(c.NetworkMode, "host") {
			exposure.HostNetwork = append(exposure.HostNetwork, domain.HostNetworkRef{
				ContainerID:   c.ID,
				ContainerName: c.Name,
			})
			continue
		}

		for _, p := range c.Ports {
			// Only published ports (a host port is assigned) are reachable.
			if p.PublicPort == 0 {
				continue
			}
			reach := classifyReach(p.IP)
			exposure.Bindings = append(exposure.Bindings, domain.PortBinding{
				HostRef:       hostRef,
				ContainerID:   c.ID,
				ContainerName: c.Name,
				HostIP:        p.IP,
				HostPort:      p.PublicPort,
				ContainerPort: p.PrivatePort,
				Protocol:      p.Protocol,
				Reach:         reach,
				Flagged:       reach == domain.ReachAllInterfaces && remoteTransport,
			})
		}
	}

	// Most-exposed first: flagged, then all-interfaces, then by host port — so the
	// dangerous bindings surface at the top.
	sort.SliceStable(exposure.Bindings, func(i, j int) bool {
		a, b := exposure.Bindings[i], exposure.Bindings[j]
		if a.Flagged != b.Flagged {
			return a.Flagged
		}
		if reachRank(a.Reach) != reachRank(b.Reach) {
			return reachRank(a.Reach) < reachRank(b.Reach)
		}
		return a.HostPort < b.HostPort
	})
	return exposure
}

// classifyReach derives a binding's reach from its host IP.
func classifyReach(hostIP string) domain.Reach {
	switch {
	case hostIP == "" || hostIP == "0.0.0.0" || hostIP == "::":
		return domain.ReachAllInterfaces
	case hostIP == "::1" || strings.HasPrefix(hostIP, "127."):
		return domain.ReachLoopback
	default:
		return domain.ReachPrivate
	}
}

// reachRank orders reaches most-exposed first for sorting.
func reachRank(r domain.Reach) int {
	switch r {
	case domain.ReachAllInterfaces:
		return 0
	case domain.ReachPrivate:
		return 1
	default: // loopback
		return 2
	}
}
