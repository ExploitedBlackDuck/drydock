// Package journal serves Drydock's accountability views (PROJECT-BOOK §7.11.8):
// the operation-history browser with its queries, the audit-log view with a
// chain-verification indicator, and a portable export of both. It owns the read
// model so the binding layer never composes a query or re-implements the
// destructive-only filter; integrity checks reuse the pure audit verifier.
package journal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
)

// ExportSchemaVersion identifies the export envelope's shape so a future reader
// can detect and migrate older files.
const ExportSchemaVersion = 1

// Store is the narrow read port the journal needs, declared by this consumer
// (PROJECT-BOOK §2.1) and satisfied by the sqlitestore adapter.
type Store interface {
	// Operations returns recorded operations matching q, most recent first.
	Operations(ctx context.Context, q domain.OperationQuery) ([]domain.Operation, error)
	// AuditEntries returns every audit entry ordered by ascending sequence.
	AuditEntries(ctx context.Context) ([]domain.AuditEntry, error)
}

// Service answers history and audit queries and produces exports.
type Service struct {
	store Store
}

// New constructs the journal service over a read store.
func New(store Store) *Service {
	return &Service{store: store}
}

// Filter is the history view's query (PROJECT-BOOK §7.11.8): by host, by kind,
// or restricted to destructive operations only. It is the binding-friendly shape
// the UI builds; the service translates it into a domain.OperationQuery so the
// store stays unaware of the destructive-kind set.
type Filter struct {
	HostRef         string
	Kind            string
	DestructiveOnly bool
	Limit           int
}

func (f Filter) query() domain.OperationQuery {
	q := domain.OperationQuery{HostRef: f.HostRef, Limit: f.Limit}
	switch {
	case f.DestructiveOnly:
		// Destructive-only is defined from the single source of truth in domain,
		// never duplicated as a hand-written kind list here or in SQL.
		q.Kinds = domain.DestructiveKinds()
	case f.Kind != "":
		q.Kinds = []domain.OperationKind{domain.OperationKind(f.Kind)}
	}
	return q
}

// Operations returns the operation history matching the filter.
func (s *Service) Operations(ctx context.Context, f Filter) ([]domain.Operation, error) {
	ops, err := s.store.Operations(ctx, f.query())
	if err != nil {
		return nil, fmt.Errorf("querying operation history: %w", err)
	}
	return ops, nil
}

// AuditStatus is the audit view's model: the entries plus the chain-verification
// result that drives the green/red indicator (PROJECT-BOOK §7.11.8).
type AuditStatus struct {
	Entries []domain.AuditEntry
	// Verified is true when the whole chain validates.
	Verified bool
	// VerifiedCount is the number of entries that validated before any break
	// (equal to len(Entries) when Verified).
	VerifiedCount int
	// Error explains the break when Verified is false (empty otherwise).
	Error string
}

// AuditTrail loads the audit log and verifies its chain in memory, so the view
// shows both the entries and whether they are intact (PROJECT-BOOK §7.8/§7.11.8).
func (s *Service) AuditTrail(ctx context.Context) (AuditStatus, error) {
	entries, err := s.store.AuditEntries(ctx)
	if err != nil {
		return AuditStatus{}, fmt.Errorf("loading audit entries: %w", err)
	}
	return verify(entries), nil
}

func verify(entries []domain.AuditEntry) AuditStatus {
	status := AuditStatus{Entries: entries}
	n, err := audit.VerifyEntries(entries)
	status.VerifiedCount = n
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Verified = true
	return status
}

// Export is the portable, self-verifying snapshot of the accountability record
// (PROJECT-BOOK §7.11.8): every operation and the full audit chain, with the
// chain's integrity recorded at export time. It marshals to and from JSON
// losslessly.
type Export struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Operations    []domain.Operation  `json:"operations"`
	Audit         []domain.AuditEntry `json:"audit"`
	AuditVerified bool                `json:"auditVerified"`
	AuditError    string              `json:"auditError,omitempty"`
}

// Export gathers all operations and the full audit chain into a portable
// envelope, recording whether the chain verified.
func (s *Service) Export(ctx context.Context) (Export, error) {
	// Limit < 0 means unbounded: an export is the whole record, not a page.
	ops, err := s.store.Operations(ctx, domain.OperationQuery{Limit: -1})
	if err != nil {
		return Export{}, fmt.Errorf("collecting operations for export: %w", err)
	}
	trail, err := s.AuditTrail(ctx)
	if err != nil {
		return Export{}, err
	}
	return Export{
		SchemaVersion: ExportSchemaVersion,
		Operations:    ops,
		Audit:         trail.Entries,
		AuditVerified: trail.Verified,
		AuditError:    trail.Error,
	}, nil
}

// MarshalExport renders an export as indented JSON suitable for writing to a file.
func MarshalExport(e Export) ([]byte, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding export: %w", err)
	}
	return data, nil
}

// ParseExport reads an export envelope back from JSON (the inverse of
// MarshalExport), used to confirm an export round-trips.
func ParseExport(data []byte) (Export, error) {
	var e Export
	if err := json.Unmarshal(data, &e); err != nil {
		return Export{}, fmt.Errorf("decoding export: %w", err)
	}
	return e, nil
}

// VerifyExportedChain confirms an export's audit chain is internally consistent,
// independent of any database (PROJECT-BOOK §7.11.8). It lets a recipient of an
// exported file check the chain on its own.
func VerifyExportedChain(e Export) (int, error) {
	return audit.VerifyEntries(e.Audit)
}
