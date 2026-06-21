// Package hosts manages the set of Docker hosts: their persisted profiles, their
// connection lifecycle, and the observe-mode guard. It is the single place a
// mutating operation passes through, so observe-only hosts are rejected in the
// core before any request reaches the engine (ADR-0013).
package hosts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// ErrUnknownHost is returned for an id with no profile.
var ErrUnknownHost = errors.New("unknown host")

// ErrNotConnected is returned when an operation needs a live engine but the host
// is not connected.
var ErrNotConnected = errors.New("host is not connected")

// Store is the persistence port the registry needs (consumer-defined, §2.1).
type Store interface {
	SaveHost(ctx context.Context, h domain.Host) error
	Hosts(ctx context.Context) ([]domain.Host, error)
	Host(ctx context.Context, id string) (domain.Host, error)
	DeleteHost(ctx context.Context, id string) error
	SetHostObserveMode(ctx context.Context, id string, observe bool) error
}

// Connector opens an engine connection for a host profile. Implemented by an
// adapter that knows local, SSH, and TLS transports.
type Connector interface {
	Connect(ctx context.Context, h domain.Host) (engine.Engine, error)
}

// Auditor records consequential host actions to the audit log.
type Auditor interface {
	Append(ctx context.Context, r audit.Record) (domain.AuditEntry, error)
}

// Status pairs a host profile with its live connection state for the UI.
type Status struct {
	Host      domain.Host
	Connected bool
}

// Registry is the authoritative set of hosts and their connections.
type Registry struct {
	store     Store
	connector Connector
	auditor   Auditor
	now       func() time.Time

	mu    sync.Mutex
	index map[string]domain.Host
	conns map[string]engine.Engine
}

// New constructs a registry. now defaults to time.Now.
func New(store Store, connector Connector, auditor Auditor, now func() time.Time) *Registry {
	if now == nil {
		now = time.Now
	}
	return &Registry{
		store:     store,
		connector: connector,
		auditor:   auditor,
		now:       now,
		index:     map[string]domain.Host{},
		conns:     map[string]engine.Engine{},
	}
}

// Load reads persisted host profiles into the in-memory index.
func (r *Registry) Load(ctx context.Context) error {
	saved, err := r.store.Hosts(ctx)
	if err != nil {
		return fmt.Errorf("loading host profiles: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, h := range saved {
		r.index[h.ID] = h
	}
	return nil
}

// List returns every host profile with its current connection status.
func (r *Registry) List() []Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Status, 0, len(r.index))
	for id, h := range r.index {
		_, connected := r.conns[id]
		out = append(out, Status{Host: h, Connected: connected})
	}
	return out
}

// Add validates and persists a new host profile, deriving its trust level. It
// does not connect; call Connect afterwards.
func (r *Registry) Add(ctx context.Context, h domain.Host) (domain.Host, error) {
	if h.ID == "" {
		h.ID = newID()
	}
	h.Trust = domain.TrustFor(h.Transport, h.Endpoint)
	if err := h.Validate(); err != nil {
		return domain.Host{}, fmt.Errorf("invalid host: %w", err)
	}
	if err := r.store.SaveHost(ctx, h); err != nil {
		return domain.Host{}, err
	}
	r.mu.Lock()
	r.index[h.ID] = h
	r.mu.Unlock()
	return h, nil
}

// Remove disconnects (if connected) and deletes a host profile.
func (r *Registry) Remove(ctx context.Context, id string) error {
	_ = r.Disconnect(ctx, id)
	if err := r.store.DeleteHost(ctx, id); err != nil {
		return err
	}
	r.mu.Lock()
	delete(r.index, id)
	r.mu.Unlock()
	return nil
}

// Connect opens an engine for the host, records its engine/API version, and
// audits the connection. It is idempotent: an already-connected host is a no-op.
func (r *Registry) Connect(ctx context.Context, id string) (domain.Host, error) {
	r.mu.Lock()
	host, ok := r.index[id]
	_, already := r.conns[id]
	r.mu.Unlock()
	if !ok {
		return domain.Host{}, ErrUnknownHost
	}
	if already {
		return host, nil
	}

	eng, err := r.connector.Connect(ctx, host)
	if err != nil {
		return domain.Host{}, fmt.Errorf("connecting to host %q: %w", host.Name, err)
	}

	if info, infoErr := eng.Info(ctx); infoErr == nil {
		host.EngineVersion = info.EngineVersion
		host.APIVersion = info.APIVersion
		_ = r.store.SaveHost(ctx, host)
	}

	r.mu.Lock()
	r.conns[id] = eng
	r.index[id] = host
	r.mu.Unlock()

	r.audit(ctx, audit.Record{
		Action:  domain.ActionHostConnect,
		HostRef: host.ID,
		Subject: host.Name,
		Detail:  map[string]any{"transport": string(host.Transport), "trust": string(host.Trust)},
	})
	return host, nil
}

