package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Transport is how Drydock reaches a host's engine (ADR-0005).
type Transport string

const (
	// TransportLocal is the local Unix socket.
	TransportLocal Transport = "local"
	// TransportSSH dials the remote Unix socket over SSH (the default for remote).
	TransportSSH Transport = "ssh"
	// TransportTLS is mTLS-over-TCP, for hosts already configured that way.
	TransportTLS Transport = "tls"
)

// Valid reports whether the transport is a recognized kind.
func (t Transport) Valid() bool {
	switch t {
	case TransportLocal, TransportSSH, TransportTLS:
		return true
	default:
		return false
	}
}

// Trust describes how much the transport can be relied upon (ADR-0005).
type Trust string

const (
	// TrustTrusted is a transport with proper authentication (local, SSH, mTLS).
	TrustTrusted Trust = "trusted"
	// TrustUntrusted is an unauthenticated TCP socket — a root-equivalent
	// backdoor. Drydock never creates one and flags any it detects.
	TrustUntrusted Trust = "untrusted"
)

// ErrObserveMode is returned when a mutating operation targets an observe-only
// host. It is checked in the core before any request reaches the engine
// (ADR-0013). Maps to ERR_OBSERVE_MODE (PROJECT-BOOK §8.4).
var ErrObserveMode = errors.New("host is in observe-only mode")

// Host is a connection profile for a Docker engine (PROJECT-BOOK §7.1). Trust is
// derived from the transport and endpoint, not stored.
type Host struct {
	ID            string
	Name          string
	Transport     Transport
	Endpoint      string
	ObserveMode   bool
	Trust         Trust
	EngineVersion string
	APIVersion    string
}

// Validate checks the profile's invariants.
func (h Host) Validate() error {
	if strings.TrimSpace(h.Name) == "" {
		return errors.New("host name is required")
	}
	if !h.Transport.Valid() {
		return fmt.Errorf("unknown transport %q", h.Transport)
	}
	if strings.TrimSpace(h.Endpoint) == "" {
		return errors.New("host endpoint is required")
	}
	return nil
}

// EnsureMutable returns ErrObserveMode when the host is observe-only, and is the
// single guard every mutating operation passes through in the core (ADR-0013).
func (h Host) EnsureMutable() error {
	if h.ObserveMode {
		return fmt.Errorf("%w: %q", ErrObserveMode, h.Name)
	}
	return nil
}

// TrustFor classifies a transport/endpoint pair. An unauthenticated TCP endpoint
// (tcp:// or http:// without mTLS) is untrusted; everything else is trusted
// (ADR-0005). Drydock surfaces a warning for untrusted hosts and never creates
// one itself.
func TrustFor(transport Transport, endpoint string) Trust {
	if transport == TransportTLS {
		return TrustTrusted
	}
	if IsInsecureTCP(endpoint) {
		return TrustUntrusted
	}
	return TrustTrusted
}

// IsInsecureTCP reports whether endpoint is a plain (non-TLS) TCP socket — the
// internet-scannable, root-equivalent backdoor Drydock refuses to encourage.
func IsInsecureTCP(endpoint string) bool {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "tcp", "http":
		return true
	default:
		return false
	}
}
