package operations_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
	"github.com/drydock/drydock/internal/core/operations"
	"github.com/drydock/drydock/internal/core/options"
)

// fakeEngine records which mutation was invoked.
type fakeEngine struct {
	started, stopped, removed bool
	composeUp                 bool
	composeDownVolumes        *bool
}

func (e *fakeEngine) Info(context.Context) (domain.EngineInfo, error) {
	return domain.EngineInfo{}, nil
}

func (e *fakeEngine) ListContainers(context.Context) ([]domain.Container, error) {
	return nil, nil
}
func (e *fakeEngine) ListImages(context.Context) ([]domain.Image, error)     { return nil, nil }
func (e *fakeEngine) ListVolumes(context.Context) ([]domain.Volume, error)   { return nil, nil }
func (e *fakeEngine) ListNetworks(context.Context) ([]domain.Network, error) { return nil, nil }
func (e *fakeEngine) StartContainer(context.Context, string) error           { e.started = true; return nil }

func (e *fakeEngine) StopContainer(context.Context, string) error    { e.stopped = true; return nil }
func (e *fakeEngine) RestartContainer(context.Context, string) error { return nil }
func (e *fakeEngine) KillContainer(context.Context, string) error    { return nil }
func (e *fakeEngine) RemoveContainer(context.Context, string, engine.RemoveOptions) error {
	e.removed = true
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
	return domain.DiskUsage{
		Images: []domain.DiskImage{{ID: "i", Size: 1000, Dangling: true}},
	}, nil
}
func (e *fakeEngine) PruneContainers(context.Context) (int64, error) { return 0, nil }
func (e *fakeEngine) PruneImages(context.Context, bool) (int64, error) {
	e.removed = true
	return 1000, nil
}
func (e *fakeEngine) PruneBuildCache(context.Context) (int64, error) { return 0, nil }
func (e *fakeEngine) RemoveVolume(context.Context, string, bool) error {
	e.removed = true
	return nil
}

func (e *fakeEngine) ComposeUp(context.Context, string) error { e.composeUp = true; return nil }

func (e *fakeEngine) ComposeDown(_ context.Context, _ string, volumes bool) error {
	e.composeDownVolumes = &volumes
	return nil
}

func (e *fakeEngine) StreamEvents(context.Context, func(domain.EngineEvent)) error {
	return nil
}
func (e *fakeEngine) Close() error { return nil }

// fakeMutator emulates the registry guard: when observe is set, it rejects with
// ErrObserveMode before calling fn (like hosts.Registry.Mutate).
type fakeMutator struct {
	eng     *fakeEngine
	observe bool
	called  bool
}

func (m *fakeMutator) Mutate(ctx context.Context, _ string, fn func(context.Context, engine.Engine) error) error {
	if m.observe {
		return domain.ErrObserveMode
	}
	m.called = true
	return fn(ctx, m.eng)
}

type fakeStore struct {
	ops     []domain.Operation
	impacts map[string]domain.PruneImpact
}

func (s *fakeStore) SaveOperation(_ context.Context, op domain.Operation) error {
	s.ops = append(s.ops, op)
	return nil
}

func (s *fakeStore) SavePruneImpact(_ context.Context, operationID string, impact domain.PruneImpact) error {
	if s.impacts == nil {
		s.impacts = map[string]domain.PruneImpact{}
	}
	s.impacts[operationID] = impact
	return nil
}

type fakeAuditor struct{ records []audit.Record }

func (a *fakeAuditor) Append(_ context.Context, r audit.Record) (domain.AuditEntry, error) {
	a.records = append(a.records, r)
	return domain.AuditEntry{}, nil
}

func newService(observe bool) (*operations.Service, *fakeMutator, *fakeStore, *fakeAuditor) {
	eng := &fakeEngine{}
	m := &fakeMutator{eng: eng, observe: observe}
	store := &fakeStore{}
	aud := &fakeAuditor{}
	catalog, err := options.DefaultCatalog()
	if err != nil {
		panic(err)
	}
	return operations.New(m, store, aud, catalog, nil), m, store, aud
}

func TestStartRecordsOperationAndAudit(t *testing.T) {
	svc, m, store, aud := newService(false)

	require.NoError(t, svc.Start(context.Background(), "h1", "c1"))
	assert.True(t, m.eng.started)
	require.Len(t, store.ops, 1)
	assert.Equal(t, domain.OpContainerStart, store.ops[0].Kind)
	assert.Equal(t, "ok", store.ops[0].Result)
	require.Len(t, aud.records, 1)
	assert.Equal(t, domain.ActionContainerStart, aud.records[0].Action)
}

func TestRemoveRequiresAcknowledgement(t *testing.T) {
	svc, m, store, _ := newService(false)

	err := svc.Remove(context.Background(), "h1", "c1", engine.RemoveOptions{Force: true}, false)
	assert.ErrorIs(t, err, operations.ErrConfirmationRequired)
	assert.False(t, m.called, "no engine call without acknowledgement")
	assert.Empty(t, store.ops, "an unconfirmed destructive op is not recorded")
}

func TestRemoveWithAckProceeds(t *testing.T) {
	svc, m, store, _ := newService(false)

	require.NoError(t, svc.Remove(context.Background(), "h1", "c1", engine.RemoveOptions{Force: true}, true))
	assert.True(t, m.eng.removed)
	require.Len(t, store.ops, 1)
	assert.Equal(t, domain.OpContainerRemove, store.ops[0].Kind)
}

