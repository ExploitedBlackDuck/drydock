package hosts_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
	"github.com/drydock/drydock/internal/core/hosts"
)

// --- fakes -----------------------------------------------------------------

type fakeStore struct {
	mu sync.Mutex
	m  map[string]domain.Host
}

func newFakeStore() *fakeStore { return &fakeStore{m: map[string]domain.Host{}} }

func (s *fakeStore) SaveHost(_ context.Context, h domain.Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[h.ID] = h
	return nil
}

func (s *fakeStore) Hosts(_ context.Context) ([]domain.Host, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Host, 0, len(s.m))
	for _, h := range s.m {
		out = append(out, h)
	}
	return out, nil
}

func (s *fakeStore) Host(_ context.Context, id string) (domain.Host, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.m[id]
	if !ok {
		return domain.Host{}, errors.New("not found")
	}
	return h, nil
}

func (s *fakeStore) DeleteHost(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
	return nil
}

func (s *fakeStore) SetHostObserveMode(_ context.Context, id string, observe bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.m[id]
	if !ok {
		return errors.New("not found")
	}
	h.ObserveMode = observe
	s.m[id] = h
	return nil
}

type fakeEngine struct {
	closed bool
}

func (e *fakeEngine) Info(context.Context) (domain.EngineInfo, error) {
	return domain.EngineInfo{EngineVersion: "28.5.2", APIVersion: "1.51"}, nil
}
func (e *fakeEngine) ListContainers(context.Context) ([]domain.Container, error) { return nil, nil }
func (e *fakeEngine) ListImages(context.Context) ([]domain.Image, error)         { return nil, nil }
func (e *fakeEngine) ListVolumes(context.Context) ([]domain.Volume, error)       { return nil, nil }
func (e *fakeEngine) ListNetworks(context.Context) ([]domain.Network, error)     { return nil, nil }
func (e *fakeEngine) StartContainer(context.Context, string) error               { return nil }
func (e *fakeEngine) StopContainer(context.Context, string) error                { return nil }
func (e *fakeEngine) RestartContainer(context.Context, string) error             { return nil }
func (e *fakeEngine) KillContainer(context.Context, string) error                { return nil }
func (e *fakeEngine) RemoveContainer(context.Context, string, engine.RemoveOptions) error {
	return nil
}

func (e *fakeEngine) ContainerLogs(context.Context, string, engine.LogOptions) (io.ReadCloser, error) {
	return nil, nil
}

func (e *fakeEngine) StreamStats(context.Context, string, func(domain.ResourceSample)) error {
	return nil
}

func (e *fakeEngine) Exec(context.Context, string, engine.ExecSpec) (engine.ExecStream, error) {
	return nil, nil
}

func (e *fakeEngine) DiskUsage(context.Context) (domain.DiskUsage, error) {
	return domain.DiskUsage{}, nil
}
func (e *fakeEngine) PruneContainers(context.Context) (int64, error)   { return 0, nil }
func (e *fakeEngine) PruneImages(context.Context, bool) (int64, error) { return 0, nil }
func (e *fakeEngine) PruneBuildCache(context.Context) (int64, error)   { return 0, nil }
func (e *fakeEngine) RemoveVolume(context.Context, string, bool) error { return nil }
func (e *fakeEngine) StreamEvents(context.Context, func(domain.EngineEvent)) error {
	return nil
}
func (e *fakeEngine) Close() error { e.closed = true; return nil }

type fakeConnector struct {
	engines []*fakeEngine
}

func (c *fakeConnector) Connect(context.Context, domain.Host) (engine.Engine, error) {
	e := &fakeEngine{}
	c.engines = append(c.engines, e)
	return e, nil
}

type fakeAuditor struct {
	records []audit.Record
}

func (a *fakeAuditor) Append(_ context.Context, r audit.Record) (domain.AuditEntry, error) {
	a.records = append(a.records, r)
	return domain.AuditEntry{}, nil
}

func newRegistry(t *testing.T) (*hosts.Registry, *fakeConnector, *fakeAuditor) {
	t.Helper()
	conn := &fakeConnector{}
	aud := &fakeAuditor{}
	return hosts.New(newFakeStore(), conn, aud, nil), conn, aud
}

// --- tests -----------------------------------------------------------------

func TestAddDerivesTrustAndPersists(t *testing.T) {
	ctx := context.Background()
	reg, _, _ := newRegistry(t)

	h, err := reg.Add(ctx, domain.Host{Name: "insecure", Transport: domain.TransportSSH, Endpoint: "tcp://example-host:2375"})
	require.NoError(t, err)
	assert.NotEmpty(t, h.ID, "an id is generated")
	assert.Equal(t, domain.TrustUntrusted, h.Trust, "plain TCP endpoint is untrusted")

	list := reg.List()
	require.Len(t, list, 1)
	assert.False(t, list[0].Connected)
}

