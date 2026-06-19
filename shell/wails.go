package shell

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsRuntime implements Runtime against the live Wails v2 runtime. It is
// stateless; the per-call context carries the binding.
type WailsRuntime struct{}

// EmitEvent forwards the event to the Wails runtime event bus.
func (WailsRuntime) EmitEvent(ctx context.Context, name string, data ...any) {
	wailsruntime.EventsEmit(ctx, name, data...)
}
