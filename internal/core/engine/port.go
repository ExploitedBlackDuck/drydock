// Package engine defines the Engine port — Drydock's interface to a Docker (or
// API-compatible) engine — and the read-only services over it. The port is
// declared here, by its consumer (PROJECT-BOOK §2.1); the Docker Go SDK
// implementation lives in the dockerengine adapter (ADR-0003). The core knows
// nothing of the SDK's types.
package engine

import (
	"context"
	"io"

	"github.com/drydock/drydock/internal/core/domain"
)

// LogOptions configures a container log stream.
type LogOptions struct {
	// Follow keeps the stream open, emitting new lines as they arrive.
	Follow bool
	// Tail is the number of trailing lines to start from (0 = all).
	Tail int
	// Timestamps prefixes each line with its RFC3339 timestamp.
	Timestamps bool
}

// RemoveOptions configures container removal.
type RemoveOptions struct {
	// Force removes a running container (destructive: in-flight work is lost).
	Force bool
	// Volumes also removes anonymous volumes attached to the container.
	Volumes bool
}

// ExecSpec describes a command to run inside a container. The command is always
// argv — never a shell string (ADR-0004).
type ExecSpec struct {
	Cmd        []string
	User       string
	WorkingDir string
	Tty        bool
	// Env is additional environment for the command (NAME=value). Its values are
	// secret material: redacted from the recorded operation and audit (ADR-0023).
	Env []string
}

// ExecStream is the bidirectional I/O of a running exec session: read the
// container's output, write the operator's keystrokes, Resize the remote TTY as
// the terminal pane changes, and Close to end the session.
type ExecStream interface {
	io.ReadWriteCloser
	// Resize sets the remote pseudo-TTY to cols×rows character cells.
	Resize(ctx context.Context, cols, rows uint16) error
	// CloseStdin half-closes the input side — sends EOF to the command without
	// ending the session — so a command that reads stdin to completion (e.g.
	// `cat`, a piped script) finishes while its output still streams back
	// (ADR-0022). Closing the whole stream still ends the session.
	CloseStdin() error
}

// MinAPIVersion is the lowest Docker Engine API version Drydock fully supports.
// A host below it connects in a clearly-labelled reduced-capability mode rather
// than failing opaquely (ADR-0008).
const MinAPIVersion = "1.41"

// LocalHostID is the identifier for the implicit local engine. Multi-host
// profiles with their own identifiers arrive with the hosts registry (P3).
const LocalHostID = "local"

// Engine is the port for a single connected engine. All methods take a context
// so cancellation (UI "stop"/"disconnect") propagates to the request and the
// underlying connection (PROJECT-BOOK §2.3). Read-only in this phase; mutating
// methods are added in P4.
type Engine interface {
	// Info returns the engine and negotiated API version (ADR-0008).
	Info(ctx context.Context) (domain.EngineInfo, error)
	// ListContainers lists all containers, running and stopped.
	ListContainers(ctx context.Context) ([]domain.Container, error)
	// ListImages lists images held on the engine.
	ListImages(ctx context.Context) ([]domain.Image, error)
	// ListVolumes lists volumes known to the engine.
	ListVolumes(ctx context.Context) ([]domain.Volume, error)
	// ListNetworks lists networks known to the engine.
	ListNetworks(ctx context.Context) ([]domain.Network, error)

	// StartContainer starts a stopped container.
	StartContainer(ctx context.Context, id string) error
	// StopContainer gracefully stops a running container.
	StopContainer(ctx context.Context, id string) error
	// RestartContainer restarts a container.
	RestartContainer(ctx context.Context, id string) error
	// KillContainer sends SIGKILL to a container (in-flight work is lost).
	KillContainer(ctx context.Context, id string) error
	// RemoveContainer removes a container per opts.
	RemoveContainer(ctx context.Context, id string, opts RemoveOptions) error

	// ContainerLogs returns a demultiplexed, plain-text log stream. Closing the
	// reader (or cancelling ctx) stops the stream.
	ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)
	// StreamStats samples a container's live stats into sink until ctx is
	// cancelled, then returns. No goroutine outlives the call.
	StreamStats(ctx context.Context, id string, sink func(domain.ResourceSample)) error
	// Exec starts a command (argv) inside a container and returns its stream.
	Exec(ctx context.Context, id string, spec ExecSpec) (ExecStream, error)

	// DiskUsage returns the engine's `system df` for the prune-impact preview.
	DiskUsage(ctx context.Context) (domain.DiskUsage, error)
	// PruneContainers removes stopped containers, returning bytes reclaimed.
	PruneContainers(ctx context.Context) (int64, error)
	// PruneImages removes dangling images, or all unused images when all is true.
	PruneImages(ctx context.Context, all bool) (int64, error)
	// PruneBuildCache removes unused build cache, returning bytes reclaimed.
	PruneBuildCache(ctx context.Context) (int64, error)
	// RemoveVolume removes a single named volume. There is deliberately no bulk
	// volume-prune: volumes are only ever removed one at a time (ADR-0011, §7.4).
	RemoveVolume(ctx context.Context, name string, force bool) error

	// ComposeUp brings a Compose stack up by starting every (stopped) container
	// the project's label selects. The stack is acted on as a unit over the SDK —
	// there is no compose-binary shell-out, so it works identically for local and
	// SSH-tunnelled hosts (ADR-0003/0004, §7.11.6).
	ComposeUp(ctx context.Context, project string) error
	// ComposeDown takes a Compose stack down: it stops and removes the project's
	// containers. When volumes is true it also removes the project's named volumes
	// (compose `down -v`), which the caller must have acknowledged (§7.5).
	ComposeDown(ctx context.Context, project string, volumes bool) error

	// SnapshotVolume streams a read-only tar of the volume's contents to the file
	// dest, via a throwaway helper container that mounts the volume read-only and
	// runs `tar` (argv, never a shell — ADR-0004/0020). The helper image is
	// digest-pinned; it is preferred if already present and otherwise pulled,
	// failing closed on an air-gapped host. The helper is always removed, even on
	// cancellation. Returns the number of bytes written.
	SnapshotVolume(ctx context.Context, volume, helperImage, dest string) (int64, error)
	// RestoreVolume extracts the tar at src into the volume (read-write) via a
	// throwaway helper container.
	RestoreVolume(ctx context.Context, volume, helperImage, src string) error

	// RegistryDigest returns the registry's current digest for an image reference,
	// resolved through the daemon's distribution-inspect endpoint so it reflects
	// the host's registry reachability and credentials, not the desktop's
	// (ADR-0019). It is operator-initiated — never called in the background.
	RegistryDigest(ctx context.Context, imageRef string) (string, error)

	// StreamEvents delivers engine events to sink until ctx is cancelled. It
	// drives live UI updates and restart-loop detection (§7.6).
	StreamEvents(ctx context.Context, sink func(domain.EngineEvent)) error

	// Close releases the engine connection and any associated resources.
	Close() error
}
