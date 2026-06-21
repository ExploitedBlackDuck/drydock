// Package audit implements Drydock's append-only, hash-chained audit log
// (PROJECT-BOOK §7.8, ADR-0010). Each entry's hash chains to the previous one,
// so any later mutation of a recorded entry breaks verification — the log is
// tamper-evident.
package audit

import (
	"bytes"
	"crypto/hmac"
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

// ComputeMAC returns the hex-encoded authentication code over the previous
// entry's MAC concatenated with this entry's canonical encoding (ADR-0025):
//
//	mac = HMAC-SHA256(key, prev_mac || canonical(entry))
//
// When key is empty the function degrades to a plain SHA-256 — a structural
// chain that is still tamper-evident against in-place edits but not keyed; this
// is the key-unavailable mode, flagged as such on verify rather than claimed
// intact.
func ComputeMAC(key []byte, prevMAC string, e domain.AuditEntry) (string, error) {
	body, err := canonical(e)
	if err != nil {
		return "", err
	}
	var h interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	}
	if len(key) > 0 {
		h = hmac.New(sha256.New, key)
	} else {
		h = sha256.New()
	}
	_, _ = h.Write([]byte(prevMAC))
	_, _ = h.Write(body)
	return hex.EncodeToString(h.Sum(nil)), nil
}
