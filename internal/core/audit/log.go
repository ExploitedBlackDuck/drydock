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
// a missing entry, a non-contiguous sequence, a broken back-link, a MAC that
// does not match the recorded content (in-place tampering), or a tail that is
// shorter than the external high-water mark (truncation). Maps to the
// ERR_AUDIT_CHAIN_BROKEN error code (PROJECT-BOOK §8.4).
var ErrChainBroken = errors.New("audit chain broken")

// Store is the narrow persistence port the audit log requires. It is declared
// by this consumer (PROJECT-BOOK §2.1) and implemented by the sqlitestore
// adapter. All access is append-only; there is no update or delete.
type Store interface {
	// LastAuditEntry returns the most recent entry and true, or false when the
	// log is empty.
	LastAuditEntry(ctx context.Context) (domain.AuditEntry, bool, error)
	// AppendAuditEntry persists a fully-formed entry (Seq and MACs already set).
	AppendAuditEntry(ctx context.Context, entry domain.AuditEntry) error
	// AuditEntries returns every entry ordered by ascending Seq.
	AuditEntries(ctx context.Context) ([]domain.AuditEntry, error)
}

// HighWaterMark is the latest (Seq, MAC) the log has appended, persisted outside
// the audit table so that truncating the table's tail is detectable on verify
// (ADR-0025).
type HighWaterMark struct {
	Seq int64
	MAC string
}

// MarkStore persists the external high-water mark. It is declared by this
// consumer and implemented by a small file/keyring adapter; a nil MarkStore
// disables truncation detection (the chain is still keyed and tamper-evident).
type MarkStore interface {
	Get(ctx context.Context) (HighWaterMark, bool, error)
	Set(ctx context.Context, m HighWaterMark) error
}

// Clock supplies the current time, injected so tests are deterministic.
type Clock func() time.Time

// Log appends to and verifies the audit chain. Appends are serialized so the
// sequence and back-links are assigned without races within the process.
type Log struct {
	store Store
	now   Clock
	// key is the per-install audit HMAC key (ADR-0025). Empty means the
	// key-unavailable degraded mode: entries still chain via plain SHA-256, but
	// verification cannot confirm authenticity and reports key-unavailable.
	key  []byte
	mark MarkStore
	mu   sync.Mutex
}

// New constructs a Log over the given store. key is the per-install audit HMAC
// key (nil/empty for the degraded key-unavailable mode); mark persists the
// truncation high-water mark (nil to disable truncation detection). If now is
// nil, time.Now is used.
func New(store Store, now Clock, key []byte, mark MarkStore) *Log {
	if now == nil {
		now = time.Now
	}
	return &Log{store: store, now: now, key: key, mark: mark}
}

// Record is the caller-supplied content of an audit entry; the log assigns Seq,
// At, PrevMAC, and MAC.
type Record struct {
	Action  domain.Action
	HostRef string
	Subject string
	Detail  map[string]any
}

// Append links a new entry to the end of the chain, persists it, and advances
// the external high-water mark, returning the stored entry.
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
		entry.PrevMAC = last.MAC
	}

	mac, err := ComputeMAC(l.key, entry.PrevMAC, entry)
	if err != nil {
		return domain.AuditEntry{}, fmt.Errorf("authenticating audit entry: %w", err)
	}
	entry.MAC = mac

	if err := l.store.AppendAuditEntry(ctx, entry); err != nil {
		return domain.AuditEntry{}, fmt.Errorf("appending audit entry: %w", err)
	}
	// Advance the high-water mark. Best-effort: a lagging mark only weakens
	// truncation detection, it never invents a break (verify treats the mark as a
	// floor, not an exact tail).
	if l.mark != nil {
		_ = l.mark.Set(ctx, HighWaterMark{Seq: entry.Seq, MAC: entry.MAC})
	}
	return entry, nil
}

// VerifyState is the outcome of verifying the chain (PROJECT-BOOK §7.11.8).
type VerifyState string

