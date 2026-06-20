// Package operations performs mutating container operations through the single
// guarded path: every mutation passes the registry's observe-mode check before
// the engine, is captured as a recorded Operation, and writes an audit entry
// (PROJECT-BOOK §2.9, ADR-0010/0013). Destructive operations require an explicit
// acknowledgement (ADR-0011).
package operations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// ErrConfirmationRequired is returned when a destructive operation is attempted
// without acknowledgement. Maps to ERR_DESTRUCTIVE_NOT_CONFIRMED (§8.4).
var ErrConfirmationRequired = errors.New("destructive operation requires acknowledgement")

// Mutator runs a function against a host's engine after enforcing observe-mode
// (satisfied by *hosts.Registry).
type Mutator interface {
	Mutate(ctx context.Context, hostID string, fn func(context.Context, engine.Engine) error) error
}

// Store persists operation records.
type Store interface {
	SaveOperation(ctx context.Context, op domain.Operation) error
}

// Auditor records consequential actions.
type Auditor interface {
	Append(ctx context.Context, r audit.Record) (domain.AuditEntry, error)
}

// Service executes and records mutating operations.
type Service struct {
	mutator Mutator
	store   Store
	auditor Auditor
	now     func() time.Time
}

// New constructs the service. now defaults to time.Now.
func New(mutator Mutator, store Store, auditor Auditor, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{mutator: mutator, store: store, auditor: auditor, now: now}
}

// Start starts a container.
func (s *Service) Start(ctx context.Context, hostID, containerID string) error {
	return s.run(ctx, hostID, domain.OpContainerStart, containerID, nil, true,
		func(ctx context.Context, e engine.Engine) error { return e.StartContainer(ctx, containerID) })
}

// Stop gracefully stops a container.
func (s *Service) Stop(ctx context.Context, hostID, containerID string) error {
	return s.run(ctx, hostID, domain.OpContainerStop, containerID, nil, true,
		func(ctx context.Context, e engine.Engine) error { return e.StopContainer(ctx, containerID) })
}

// Restart restarts a container.
func (s *Service) Restart(ctx context.Context, hostID, containerID string) error {
	return s.run(ctx, hostID, domain.OpContainerRestart, containerID, nil, true,
		func(ctx context.Context, e engine.Engine) error { return e.RestartContainer(ctx, containerID) })
}

// Kill force-kills a container; destructive, so ack must be true.
func (s *Service) Kill(ctx context.Context, hostID, containerID string, ack bool) error {
	return s.run(ctx, hostID, domain.OpContainerKill, containerID, nil, ack,
		func(ctx context.Context, e engine.Engine) error { return e.KillContainer(ctx, containerID) })
}

// Remove removes a container; destructive, so ack must be true.
func (s *Service) Remove(ctx context.Context, hostID, containerID string, opts engine.RemoveOptions, ack bool) error {
	optionSet := map[string]any{"force": opts.Force, "volumes": opts.Volumes}
	return s.run(ctx, hostID, domain.OpContainerRemove, containerID, optionSet, ack,
		func(ctx context.Context, e engine.Engine) error { return e.RemoveContainer(ctx, containerID, opts) })
}

// Exec starts an argv command in a container after the observe-mode guard, audits
// it, and returns the live stream. Exec is treated as a mutation (ADR-0013).
func (s *Service) Exec(ctx context.Context, hostID, containerID string, spec engine.ExecSpec) (engine.ExecStream, error) {
	var stream engine.ExecStream
	err := s.mutator.Mutate(ctx, hostID, func(ctx context.Context, e engine.Engine) error {
		st, execErr := e.Exec(ctx, containerID, spec)
		if execErr != nil {
			return execErr
		}
		stream = st
		return nil
	})
	s.record(ctx, hostID, domain.OpContainerExec, containerID,
		map[string]any{"cmd": spec.Cmd, "user": spec.User}, s.now(), s.now(), err)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (s *Service) run(
	ctx context.Context,
	hostID string,
	kind domain.OperationKind,
	target string,
	optionSet map[string]any,
	ack bool,
	fn func(context.Context, engine.Engine) error,
) error {
	if kind.Destructive() && !ack {
		return fmt.Errorf("%w: %s", ErrConfirmationRequired, kind)
	}

	started := s.now()
	err := s.mutator.Mutate(ctx, hostID, fn)
	s.record(ctx, hostID, kind, target, optionSet, started, s.now(), err)
	return err
}

// record persists the operation outcome and writes an audit entry. Failures to
// record never mask the operation's own result.
func (s *Service) record(
	ctx context.Context,
	hostID string,
	kind domain.OperationKind,
	target string,
	optionSet map[string]any,
	started, ended time.Time,
	opErr error,
) {
	result := "ok"
	if opErr != nil {
		result = "error: " + opErr.Error()
	}

	op := domain.Operation{
		ID:        newID(),
		HostRef:   hostID,
		Kind:      kind,
		Target:    target,
		OptionSet: optionSet,
		Result:    result,
		StartedAt: started,
		EndedAt:   ended,
	}
	if s.store != nil {
		_ = s.store.SaveOperation(ctx, op)
	}
	if s.auditor != nil {
		detail := map[string]any{"result": result}
		for k, v := range optionSet {
			detail[k] = v
		}
		_, _ = s.auditor.Append(ctx, audit.Record{
			Action:  auditAction(kind),
			HostRef: hostID,
			Subject: target,
			Detail:  detail,
		})
	}
}

func auditAction(kind domain.OperationKind) domain.Action {
	switch kind {
	case domain.OpContainerStart:
		return domain.ActionContainerStart
	case domain.OpContainerStop:
		return domain.ActionContainerStop
	case domain.OpContainerRestart:
		return domain.ActionContainerRestart
	case domain.OpContainerKill:
		return domain.ActionContainerKill
	case domain.OpContainerRemove:
		return domain.ActionContainerRemove
	case domain.OpContainerExec:
		return domain.ActionContainerExec
	default:
		return domain.Action(kind)
	}
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
