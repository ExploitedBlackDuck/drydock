package expose_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/expose"
)

func bindOn(ip string) domain.Container {
	return domain.Container{
		ID:    "c-" + ip,
		Name:  "svc",
		Ports: []domain.Port{{IP: ip, PublicPort: 8080, PrivatePort: 80, Protocol: "tcp"}},
	}
}

func TestClassifyReachAcrossHostIPs(t *testing.T) {
	cases := map[string]domain.Reach{
		"127.0.0.1":   domain.ReachLoopback,
		"127.0.0.53":  domain.ReachLoopback,
		"::1":         domain.ReachLoopback,
		"0.0.0.0":     domain.ReachAllInterfaces,
		"::":          domain.ReachAllInterfaces,
		"":            domain.ReachAllInterfaces,
		"192.168.1.5": domain.ReachPrivate,
		"10.0.0.4":    domain.ReachPrivate,
	}
	for ip, want := range cases {
		m := expose.Compute("h", false, []domain.Container{bindOn(ip)})
		require.Len(t, m.Bindings, 1, "ip %q", ip)
		assert.Equal(t, want, m.Bindings[0].Reach, "ip %q", ip)
	}
}

func TestAllInterfacesFlaggedOnRemoteTransport(t *testing.T) {
	containers := []domain.Container{bindOn("0.0.0.0")}

	remote := expose.Compute("ssh-host", true, containers)
	require.Len(t, remote.Bindings, 1)
	assert.True(t, remote.Bindings[0].Flagged, "all-interfaces on a remote host is flagged")

	local := expose.Compute("local", false, containers)
	assert.False(t, local.Bindings[0].Flagged, "all-interfaces on a loopback-transport host is not flagged")
	assert.Equal(t, domain.ReachAllInterfaces, local.Bindings[0].Reach, "but the reach is still reported")
}

func TestHostNetworkListedNotShownAsNoExposure(t *testing.T) {
	containers := []domain.Container{
		{ID: "h1", Name: "monitor", NetworkMode: "host"}, // no published ports
	}
	m := expose.Compute("h", true, containers)

	assert.Empty(t, m.Bindings, "a host-network container has no derivable bindings")
	require.Len(t, m.HostNetwork, 1, "but it is surfaced explicitly, not as exposing nothing")
	assert.Equal(t, "monitor", m.HostNetwork[0].ContainerName)
}

func TestUnpublishedPortsAreNotExposed(t *testing.T) {
	containers := []domain.Container{{
		ID:    "c1",
		Name:  "internal",
		Ports: []domain.Port{{IP: "", PublicPort: 0, PrivatePort: 5432, Protocol: "tcp"}},
	}}
	m := expose.Compute("h", true, containers)
	assert.Empty(t, m.Bindings, "a container port with no host port is not reachable")
}

func TestBindingsSortedMostExposedFirst(t *testing.T) {
	containers := []domain.Container{
		{ID: "a", Name: "loop", Ports: []domain.Port{{IP: "127.0.0.1", PublicPort: 3000, PrivatePort: 3000, Protocol: "tcp"}}},
		{ID: "b", Name: "open", Ports: []domain.Port{{IP: "0.0.0.0", PublicPort: 9000, PrivatePort: 9000, Protocol: "tcp"}}},
		{ID: "c", Name: "lan", Ports: []domain.Port{{IP: "192.168.0.2", PublicPort: 6000, PrivatePort: 6000, Protocol: "tcp"}}},
	}
	m := expose.Compute("ssh-host", true, containers)
	require.Len(t, m.Bindings, 3)
	// Flagged all-interfaces first, then private, then loopback.
	assert.Equal(t, domain.ReachAllInterfaces, m.Bindings[0].Reach)
	assert.True(t, m.Bindings[0].Flagged)
	assert.Equal(t, domain.ReachPrivate, m.Bindings[1].Reach)
	assert.Equal(t, domain.ReachLoopback, m.Bindings[2].Reach)
}
