// Package frontend embeds the built Svelte application. The embed lives here,
// next to the asset directory, because //go:embed cannot traverse parent
// directories from cmd/drydock/main.go. The dist directory is produced by the
// frontend build (`task build`); a fresh checkout contains only a placeholder.
package frontend

import "embed"

// Assets is the built frontend served by the Wails asset server.
//
//go:embed all:dist
var Assets embed.FS
