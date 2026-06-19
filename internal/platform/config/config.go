package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is the typed application configuration loaded from config.toml. Every
// field has a safe zero-value default so a missing file yields a usable config.
type Config struct {
	// Log configures operational logging (distinct from the audit log).
	Log LogConfig `toml:"log"`
	// History bounds the rolling resource-sample retention window.
	History HistoryConfig `toml:"history"`
}

// LogConfig controls the operational logger.
type LogConfig struct {
	// Level is one of "debug", "info", "warn", "error".
	Level string `toml:"level"`
	// Format is "json" (file) or "text" (pretty, for development).
	Format string `toml:"format"`
}

// HistoryConfig bounds the rolling resource-sample retention.
type HistoryConfig struct {
	// ResourceRetention is how long container resource samples are kept.
	ResourceRetention Duration `toml:"resource_retention"`
}

// Default returns the configuration used when no file is present.
func Default() Config {
	return Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		History: HistoryConfig{
			ResourceRetention: Duration{24 * time.Hour},
		},
	}
}

// Load reads config.toml from path, returning defaults when the file is absent.
// A present-but-invalid file is an error; a missing file is not.
func Load(path string) (Config, error) {
	cfg := Default()

	// path is Drydock's own XDG-resolved config location (config.ResolvePaths),
	// never operator-supplied input, so directory traversal does not apply.
	data, err := os.ReadFile(path) //nolint:gosec // G304: trusted internal path
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("reading config %q: %w", path, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}

var validLevels = map[string]struct{}{
	"debug": {}, "info": {}, "warn": {}, "error": {},
}

var validFormats = map[string]struct{}{
	"json": {}, "text": {},
}

func (c Config) validate() error {
	if _, ok := validLevels[c.Log.Level]; !ok {
		return fmt.Errorf("log.level %q is not one of debug/info/warn/error", c.Log.Level)
	}
	if _, ok := validFormats[c.Log.Format]; !ok {
		return fmt.Errorf("log.format %q is not one of json/text", c.Log.Format)
	}
	if c.History.ResourceRetention.Duration <= 0 {
		return errors.New("history.resource_retention must be positive")
	}
	return nil
}

// Duration is a time.Duration that (un)marshals from a TOML string such as
// "24h" or "30m", which plain time.Duration does not support in TOML.
type Duration struct {
	time.Duration
}

// UnmarshalText parses a Go duration string (e.g. "24h").
func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("parsing duration %q: %w", text, err)
	}
	d.Duration = parsed
	return nil
}

// MarshalText renders the duration as a Go duration string.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}
