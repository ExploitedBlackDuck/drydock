package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/drydock/drydock/internal/core/domain"
)

// ErrHostNotFound is returned when no host profile exists for an id.
var ErrHostNotFound = errors.New("host not found")

// SaveHost inserts or updates a host profile. Trust is not persisted — it is
// derived from the transport and endpoint on load (domain.TrustFor).
func (s *Store) SaveHost(ctx context.Context, h domain.Host) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO hosts (id, name, transport, endpoint, observe_mode,
		     last_engine_version, last_api_version, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name = excluded.name,
		     transport = excluded.transport,
		     endpoint = excluded.endpoint,
		     observe_mode = excluded.observe_mode,
		     last_engine_version = excluded.last_engine_version,
		     last_api_version = excluded.last_api_version`,
		h.ID, h.Name, string(h.Transport), h.Endpoint, boolToInt(h.ObserveMode),
		nullable(h.EngineVersion), nullable(h.APIVersion), s.now().UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("saving host %q: %w", h.ID, err)
	}
	return nil
}

// Hosts returns all saved host profiles ordered by name.
func (s *Store) Hosts(ctx context.Context) ([]domain.Host, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, transport, endpoint, observe_mode,
		        last_engine_version, last_api_version
		 FROM hosts ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("querying hosts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var hosts []domain.Host
	for rows.Next() {
		h, scanErr := scanHost(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning host: %w", scanErr)
		}
		hosts = append(hosts, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hosts: %w", err)
	}
	return hosts, nil
}

// Host returns the profile for id, or ErrHostNotFound.
func (s *Store) Host(ctx context.Context, id string) (domain.Host, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, transport, endpoint, observe_mode,
		        last_engine_version, last_api_version
		 FROM hosts WHERE id = ?`, id)
	h, err := scanHost(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Host{}, ErrHostNotFound
	}
	if err != nil {
		return domain.Host{}, fmt.Errorf("reading host %q: %w", id, err)
	}
	return h, nil
}

// DeleteHost removes a host profile and its recorded data; removing an absent
// host is not an error. The host's operations, resource samples, and timeline
// are deleted (children before parents, in one transaction) so removal succeeds
// even with history, which the operations/resource_samples foreign keys would
// otherwise restrict. The hash-chained audit log has no foreign key and is
// deliberately left intact — a host's audited actions stay durable after the
// profile is removed (ADR-0010).
func (s *Store) DeleteHost(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("removing host %q: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		`DELETE FROM prune_impacts WHERE operation_id IN (SELECT id FROM operations WHERE host_id = ?)`,
		`DELETE FROM operations WHERE host_id = ?`,
		`DELETE FROM resource_samples WHERE host_id = ?`,
		`DELETE FROM timeline_entries WHERE host_id = ?`,
		`DELETE FROM hosts WHERE id = ?`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt, id); err != nil {
			return fmt.Errorf("removing host %q data: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing host %q removal: %w", id, err)
	}
	return nil
}

// SetHostObserveMode updates only the observe-mode flag for a host.
func (s *Store) SetHostObserveMode(ctx context.Context, id string, observe bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE hosts SET observe_mode = ? WHERE id = ?`, boolToInt(observe), id)
	if err != nil {
		return fmt.Errorf("updating observe mode for host %q: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirming observe-mode update for host %q: %w", id, err)
	}
	if affected == 0 {
		return ErrHostNotFound
	}
	return nil
}

func scanHost(row scanner) (domain.Host, error) {
	var (
		h         domain.Host
		transport string
		observe   int
		engineVer sql.NullString
		apiVer    sql.NullString
	)
	if err := row.Scan(&h.ID, &h.Name, &transport, &h.Endpoint, &observe, &engineVer, &apiVer); err != nil {
		return domain.Host{}, err
	}
	h.Transport = domain.Transport(transport)
	h.ObserveMode = observe != 0
	h.EngineVersion = engineVer.String
	h.APIVersion = apiVer.String
	h.Trust = domain.TrustFor(h.Transport, h.Endpoint)
	return h, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
