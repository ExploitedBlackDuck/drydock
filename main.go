// Command drydock is the desktop control panel for local and remote Docker
// hosts. This file is the sole composition root (ADR-0014): it resolves
// configuration, constructs dependencies, wires them together, and runs the
// application. Business logic here is a defect.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/drydock/drydock/app"
	"github.com/drydock/drydock/frontend"
	"github.com/drydock/drydock/internal/adapters/auditmark"
	"github.com/drydock/drydock/internal/adapters/connector"
	"github.com/drydock/drydock/internal/adapters/keyring"
	"github.com/drydock/drydock/internal/adapters/sqlitestore"
	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
	"github.com/drydock/drydock/internal/core/history"
	"github.com/drydock/drydock/internal/core/hosts"
	"github.com/drydock/drydock/internal/core/journal"
	"github.com/drydock/drydock/internal/core/operations"
	"github.com/drydock/drydock/internal/core/secret"
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

	log.Info(
		"drydock initialized",
		slog.String("version", version),
		slog.String("config_dir", paths.ConfigDir),
		slog.String("data_dir", paths.DataDir),
	)

	ctx := context.Background()

	store, err := sqlitestore.Open(ctx, paths.DatabaseFile())
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer func() { _ = store.Close() }()

	if schema, schemaErr := store.SchemaVersion(ctx); schemaErr == nil {
		log.Info("store ready", slog.Int("schema_version", schema))
	}

	// Load the per-install audit HMAC key from the OS keyring (ADR-0025). If the
	// keyring is unavailable, the audit log runs in its degraded key-unavailable
	// mode rather than blocking startup — verification will flag it as such.
	keyStore := keyring.New()
	auditKey, keyErr := secret.LoadOrCreateAuditKey(ctx, keyStore)
	if keyErr != nil {
		log.Warn("audit key unavailable; audit chain runs unkeyed", slog.Any("error", keyErr))
		auditKey = nil
	}
	auditMark := auditmark.NewFile(filepath.Join(paths.DataDir, "audit.hwm"))

	// Verify the audit chain at startup so tampering is surfaced early; the
	// result is exposed in the Audit view (§7.11.8).
	auditLog := audit.New(store, nil, auditKey, auditMark)
	if result, verifyErr := auditLog.Verify(ctx); verifyErr != nil {
		log.Error("audit chain verification failed",
			slog.String("state", string(result.State)), slog.Any("error", verifyErr))
	} else {
		log.Info("audit chain verified",
			slog.String("state", string(result.State)), slog.Int("entries", result.VerifiedCount))
	}

	// Multi-host registry: load saved profiles, then ensure the implicit local
	// host exists and try to connect it (best effort — a missing daemon is fine).
	registry := hosts.New(store, connector.New(), auditLog, nil)
	if err := registry.Load(ctx); err != nil {
		return fmt.Errorf("loading host registry: %w", err)
	}
	if _, err := registry.Add(ctx, domain.Host{
		ID:        engine.LocalHostID,
		Name:      "local",
		Transport: domain.TransportLocal,
		Endpoint:  "unix:///var/run/docker.sock",
	}); err != nil {
		return fmt.Errorf("registering local host: %w", err)
	}
	if _, err := registry.Connect(ctx, engine.LocalHostID); err != nil {
		log.Info("local engine not connected", slog.Any("error", err))
	}
	defer func() { _ = registry.Close(context.Background()) }()

	ops := operations.New(registry, store, auditLog, nil)
	jrnl := journal.New(store, auditLog)

	// Roll off resource history beyond the configured retention window. The
	// goroutine is owned by retentionCtx and stops when run() returns.
	retentionCtx, stopRetention := context.WithCancel(ctx)
	defer stopRetention()
	go history.NewRetention(store, cfg.History.ResourceRetention.Duration).
		Run(retentionCtx, time.Hour, nil)

	assets, err := fs.Sub(frontend.Assets, "dist")
	if err != nil {
		return fmt.Errorf("loading embedded frontend: %w", err)
	}

	return app.Run(assets, log, registry, ops, jrnl, store, store, version)
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
