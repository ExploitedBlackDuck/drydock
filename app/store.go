package app

import "context"

// BackupStore is the persistence capability the backup binding needs, satisfied
// by the SQLite store. It is consumer-defined here (PROJECT-BOOK §2.1).
type BackupStore interface {
	// Backup writes a consistent snapshot to dest and returns the path.
	Backup(ctx context.Context, dest string) (string, error)
	// DefaultBackupPath returns a timestamped path beside the live database.
	DefaultBackupPath() string
}

// BackupDatabase writes a consistent snapshot of Drydock's database (operations,
// audit log, history) using the SQLite backup path — never a raw file copy
// (ADR-0024) — and returns the file it wrote. The operator is responsible for
// protecting the destination.
func (a *App) BackupDatabase() (string, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.backup.Backup(ctx, a.backup.DefaultBackupPath())
}
