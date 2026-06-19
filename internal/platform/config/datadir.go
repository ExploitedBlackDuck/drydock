package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// userDataDir returns the platform base directory for application data,
// following the same conventions as os.UserConfigDir but for data rather than
// configuration:
//
//   - Linux: $XDG_DATA_HOME, or ~/.local/share when unset.
//   - macOS: ~/Library/Application Support.
//
// Windows is out of scope for v1 (PROJECT-BOOK §1.3) and returns an error.
func userDataDir() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" && filepath.IsAbs(dir) {
			return dir, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	default:
		return "", errors.New("unsupported platform: Drydock targets macOS and Linux")
	}
}
