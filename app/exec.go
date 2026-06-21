package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/drydock/drydock/internal/core/engine"
)

// Interactive exec terminal (PROJECT-BOOK §7.11.4): a reliable in-app shell into
// a container over the API exec endpoints, working over SSH-tunnelled remotes.
// The command is argv (a shell *binary*, never `sh -c` of operator input —
// ADR-0004); the operator types into the allocated TTY. Output is streamed to
// the frontend base64-encoded so raw terminal bytes survive the event bus
// intact; keystrokes come back the same way.

// defaultExecShell is tried when the caller does not name one. It is a program
// to run (argv[0]), not a shell string to interpret.
const defaultExecShell = "/bin/sh"

// ErrNoExecSession is returned when an input/resize/stop targets an unknown or
// already-closed session.
var ErrNoExecSession = errors.New("no such exec session")

type execSession struct {
	stream engine.ExecStream
	cancel context.CancelFunc
}

func execOutputEvent(id string) string { return "exec:" + id }
func execExitEvent(id string) string   { return "exec-exit:" + id }

// StartExec opens an interactive exec session into a container and streams its
// output on "exec:<sessionId>", returning the session id. shell is the program
// to run (empty → /bin/sh). The session is guarded by observe-mode and audited
// like any mutation, via the operations service.
func (a *App) StartExec(hostID, containerID, shell string) (string, error) {
	if shell == "" {
		shell = defaultExecShell
	}

	// A long-lived context: the hijacked exec connection must outlive a single
	// request and is torn down only by StopExec or app shutdown.
	ctx, cancel := context.WithCancel(a.baseCtx())
	stream, err := a.ops.Exec(ctx, hostID, containerID, engine.ExecSpec{
		Cmd: []string{shell},
		Tty: true,
	})
	if err != nil {
		cancel()
		return "", err
	}

	id := newSessionID()
	a.setExec(id, &execSession{stream: stream, cancel: cancel})

	go a.pumpExecOutput(ctx, id, stream)
	return id, nil
}

// pumpExecOutput reads container output until the stream ends, emitting each
// chunk base64-encoded, then signals exit and cleans the session up.
func (a *App) pumpExecOutput(ctx context.Context, id string, stream engine.ExecStream) {
	defer a.clearExec(id)
	buf := make([]byte, 32*1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			chunk := base64.StdEncoding.EncodeToString(buf[:n])
			// Emit on the app-lifetime event bus, not the stream ctx.
			a.runtime.EmitEvent(a.baseCtx(), execOutputEvent(id), chunk) //nolint:contextcheck // app-lifetime event bus
		}
		if err != nil {
			break
		}
		if ctx.Err() != nil {
			break
		}
	}
	a.runtime.EmitEvent(a.baseCtx(), execExitEvent(id), "") //nolint:contextcheck // app-lifetime event bus
}

// SendExecInput writes operator keystrokes (base64-encoded bytes) to a session.
func (a *App) SendExecInput(sessionID, dataB64 string) error {
	session, ok := a.getExec(sessionID)
	if !ok {
		return ErrNoExecSession
	}
	data, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return err
	}
	for len(data) > 0 {
		n, werr := session.stream.Write(data)
		if werr != nil {
			return werr
		}
		data = data[n:]
	}
	return nil
}

// CloseExecInput half-closes a session's stdin (sends EOF) without ending the
// session, so a command reading stdin to completion finishes while its output
// continues to stream (ADR-0022).
func (a *App) CloseExecInput(sessionID string) error {
	session, ok := a.getExec(sessionID)
	if !ok {
		return ErrNoExecSession
	}
	return session.stream.CloseStdin()
}

// ResizeExec sets the session's remote TTY to cols×rows.
func (a *App) ResizeExec(sessionID string, cols, rows int) error {
	session, ok := a.getExec(sessionID)
	if !ok {
		return ErrNoExecSession
	}
	ctx, cancel := a.requestCtx()
	defer cancel()
	return session.stream.Resize(ctx, clampDim(cols), clampDim(rows))
}

// StopExec ends a session: it closes the stream and cancels its context so the
// connection (and any tunnel it rides) is released.
func (a *App) StopExec(sessionID string) {
	a.execMu.Lock()
	session, ok := a.execs[sessionID]
	delete(a.execs, sessionID)
	a.execMu.Unlock()
	if !ok {
		return
	}
	_ = session.stream.Close()
	session.cancel()
}

func (a *App) setExec(id string, s *execSession) {
	a.execMu.Lock()
	defer a.execMu.Unlock()
	a.execs[id] = s
}

func (a *App) getExec(id string) (*execSession, bool) {
	a.execMu.Lock()
	defer a.execMu.Unlock()
	s, ok := a.execs[id]
	return s, ok
}

// clearExec drops a session whose output pump has ended, closing its stream so
// the connection is not left half-open after the remote command exits.
func (a *App) clearExec(id string) {
	a.execMu.Lock()
	session, ok := a.execs[id]
	delete(a.execs, id)
	a.execMu.Unlock()
	if ok {
		_ = session.stream.Close()
		session.cancel()
	}
}

// clampDim keeps a terminal dimension within a sane uint16 range.
func clampDim(v int) uint16 {
	switch {
	case v < 1:
		return 1
	case v > 1000:
		return 1000
	default:
		return uint16(v) //nolint:gosec // bounded to [1,1000] above
	}
}

func newSessionID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
