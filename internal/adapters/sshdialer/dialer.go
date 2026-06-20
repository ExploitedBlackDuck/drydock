// Package sshdialer dials a remote Docker Unix socket over SSH (ADR-0005). It is
// the safe-by-default remote transport: it reuses the operator's existing keys
// and agent (referenced, never copied — ADR-0009) and verifies host keys against
// known_hosts. The connection is supervised — a keepalive runs until Close,
// which tears the tunnel down with no leaked goroutine or socket.
package sshdialer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	defaultPort         = "22"
	defaultRemoteSocket = "/var/run/docker.sock"
	defaultTimeout      = 15 * time.Second
	keepaliveInterval   = 30 * time.Second
)

// Config describes how to reach a host over SSH.
type Config struct {
	// Endpoint is "ssh://[user@]host[:port]" or "[user@]host[:port]".
	Endpoint string
	// User overrides the endpoint user; falls back to the endpoint, then the
	// current OS user.
	User string
	// IdentityFiles are paths to private keys to try (referenced, not copied).
	IdentityFiles []string
	// KnownHostsFiles override the default ~/.ssh/known_hosts for host-key
	// verification.
	KnownHostsFiles []string
	// RemoteSocket is the engine socket on the remote host.
	RemoteSocket string
	// Timeout bounds the TCP+handshake.
	Timeout time.Duration
	// UseAgent enables the SSH agent (default true via New).
	UseAgent bool
}

// Dialer holds a live SSH connection and dials the remote engine socket over it.
type Dialer struct {
	client       *ssh.Client
	remoteSocket string
	done         chan struct{}
}

// ParseEndpoint extracts the SSH user and host:port from an endpoint string.
func ParseEndpoint(endpoint string) (sshUser, addr string, err error) {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return "", "", errors.New("empty ssh endpoint")
	}
	if !strings.Contains(raw, "://") {
		raw = "ssh://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parsing ssh endpoint %q: %w", endpoint, err)
	}
	if u.Scheme != "ssh" {
		return "", "", fmt.Errorf("not an ssh endpoint: %q", endpoint)
	}
	host := u.Hostname()
	if host == "" {
		return "", "", fmt.Errorf("ssh endpoint %q has no host", endpoint)
	}
	port := u.Port()
	if port == "" {
		port = defaultPort
	}
	return u.User.Username(), net.JoinHostPort(host, port), nil
}

// New establishes the SSH connection described by cfg.
func New(ctx context.Context, cfg Config) (*Dialer, error) {
	endpointUser, addr, err := ParseEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	sshUser, err := resolveUser(cfg.User, endpointUser)
	if err != nil {
		return nil, err
	}

	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}

	hostKeys, err := hostKeyCallback(cfg.KnownHostsFiles)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	clientConfig := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            auth,
		HostKeyCallback: hostKeys,
		Timeout:         timeout,
	}

	netDialer := net.Dialer{Timeout: timeout}
	conn, err := netDialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh handshake with %s: %w", addr, err)
	}

	socket := cfg.RemoteSocket
	if socket == "" {
		socket = defaultRemoteSocket
	}

	d := &Dialer{
		client:       ssh.NewClient(sshConn, chans, reqs),
		remoteSocket: socket,
		done:         make(chan struct{}),
	}
	go d.keepalive()
	return d, nil
}

// DialContext opens a connection to the remote engine socket over the SSH
// tunnel. The network/addr arguments (from the HTTP transport) are ignored: the
// destination is always the configured remote socket.
func (d *Dialer) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	conn, err := d.client.Dial("unix", d.remoteSocket)
	if err != nil {
		return nil, fmt.Errorf("dialing remote socket %s: %w", d.remoteSocket, err)
	}
	return conn, nil
}

// Close stops the keepalive and tears down the SSH connection.
func (d *Dialer) Close() error {
	select {
	case <-d.done:
		// already closed
	default:
		close(d.done)
	}
	return d.client.Close()
}

// keepalive sends periodic keepalive requests so a dead tunnel is noticed
// promptly; it exits when Close signals done.
func (d *Dialer) keepalive() {
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-d.done:
			return
		case <-ticker.C:
			_, _, err := d.client.SendRequest("keepalive@drydock", true, nil)
			if err != nil {
				return
			}
		}
	}
}

func resolveUser(override, fromEndpoint string) (string, error) {
	if override != "" {
		return override, nil
	}
	if fromEndpoint != "" {
		return fromEndpoint, nil
	}
	current, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("determining current user: %w", err)
	}
	return current.Username, nil
}

// authMethods assembles SSH auth from the agent and any identity files. Keys are
// read only to obtain signers; their material is never stored by Drydock.
func authMethods(cfg Config) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if cfg.UseAgent {
		if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
			if conn, err := net.Dial("unix", sock); err == nil {
				methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
			}
		}
	}

	for _, path := range cfg.IdentityFiles {
		signer, err := loadSigner(path)
		if err != nil {
			// Skip keys we cannot use unattended (e.g. passphrase-protected);
			// the agent is the expected path for those.
			continue
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if len(methods) == 0 {
		return nil, errors.New("no usable SSH authentication (no agent and no readable identity files)")
	}
	return methods, nil
}

func loadSigner(path string) (ssh.Signer, error) {
	pem, err := os.ReadFile(path) //nolint:gosec // operator-referenced key path
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(pem)
}

// hostKeyCallback builds a known_hosts-backed verifier. With no readable
// known_hosts file, it returns an error rather than blindly trusting the host.
func hostKeyCallback(files []string) (ssh.HostKeyCallback, error) {
	if len(files) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			files = []string{filepath.Join(home, ".ssh", "known_hosts")}
		}
	}
	existing := make([]string, 0, len(files))
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			existing = append(existing, f)
		}
	}
	if len(existing) == 0 {
		return nil, errors.New("no known_hosts file found; cannot verify the host key")
	}
	cb, err := knownhosts.New(existing...)
	if err != nil {
		return nil, fmt.Errorf("loading known_hosts: %w", err)
	}
	return cb, nil
}
