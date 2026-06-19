// Package config resolves Drydock's on-disk locations and loads the typed
// application configuration. It follows platform conventions (XDG on Linux,
// Application Support on macOS) and never writes into the app bundle or the
// current working directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// appDir is the per-application subdirectory created under every base path.
const appDir = "drydock"

// Permissions for Drydock's private files. The data directory holds the SQLite
// database, the audit log, and sealed secrets, so it is owner-only.
const (
	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
)

// Paths holds the resolved, absolute locations Drydock reads and writes.
type Paths struct {
	// ConfigDir holds config.toml. Linux: $XDG_CONFIG_HOME/drydock; macOS:
	// ~/Library/Application Support/drydock.
	ConfigDir string
	// DataDir holds the SQLite database and audit log. Linux:
	// $XDG_DATA_HOME/drydock; macOS: ~/Library/Application Support/drydock.
	DataDir string
}

// ConfigFile is the absolute path to config.toml.
func (p Paths) ConfigFile() string { return filepath.Join(p.ConfigDir, "config.toml") }

// DatabaseFile is the absolute path to the SQLite database.
func (p Paths) DatabaseFile() string { return filepath.Join(p.DataDir, "drydock.db") }

// LogFile is the absolute path to the operational log (distinct from the audit
// log, which lives in the database).
func (p Paths) LogFile() string { return filepath.Join(p.DataDir, "drydock.log") }

// ResolvePaths computes Drydock's directories from the environment without
// creating them. Use EnsureDirs to create them with restrictive permissions.
func ResolvePaths() (Paths, error) {
	configBase, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolving user config dir: %w", err)
	}

	dataBase, err := userDataDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolving user data dir: %w", err)
	}

	return Paths{
		ConfigDir: filepath.Join(configBase, appDir),
		DataDir:   filepath.Join(dataBase, appDir),
	}, nil
}

// EnsureDirs creates the config and data directories with owner-only
// permissions, tightening any existing directory that is too permissive.
func (p Paths) EnsureDirs() error {
	for _, dir := range []string{p.ConfigDir, p.DataDir} {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("creating %q: %w", dir, err)
		}
		if err := os.Chmod(dir, dirPerm); err != nil {
			return fmt.Errorf("restricting permissions on %q: %w", dir, err)
		}
	}
	return nil
}
