package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

func TestEnsureMutableRejectsObserveMode(t *testing.T) {
	observe := domain.Host{Name: "prod", Transport: domain.TransportSSH, Endpoint: "ssh://user@host", ObserveMode: true}
	err := observe.EnsureMutable()
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrObserveMode)

	mutable := observe
	mutable.ObserveMode = false
	assert.NoError(t, mutable.EnsureMutable())
}

func TestTrustForAndInsecureDetection(t *testing.T) {
	tests := []struct {
		name      string
		transport domain.Transport
		endpoint  string
		insecure  bool
		trust     domain.Trust
	}{
		{"local socket", domain.TransportLocal, "unix:///var/run/docker.sock", false, domain.TrustTrusted},
		{"ssh", domain.TransportSSH, "ssh://user@host", false, domain.TrustTrusted},
		// A TCP endpoint is a TCP endpoint by scheme; the TLS transport is what
		// makes it trusted (mTLS), so TrustFor returns trusted there.
		{"mtls over tcp", domain.TransportTLS, "tcp://host:2376", true, domain.TrustTrusted},
		{"plain tcp without tls transport", domain.TransportSSH, "tcp://host:2375", true, domain.TrustUntrusted},
		{"http endpoint", domain.TransportSSH, "http://host:2375", true, domain.TrustUntrusted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.insecure, domain.IsInsecureTCP(tt.endpoint))
			assert.Equal(t, tt.trust, domain.TrustFor(tt.transport, tt.endpoint))
		})
	}
}

func TestHostValidate(t *testing.T) {
	valid := domain.Host{Name: "h", Transport: domain.TransportLocal, Endpoint: "unix:///x"}
	assert.NoError(t, valid.Validate())

	assert.Error(t, domain.Host{Transport: domain.TransportLocal, Endpoint: "x"}.Validate())
	assert.Error(t, domain.Host{Name: "h", Transport: "bogus", Endpoint: "x"}.Validate())
	assert.Error(t, domain.Host{Name: "h", Transport: domain.TransportLocal}.Validate())
}