func TestConnectDisconnectLifecycle(t *testing.T) {
	ctx := context.Background()
	reg, conn, aud := newRegistry(t)

	h, err := reg.Add(ctx, domain.Host{Name: "h", Transport: domain.TransportSSH, Endpoint: "ssh://user@host"})
	require.NoError(t, err)

	connected, err := reg.Connect(ctx, h.ID)
	require.NoError(t, err)
	assert.Equal(t, "28.5.2", connected.EngineVersion, "version recorded from Info")

	eng, err := reg.Engine(h.ID)
	require.NoError(t, err)
	assert.NotNil(t, eng)

	require.NoError(t, reg.Disconnect(ctx, h.ID))
	require.Len(t, conn.engines, 1)
	assert.True(t, conn.engines[0].closed, "disconnect closes the engine/tunnel")

	_, err = reg.Engine(h.ID)
	assert.ErrorIs(t, err, hosts.ErrNotConnected)

	// connect + disconnect both audited.
	actions := []domain.Action{aud.records[0].Action, aud.records[1].Action}
	assert.Equal(t, []domain.Action{domain.ActionHostConnect, domain.ActionHostDisconnect}, actions)
}

func TestMutateRejectsObserveModeBeforeEngine(t *testing.T) {
	ctx := context.Background()
	reg, _, _ := newRegistry(t)

	h, err := reg.Add(ctx, domain.Host{Name: "prod", Transport: domain.TransportSSH, Endpoint: "ssh://user@host", ObserveMode: true})
	require.NoError(t, err)
	_, err = reg.Connect(ctx, h.ID)
	require.NoError(t, err)

	reached := false
	err = reg.Mutate(ctx, h.ID, func(context.Context, engine.Engine) error {
		reached = true
		return nil
	})

	assert.ErrorIs(t, err, domain.ErrObserveMode)
	assert.False(t, reached, "the engine is never touched on an observe-mode host")
}

func TestMutateRunsWhenMutable(t *testing.T) {
	ctx := context.Background()
	reg, _, _ := newRegistry(t)

	h, err := reg.Add(ctx, domain.Host{Name: "h", Transport: domain.TransportSSH, Endpoint: "ssh://user@host"})
	require.NoError(t, err)
	_, err = reg.Connect(ctx, h.ID)
	require.NoError(t, err)

	reached := false
	err = reg.Mutate(ctx, h.ID, func(context.Context, engine.Engine) error {
		reached = true
		return nil
	})
	require.NoError(t, err)
	assert.True(t, reached)
}

func TestMutateOnObserveModeSkipsConnectionRequirement(t *testing.T) {
	ctx := context.Background()
	reg, _, _ := newRegistry(t)

	// Observe-mode must reject even when the host is not connected — the guard
	// is about intent, checked before anything else.
	h, err := reg.Add(ctx, domain.Host{Name: "p", Transport: domain.TransportSSH, Endpoint: "ssh://u@h", ObserveMode: true})
	require.NoError(t, err)

	err = reg.Mutate(ctx, h.ID, func(context.Context, engine.Engine) error { return nil })
	assert.ErrorIs(t, err, domain.ErrObserveMode)
}

func TestCloseDisconnectsAllTunnels(t *testing.T) {
	ctx := context.Background()
	reg, conn, _ := newRegistry(t)

	for _, name := range []string{"a", "b", "c"} {
		h, err := reg.Add(ctx, domain.Host{Name: name, Transport: domain.TransportSSH, Endpoint: "ssh://user@host"})
		require.NoError(t, err)
		_, err = reg.Connect(ctx, h.ID)
		require.NoError(t, err)
	}

	require.NoError(t, reg.Close(ctx))
	require.Len(t, conn.engines, 3)
	for _, e := range conn.engines {
		assert.True(t, e.closed, "every tunnel closed on shutdown")
	}
}

func TestSetObserveModeAudited(t *testing.T) {
	ctx := context.Background()
	reg, _, aud := newRegistry(t)

	h, err := reg.Add(ctx, domain.Host{Name: "h", Transport: domain.TransportSSH, Endpoint: "ssh://user@host"})
	require.NoError(t, err)

	require.NoError(t, reg.SetObserveMode(ctx, h.ID, true))
	require.NoError(t, reg.SetObserveMode(ctx, h.ID, false))

	require.Len(t, aud.records, 2)
	assert.Equal(t, domain.ActionObserveEnabled, aud.records[0].Action)
	assert.Equal(t, domain.ActionObserveDisabled, aud.records[1].Action)
}