// Reconnect re-establishes a host's engine after a dropped connection: it closes
// any existing (dead) engine and opens a fresh one (ADR-0021). The event
// supervisor calls it when a live stream drops, so the UI can resync against a
// live connection rather than resume a stale one. A failed reconnect leaves the
// host disconnected (no engine), which the next attempt retries.
func (r *Registry) Reconnect(ctx context.Context, id string) (domain.Host, error) {
	if err := r.Disconnect(ctx, id); err != nil {
		return domain.Host{}, fmt.Errorf("closing prior connection to host %q: %w", id, err)
	}
	return r.Connect(ctx, id)
}

// Disconnect closes the engine connection (and its SSH tunnel) and audits it.
// Disconnecting an unconnected host is a no-op.
func (r *Registry) Disconnect(ctx context.Context, id string) error {
	r.mu.Lock()
	eng, ok := r.conns[id]
	host := r.index[id]
	if ok {
		delete(r.conns, id)
	}
	r.mu.Unlock()
	if !ok {
		return nil
	}

	err := eng.Close()
	r.audit(ctx, audit.Record{
		Action:  domain.ActionHostDisconnect,
		HostRef: id,
		Subject: host.Name,
	})
	if err != nil {
		return fmt.Errorf("disconnecting host %q: %w", host.Name, err)
	}
	return nil
}

// SetObserveMode toggles observe-only for a host, persisting and auditing the
// change (ADR-0013: leaving observe mode is explicit and audited).
func (r *Registry) SetObserveMode(ctx context.Context, id string, observe bool) error {
	r.mu.Lock()
	host, ok := r.index[id]
	r.mu.Unlock()
	if !ok {
		return ErrUnknownHost
	}
	if err := r.store.SetHostObserveMode(ctx, id, observe); err != nil {
		return err
	}
	host.ObserveMode = observe
	r.mu.Lock()
	r.index[id] = host
	r.mu.Unlock()

	action := domain.ActionObserveDisabled
	if observe {
		action = domain.ActionObserveEnabled
	}
	r.audit(ctx, audit.Record{Action: action, HostRef: id, Subject: host.Name})
	return nil
}

// Engine returns the live engine for a connected host.
func (r *Registry) Engine(id string) (engine.Engine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	eng, ok := r.conns[id]
	if !ok {
		if _, known := r.index[id]; !known {
			return nil, ErrUnknownHost
		}
		return nil, ErrNotConnected
	}
	return eng, nil
}

// Mutate runs fn against the host's engine, but only after confirming the host
// is not in observe mode (ADR-0013) — the check happens in the core, before fn
// ever touches the engine. This is the single path all mutating operations use.
func (r *Registry) Mutate(ctx context.Context, id string, fn func(context.Context, engine.Engine) error) error {
	r.mu.Lock()
	host, known := r.index[id]
	eng, connected := r.conns[id]
	r.mu.Unlock()

	if !known {
		return ErrUnknownHost
	}
	if err := host.EnsureMutable(); err != nil {
		return err
	}
	if !connected {
		return ErrNotConnected
	}
	return fn(ctx, eng)
}

// Close disconnects every connected host, closing all tunnels. Used on shutdown
// so no SSH tunnel or engine stream is left open (PROJECT-BOOK §2.3).
func (r *Registry) Close(ctx context.Context) error {
	r.mu.Lock()
	ids := make([]string, 0, len(r.conns))
	for id := range r.conns {
		ids = append(ids, id)
	}
	r.mu.Unlock()

	var errs []error
	for _, id := range ids {
		if err := r.Disconnect(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *Registry) audit(ctx context.Context, rec audit.Record) {
	if r.auditor == nil {
		return
	}
	if _, err := r.auditor.Append(ctx, rec); err != nil {
		// Auditing must not break the operation; the operational logger captures
		// the failure elsewhere. We deliberately swallow it here.
		_ = err
	}
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
