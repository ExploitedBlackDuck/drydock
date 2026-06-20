// Package app is the Wails binding layer: the only place besides shell/ that
// imports Wails. It is deliberately thin (PROJECT-BOOK §3) — it wires the
// composed core to the frontend and translates between them. Business logic
// here is a defect; it belongs in internal/core.
package app

import (
	"context"
	"log/slog"

	"github.com/drydock/drydock/internal/core/engine"
	"github.com/drydock/drydock/shell"
)

// App is the binding target exposed to the frontend. Its exported methods
// become generated, typed frontend bindings.
type App struct {
	log     *slog.Logger
	runtime shell.Runtime
	version string
	engine  engine.Engine
	ctx     context.Context
}

// New constructs the binding layer with its injected dependencies. Nothing is
// constructed globally (PROJECT-BOOK §2.3); main is the composition root.
func New(log *slog.Logger, runtime shell.Runtime, version string, eng engine.Engine) *App {
	return &App{
		log:     log,
		runtime: runtime,
		version: version,
		engine:  eng,
	}
}

// startup is invoked by the desktop shell once the window and runtime are
// ready. It receives the binding context used for the app's lifetime, which
// scopes engine requests so cancellation propagates on quit (PROJECT-BOOK §2.3).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.log.Info("drydock starting", slog.String("version", a.version))
	a.runtime.EmitEvent(ctx, EventAppReady, a.version)
}

// Version reports the running Drydock version. Bound to the frontend.
func (a *App) Version() string {
	return a.version
}
