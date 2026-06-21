package domain

// Reach classifies how far a published port is reachable, derived from the host
// IP it is bound to (PROJECT-BOOK §7.12.3, ADR-0017). It is daemon-layer insight:
// it reports the binding, not the host's actual reachable addresses (an upstream
// firewall or security group is invisible to Drydock).
type Reach string

const (
	// ReachLoopback is bound to 127.0.0.0/8 or ::1 — reachable only on the host.
	ReachLoopback Reach = "loopback"
	// ReachPrivate is bound to a specific non-loopback address (a LAN interface).
	ReachPrivate Reach = "private"
	// ReachAllInterfaces is bound to 0.0.0.0, ::, or an empty host IP — every
	// interface, which on a public-IP host is the public internet.
	ReachAllInterfaces Reach = "all_interfaces"
)

// PortBinding is one published container port and its computed reach.
type PortBinding struct {
	HostRef       string
	ContainerID   string
	ContainerName string
	HostIP        string
	HostPort      uint16
	ContainerPort uint16
	Protocol      string
	Reach         Reach
	// Flagged marks an all-interfaces binding on a host reached over a
	// non-loopback transport — plausibly reachable from outside the host, the
	// case the operator most needs to see.
	Flagged bool
}

// HostNetworkRef names a container sharing the host's network namespace
// (`network_mode: host`). Such a container publishes no bindings yet is fully
// on the host network, so its exposure is **not derivable from port bindings**;
// it is surfaced explicitly rather than shown as exposing nothing (the dangerous
// false negative, ADR-0017).
type HostNetworkRef struct {
	ContainerID   string
	ContainerName string
}

// ExposureMap is a host's computed, fleet-aggregable view of what its containers
// publish (PROJECT-BOOK §7.12.3). Read-only insight: Drydock never edits a
// firewall or rebinds a port.
type ExposureMap struct {
	HostRef string
	// RemoteTransport is true when the host is reached over a non-loopback
	// transport (SSH/TLS), making all-interfaces bindings plausibly internet-
	// reachable and therefore flagged.
	RemoteTransport bool
	Bindings        []PortBinding
	HostNetwork     []HostNetworkRef
}
