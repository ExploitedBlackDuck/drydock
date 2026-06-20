// Package audit implements Drydock's append-only, hash-chained audit log
// (PROJECT-BOOK §7.8, ADR-0010). Each entry's hash chains to the previous one,
// so any later mutation of a recorded entry breaks verification — the log is
// tamper-evident.
package audit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/drydock/drydock/internal/core/domain"
)

// canonical produces a deterministic byte encoding of an entry's content. Fields
// are length-prefixed so no combination of values can collide, and Detail is
// marshalled with encoding/json (which sorts map keys) for a stable form.
func canonical(e domain.AuditEntry) ([]byte, error) {
	detail, err := json.Marshal(e.Detail)
	if err != nil {
		return nil, fmt.Errorf("encoding audit detail: %w", err)
	}

	var buf bytes.Buffer
	for _, field := range [][]byte{
		[]byte(strconv.FormatInt(e.Seq, 10)),
		[]byte(strconv.FormatInt(e.At.UTC().UnixNano(), 10)),
		[]byte(e.Action),
		[]byte(e.HostRef),
		[]byte(e.Subject),
		detail,
	} {
		// Decimal length prefix + delimiter makes the encoding injective: no
		// combination of field values can produce the same byte stream.
		buf.WriteString(strconv.Itoa(len(field)))
		buf.WriteByte(':')
		buf.Write(field)
	}
	return buf.Bytes(), nil
}

// ComputeHash returns the hex-encoded SHA-256 of the previous entry's hash
// concatenated with this entry's canonical encoding:
//
//	hash = SHA256(prev_hash || canonical(entry))
func ComputeHash(prevHash string, e domain.AuditEntry) (string, error) {
	body, err := canonical(e)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil)), nil
}