const (
	// VerifyIntact means the keyed chain validates end to end and its tail
	// matches the high-water mark.
	VerifyIntact VerifyState = "intact"
	// VerifyInPlaceTampered means an entry's content no longer matches its MAC,
	// or a back-link/sequence is broken.
	VerifyInPlaceTampered VerifyState = "in_place_tampered"
	// VerifyTruncated means the keyed chain validates but is shorter than the
	// external high-water mark — entries were removed from the tail.
	VerifyTruncated VerifyState = "truncated"
	// VerifyKeyUnavailable means the audit key is absent, so authenticity cannot
	// be confirmed; only structural consistency was checked.
	VerifyKeyUnavailable VerifyState = "key_unavailable"
)

// VerifyResult is the verification outcome and how many entries were confirmed.
type VerifyResult struct {
	State VerifyState
	// VerifiedCount is the number of entries confirmed before any break (equal to
	// the total when intact).
	VerifiedCount int
}

// Verify recomputes the chain and confirms its integrity against the key and the
// external high-water mark, classifying the result into one of the four states.
// It returns an error wrapping ErrChainBroken for the tampered and truncated
// states (so callers can both branch on State and use errors.Is).
func (l *Log) Verify(ctx context.Context) (VerifyResult, error) {
	entries, err := l.store.AuditEntries(ctx)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("loading audit entries: %w", err)
	}

	var mark *HighWaterMark
	if l.mark != nil {
		if m, ok, mErr := l.mark.Get(ctx); mErr == nil && ok {
			mark = &m
		}
	}
	return verify(l.key, entries, mark)
}

// VerifyEntries verifies a chain held in memory with the given key, with no
// truncation check (there is no external mark). It is pure, so an exported chain
// can be checked on its own (PROJECT-BOOK §7.11.8) — structurally always, and
// cryptographically when the verifier supplies the key.
func VerifyEntries(key []byte, entries []domain.AuditEntry) VerifyResult {
	result, _ := verify(key, entries, nil)
	return result
}

func verify(key []byte, entries []domain.AuditEntry, mark *HighWaterMark) (VerifyResult, error) {
	// Structural consistency (sequence + back-links) is checkable without the key.
	structural := structuralPrefix(entries)

	if len(key) == 0 {
		// Cannot confirm authenticity; report key-unavailable with the structural
		// prefix length rather than a false "intact."
		return VerifyResult{State: VerifyKeyUnavailable, VerifiedCount: structural}, nil
	}

	prevMAC := ""
	for i, entry := range entries {
		if entry.Seq != int64(i+1) || entry.PrevMAC != prevMAC {
			return VerifyResult{State: VerifyInPlaceTampered, VerifiedCount: i},
				fmt.Errorf("%w: entry %d sequence or back-link mismatch", ErrChainBroken, i+1)
		}
		want, err := ComputeMAC(key, prevMAC, entry)
		if err != nil {
			return VerifyResult{State: VerifyInPlaceTampered, VerifiedCount: i},
				fmt.Errorf("authenticating audit entry %d: %w", entry.Seq, err)
		}
		if want != entry.MAC {
			return VerifyResult{State: VerifyInPlaceTampered, VerifiedCount: i},
				fmt.Errorf("%w: entry %d content does not match its MAC", ErrChainBroken, entry.Seq)
		}
		prevMAC = entry.MAC
	}

	// The keyed chain is internally valid; now detect a truncated tail.
	if mark != nil {
		lastSeq := int64(0)
		if n := len(entries); n > 0 {
			lastSeq = entries[n-1].Seq
		}
		if mark.Seq > lastSeq {
			return VerifyResult{State: VerifyTruncated, VerifiedCount: len(entries)},
				fmt.Errorf("%w: tail truncated — high-water mark seq %d > last entry seq %d",
					ErrChainBroken, mark.Seq, lastSeq)
		}
	}
	return VerifyResult{State: VerifyIntact, VerifiedCount: len(entries)}, nil
}

// structuralPrefix returns how many entries from the start have a contiguous
// 1-based sequence and a matching back-link — the most that can be confirmed
// without the key.
func structuralPrefix(entries []domain.AuditEntry) int {
	prevMAC := ""
	for i, entry := range entries {
		if entry.Seq != int64(i+1) || entry.PrevMAC != prevMAC {
			return i
		}
		prevMAC = entry.MAC
	}
	return len(entries)
}
