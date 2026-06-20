package sshdialer

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		user     string
		addr     string
		wantErr  bool
	}{
		{"full url", "ssh://deploy@example-host:2222", "deploy", "example-host:2222", false},
		{"url default port", "ssh://example-host", "", "example-host:22", false},
		{"bare user host", "deploy@example-host", "deploy", "example-host:22", false},
		{"bare host", "example-host", "", "example-host:22", false},
		{"empty", "", "", "", true},
		{"wrong scheme", "tcp://example-host:2375", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, addr, err := ParseEndpoint(tt.endpoint)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.user, user)
			assert.Equal(t, tt.addr, addr)
		})
	}
}

func TestAuthMethodsFromIdentityFile(t *testing.T) {
	keyPath := writeTestKey(t)

	methods, err := authMethods(Config{UseAgent: false, IdentityFiles: []string{keyPath}})
	require.NoError(t, err)
	assert.Len(t, methods, 1)
}

func TestAuthMethodsRequiresSomeAuth(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, err := authMethods(Config{UseAgent: false})
	assert.Error(t, err, "no agent and no identity files must error rather than connect unauthenticated")
}

func TestAuthMethodsSkipsUnreadableIdentity(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, err := authMethods(Config{
		UseAgent:      false,
		IdentityFiles: []string{filepath.Join(t.TempDir(), "absent")},
	})
	assert.Error(t, err, "an unreadable key yields no methods")
}

func TestHostKeyCallbackRequiresKnownHosts(t *testing.T) {
	_, err := hostKeyCallback([]string{filepath.Join(t.TempDir(), "absent")})
	assert.Error(t, err, "missing known_hosts must not silently trust the host")
}

func TestHostKeyCallbackLoadsKnownHosts(t *testing.T) {
	khPath := writeKnownHosts(t)
	cb, err := hostKeyCallback([]string{khPath})
	require.NoError(t, err)
	assert.NotNil(t, cb)
}

// writeTestKey generates an ed25519 private key and writes it as an OpenSSH PEM.
func writeTestKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	block, err := ssh.MarshalPrivateKey(priv, "")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "id_ed25519")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))
	return path
}

// writeKnownHosts writes a single valid known_hosts entry.
func writeKnownHosts(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signerPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)

	line := "example-host " + string(ssh.MarshalAuthorizedKey(signerPub))
	path := filepath.Join(t.TempDir(), "known_hosts")
	require.NoError(t, os.WriteFile(path, []byte(line), 0o600))
	return path
}
