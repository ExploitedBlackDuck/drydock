package app

import (
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/journal"
)

// History & audit bindings (PROJECT-BOOK §7.11.8). These are read-only views
// over the recorded operation history and the append-only audit chain; the
// journal service owns the queries and the integrity check.

// OperationHistory returns recorded operations for a host, optionally narrowed
// to a single kind or to destructive operations only, most recent first.
func (a *App) OperationHistory(hostID, kind string, destructiveOnly bool, limit int) ([]domain.Operation, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.journal.Operations(ctx, journal.Filter{
		HostRef:         hostID,
		Kind:            kind,
		DestructiveOnly: destructiveOnly,
		Limit:           limit,
	})
}

// AuditTrail returns the audit log together with its chain-verification result,
// which drives the green (intact) / red (tampered) indicator in the UI.
func (a *App) AuditTrail() (journal.AuditStatus, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.journal.AuditTrail(ctx)
}

// ExportJournal returns the full accountability record — every operation and the
// complete audit chain, with the chain's integrity stamped at export time — as
// indented JSON. The frontend offers it to the operator as a downloadable file.
func (a *App) ExportJournal() (string, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	export, err := a.journal.Export(ctx)
	if err != nil {
		return "", err
	}
	data, err := journal.MarshalExport(export)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
