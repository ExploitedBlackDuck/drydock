// Package auditmark persists the audit chain's high-water mark — the latest
// (seq, mac) — as a small JSON file kept deliberately outside the SQLite
// database (ADR-0025). Because the mark lives apart from the audit table,
// truncating that table's tail leaves the mark ahead of the last row, which the
// audit verifier detects. Writes are atomic (temp file + rename).
package auditmark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/drydock/drydock/internal/core/audit"
)

// FileMark is a file-backed audit.MarkStore.
type FileMark struct {
	path string
}

// NewFile returns a mark store backed by the file at path.
func NewFile(path string) *FileMark {
	return &FileMark{path: path}
}

type record struct {
	Seq int64  `json:"seq"`
	MAC string `json:"mac"`
}

// Get returns the stored mark, or false when none has been written yet.
func (f *FileMark) Get(ctx context.Context) (audit.HighWaterMark, bool, error) {
	if err := ctx.Err(); err != nil {
		return audit.HighWaterMark{}, false, err
	}
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return audit.HighWaterMark{}, false, nil
	}
	if err != nil {
		return audit.HighWaterMark{}, false, fmt.Errorf("reading audit mark: %w", err)
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return audit.HighWaterMark{}, false, fmt.Errorf("decoding audit mark: %w", err)
	}
	return audit.HighWaterMark{Seq: r.Seq, MAC: r.MAC}, true, nil
}

// Set replaces the stored mark atomically.
func (f *FileMark) Set(ctx context.Context, m audit.HighWaterMark) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(record{Seq: m.Seq, MAC: m.MAC})
	if err != nil {
		return fmt.Errorf("encoding audit mark: %w", err)
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing audit mark: %w", err)
	}
	if err := os.Rename(tmp, f.path); err != nil {
		return fmt.Errorf("replacing audit mark: %w", err)
	}
	return nil
}

// Compile-time assertion that FileMark satisfies the port.
var _ audit.MarkStore = (*FileMark)(nil)
