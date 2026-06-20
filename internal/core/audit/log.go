package audit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// ErrChainBroken is returned by Verify when the audit chain fails to validate —
// a missing entry, a non-contiguous sequence, a broken back-link, or a hash that
// does not match the recorded content (i.e. tampering). Maps to the
// ERR_AUDIT_CHAIN_BROKEN error code (PROJECT-BOOK §8.4).
var ErrChainBroken = errors.New("audit chain broken")

// Store is the narrow persistence port the audit log requires. It is declared
// by this consumer (PROJECT-BOOK §2.1) and implemented by the sqlitestore
// adapter. All access is append-only; there is no update or delete.
type Store interface {
	// LastAuditEntry returns the most recent entry and true, or false when the
	// log is empty.
	LastAuditEntry(ctx context.Context) (domain.AuditEntry, bool, error)
	// AppendAuditEntry persists a fully-formed entry (Seq and hashes already set).
	AppendAuditEntry(ctx context.Context, entry domain.AuditEntry) error
	// AuditEntries returns every entry ordered by ascending Seq.
	AuditEntries(ctx context.Context) ([]domain.AuditEntry, error)
}

// Clock supplies the current time, injected so tests are deterministic.
type Clock func() time.Time

// Log appends to and verifies the audit chain. Appends are serialized so the
// sequence and back-links are assigned without races within the process.
type Log struct {
	store Store
	now   Clock
	mu    sync.Mutex
}

// New constructs a Log over the given store. If now is nil, time.Now is used.
func New(store Store, now Clock) *Log {
	if now == nil {
		now = time.Now
	}
	return &Log{store: store, now: now}
}

// Record is the caller-supplied content of an audit entry; the log assigns Seq,
// At, PrevHash, and Hash.
type Record struct {
	Action  domain.Action
	HostRef string
	Subject string
	Detail  map[string]any
}

// Append links a new entry to the end of the chain and persists it, returning
// the stored entry with its assigned sequence and hash.
func (l *Log) Append(ctx context.Context, r Record) (domain.AuditEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	last, ok, err := l.store.LastAuditEntry(ctx)
	if err != nil {
		return domain.AuditEntry{}, fmt.Errorf("reading last audit entry: %w", err)
	}

	entry := domain.AuditEntry{
		Seq:     1,
		At:      l.now().UTC(),
		Action:  r.Action,
		HostRef: r.HostRef,
		Subject: r.Subject,
		Detail:  r.Detail,
	}
	if ok {
		entry.Seq = last.Seq + 1
		entry.PrevHash = last.Hash
	}

	hash, err := ComputeHash(entry.PrevHash, entry)
	if err != nil {
		return domain.AuditEntry{}, fmt.Errorf("hashing audit entry: %w", err)
	}
	entry.Hash = hash

	if err := l.store.AppendAuditEntry(ctx, entry); err != nil {
		return domain.AuditEntry{}, fmt.Errorf("appending audit entry: %w", err)
	}
	return entry, nil
}

// Verify recomputes the entire chain and confirms its integrity. It returns the
// number of entries verified, or an error wrapping ErrChainBroken at the first
// inconsistency.
func (l *Log) Verify(ctx context.Context) (int, error) {
	entries, err := l.store.AuditEntries(ctx)
	if err != nil {
		return 0, fmt.Errorf("loading audit entries: %w", err)
	}

	prevHash := ""
	for i, entry := range entries {
		expectedSeq := int64(i + 1)
		if entry.Seq != expectedSeq {
			return i, fmt.Errorf("%w: entry %d has sequence %d", ErrChainBroken, expectedSeq, entry.Seq)
		}
		if entry.PrevHash != prevHash {
			return i, fmt.Errorf("%w: entry %d back-link mismatch", ErrChainBroken, entry.Seq)
		}
		want, err := ComputeHash(prevHash, entry)
		if err != nil {
			return i, fmt.Errorf("hashing audit entry %d: %w", entry.Seq, err)
		}
		if want != entry.Hash {
			return i, fmt.Errorf("%w: entry %d content does not match its hash", ErrChainBroken, entry.Seq)
		}
		prevHash = entry.Hash
	}
	return len(entries), nil
}