func TestPruneRequiresAck(t *testing.T) {
	svc, m, _, _ := newService(false)
	_, err := svc.PruneImages(context.Background(), "h1", false, false)
	assert.ErrorIs(t, err, operations.ErrConfirmationRequired)
	assert.False(t, m.called)
}

func TestPruneWritesImpactAndAckToAudit(t *testing.T) {
	svc, _, store, aud := newService(false)

	reclaimed, err := svc.PruneImages(context.Background(), "h1", false, true)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), reclaimed)

	// Operation recorded with the bytes reclaimed.
	require.Len(t, store.ops, 1)
	assert.Equal(t, domain.OpImagePrune, store.ops[0].Kind)
	assert.Equal(t, int64(1000), store.ops[0].BytesReclaimed)
	// Confirmed impact persisted for the operation.
	require.Contains(t, store.impacts, store.ops[0].ID)
	// Audit entry carries the acknowledgement and the impact.
	require.Len(t, aud.records, 1)
	assert.Equal(t, domain.ActionImagePrune, aud.records[0].Action)
	assert.Equal(t, true, aud.records[0].Detail["ack"])
	assert.Contains(t, aud.records[0].Detail, "bytes_reclaimed")
}

func TestRemoveVolumeRequiresAckAndIsPerVolume(t *testing.T) {
	svc, m, _, aud := newService(false)

	// Without ack, no engine call (volumes are never bulk-deleted; each confirmed).
	assert.ErrorIs(t, svc.RemoveVolume(context.Background(), "h1", "db-data", false), operations.ErrConfirmationRequired)
	assert.False(t, m.called)

	require.NoError(t, svc.RemoveVolume(context.Background(), "h1", "db-data", true))
	assert.True(t, m.eng.removed)
	require.Len(t, aud.records, 1)
	assert.Equal(t, domain.ActionVolumeRemove, aud.records[0].Action)
	assert.Equal(t, "db-data", aud.records[0].Subject)
}

func TestComposeUpStartsStackWithoutAck(t *testing.T) {
	svc, m, store, aud := newService(false)

	require.NoError(t, svc.ComposeUp(context.Background(), "h1", "blog"))
	assert.True(t, m.eng.composeUp)
	require.Len(t, store.ops, 1)
	assert.Equal(t, domain.OpComposeUp, store.ops[0].Kind)
	assert.Equal(t, "blog", store.ops[0].Target)
	require.Len(t, aud.records, 1)
	assert.Equal(t, domain.ActionComposeUp, aud.records[0].Action)
}

func TestComposeDownRequiresAcknowledgement(t *testing.T) {
	svc, m, store, _ := newService(false)

	err := svc.ComposeDown(context.Background(), "h1", "blog", true, false)
	assert.ErrorIs(t, err, operations.ErrConfirmationRequired)
	assert.False(t, m.called, "no engine call without acknowledgement")
	assert.Empty(t, store.ops, "an unconfirmed down -v is not recorded")
}

func TestComposeDownVolumesWithAckProceeds(t *testing.T) {
	svc, m, store, aud := newService(false)

	require.NoError(t, svc.ComposeDown(context.Background(), "h1", "blog", true, true))
	require.NotNil(t, m.eng.composeDownVolumes)
	assert.True(t, *m.eng.composeDownVolumes, "down -v passes volumes through to the engine")
	require.Len(t, store.ops, 1)
	assert.Equal(t, domain.OpComposeDown, store.ops[0].Kind)
	assert.Equal(t, true, store.ops[0].OptionSet["volumes"])
	require.Len(t, aud.records, 1)
	assert.Equal(t, domain.ActionComposeDown, aud.records[0].Action)
	assert.Equal(t, true, aud.records[0].Detail["volumes"])
}

func TestExecRedactsSecretEnvOnCapture(t *testing.T) {
	svc, _, store, aud := newService(false)

	_, err := svc.Exec(context.Background(), "h1", "c1", engine.ExecSpec{
		Cmd: []string{"printenv"}, User: "root", Env: []string{"DB_PASSWORD=hunter2"},
	})
	require.NoError(t, err)

	// The persisted operation redacts the secret env; the non-secret user stays.
	require.Len(t, store.ops, 1)
	op := store.ops[0]
	assert.Equal(t, options.RedactedValue, op.OptionSet["env"])
	assert.Equal(t, "root", op.OptionSet["user"])

	// The audit detail redacts it too.
	require.Len(t, aud.records, 1)
	assert.Equal(t, options.RedactedValue, aud.records[0].Detail["env"])

	// The secret value appears in no persisted row or audit detail.
	assert.NotContains(t, fmt.Sprint(op.OptionSet), "hunter2")
	assert.NotContains(t, fmt.Sprint(aud.records[0].Detail), "hunter2")
}

func TestObserveModeRejectsAndIsRecorded(t *testing.T) {
	svc, m, store, _ := newService(true)

	err := svc.Stop(context.Background(), "h1", "c1")
	assert.ErrorIs(t, err, domain.ErrObserveMode)
	assert.False(t, m.eng.stopped, "the engine is never touched")
	require.Len(t, store.ops, 1, "the blocked attempt is still recorded for accountability")
	assert.Contains(t, store.ops[0].Result, "observe")
}
