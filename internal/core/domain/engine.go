package domain

import "time"

// EngineInfo describes a connected engine and the negotiated API version
// (ADR-0008). Degraded is true when the engine is below Drydock's minimum
// supported API version, in which case some capabilities are unavailable.
type EngineInfo struct {
	EngineVersion string
	APIVersion    string
	OS            string
	Arch          string
	Degraded      bool
}

// Port is a container port mapping.
type Port struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	// Protocol is "tcp", "udp", or "sctp".
	Protocol string
}

// Container is a container as listed by the engine (PROJECT-BOOK §7.1). HostRef
// is stamped by the adapter for the host it was read from.
type Container struct {
	ID      string
	HostRef string
	Name    string
	Image   string
	State   string
	Status  string
	Ports   []Port
	// NetworkMode is the container's network mode (e.g. "host", "bridge"), used
	// to surface host-network containers in the exposure map (ADR-0017).
	NetworkMode    string
	ComposeProject string
	ComposeService string
	// Compose plan inputs (ADR-0016): the per-service config hash and where the
	// project's source lives, read from the container's Compose labels.
	ComposeConfigHash  string
	ComposeConfigFiles string // comma-separated paths, as Compose stamps them
	ComposeWorkingDir  string
	Created            time.Time
}

// Image is an image summary (PROJECT-BOOK §7.1).
type Image struct {
	ID       string
	HostRef  string
	Repo     string
	Tag      string
	Size     int64
	Dangling bool
	InUse    bool
	Created  time.Time
}

// Volume is a volume summary (PROJECT-BOOK §7.1). Size is -1 when the engine did
// not report usage data (it is only populated when explicitly requested).
type Volume struct {
	Name       string
	HostRef    string
	Driver     string
	Mountpoint string
	Size       int64
	InUse      bool
}

// Network is a network summary (PROJECT-BOOK §7.1).
type Network struct {
	ID      string
	HostRef string
	Name    string
	Driver  string
	InUse   bool
}
