package app

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/drydock/drydock/internal/core/hosts"
	"github.com/drydock/drydock/internal/core/journal"
	"github.com/drydock/drydock/internal/core/operations"
	"github.com/drydock/drydock/shell"
)

// Window defaults. The window has defined minimum dimensions (PROJECT-BOOK
// §7.11.1) so the dense control surface never collapses below usability.
const (
	defaultWidth  = 1100
	defaultHeight = 720
	minWidth      = 940
	minHeight     = 560
)

// Run constructs the binding layer and starts the desktop application, blocking
// until the window closes. assets is the embedded, built frontend; registry is
// the multi-host registry, ops performs guarded mutations, and samples persists
// resource history.
func Run(assets fs.FS, log *slog.Logger, registry *hosts.Registry, ops *operations.Service, jrnl *journal.Service, samples SampleStore, backup BackupStore, timeline TimelineStore, snapshotHelper string, version string) error {
	application := New(log, shell.WailsRuntime{}, version, registry, ops, jrnl, samples, backup, timeline, snapshotHelper)

	err := wails.Run(&options.App{
		Title:     "Drydock",
		Width:     defaultWidth,
		Height:    defaultHeight,
		MinWidth:  minWidth,
		MinHeight: minHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: application.startup,
		Bind: []any{
			application,
		},
	})
	if err != nil {
		return fmt.Errorf("running desktop application: %w", err)
	}
	return nil
}
