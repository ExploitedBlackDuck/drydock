// Command drydock is the desktop control panel for local and remote Docker
// hosts. This file is the sole composition root (ADR-0014): it resolves
// configuration, constructs dependencies, wires them together, and runs the
// application. Business logic here is a defect.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/drydock/drydock/app"
	"github.com/drydock/drydock/frontend"
	"github.com/drydock/drydock/internal/platform/config"
	"github.com/drydock/drydock/internal/platform/logging"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		// Startup failures are unrecoverable; report and exit non-zero. This is
		// the one place a hard exit is permitted (PROJECT-BOOK §2.2).
		fmt.Fprintf(os.Stderr, "drydock: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	paths, err := config.ResolvePaths()
	if err != nil {
		return fmt.Errorf("resolving paths: %w", err)
	}
	if err := paths.EnsureDirs(); err != nil {
		return fmt.Errorf("preparing data directories: %w", err)
	}

	cfg, err := config.Load(paths.ConfigFile())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log, closeLog, err := buildLogger(cfg, paths)
	if err != nil {
		return fmt.Errorf("building logger: %w", err)
	}
	defer closeLog()

	log.Info("drydock initialized",
		slog.String("version", version),
		slog.String("config_dir", paths.ConfigDir),
		slog.String("data_dir", paths.DataDir),
	)

	assets, err := fs.Sub(frontend.Assets, "dist")
	if err != nil {
		return fmt.Errorf("loading embedded frontend: %w", err)
	}

	return app.Run(assets, log, version)
}

// buildLogger constructs the operational logger. In dev (text format) it writes
// to stderr; otherwise it writes JSON to a rotating file in the data dir. The
// returned closer flushes and releases the log sink.
func buildLogger(cfg config.Config, paths config.Paths) (*slog.Logger, func(), error) {
	opts := logging.Options{Level: cfg.Log.Level, Format: cfg.Log.Format}

	if cfg.Log.Format == "text" {
		return logging.New(os.Stderr, opts), func() {}, nil
	}

	sink := logging.RotatingFile(paths.LogFile())
	return logging.New(sink, opts), func() { _ = sink.Close() }, nil
}
