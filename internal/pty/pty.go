// Package pty manages pseudo-terminal sessions.
//
// Each session owns a process and a bidirectional stream that proxies its
// stdio. Output is forwarded to the frontend via a unique event name
// registered through a Channel (see the JS-side shim).
//
// Backends:
//
//   Windows 10 1809 / 11 / Server 2019+  → `aymanbagabas/go-pty` (ConPTY)
//   Older Windows                        → plain `exec.Cmd` + anonymous
//                                          pipes (no real PTY; ANSI and
//                                          Ctrl+C escape sequences are
//                                          not interpreted)
//   Unix                                  → `aymanbagabas/go-pty`
//                                          (POSIX openpty(3) via
//                                          creack/pty underneath)
package pty

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	gopty "github.com/aymanbagabas/go-pty"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Session is a single PTY instance.
type Session struct {
	ID        int
	Shell     string
	Args      []string
	Cwd       string
	Workspace string

	// master is the read/write end connected to the child's stdio. On the
	// ConPTY / Unix PTY path it's a `gopty.Pty`; on the legacy Windows
	// pipe-fallback path it's a `*pipeBackend`. Both satisfy the same
	// interface used by Read/Write/Close.
	master io.ReadWriteCloser

	// proc is the running child. Use `waitProc` to block until it exits
	// and `procState` to read its exit code.
	proc      childProc
	procState childState

	// emitEvent is a stored callback for forwarding PTY output to the frontend.
	// The callback receives the event name and data; the implementation is
	// provided by the App at session creation time via the v3 event system.
	emitEvent func(event string, data interface{})

	mu      sync.Mutex
	closed  atomic.Bool
	started atomic.Bool
}

// childProc is the subset of exec.Cmd we use. Defined as an interface so the
// gopty.Cmd (which exposes Wait + ProcessState) and the fallback *exec.Cmd
// can both satisfy it.
type childProc interface {
	Wait() error
}

type childState interface {
	ExitCode() int
}

// Manager owns all live PTY sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[int]*Session
	counter  atomic.Int32
}

// NewManager creates an empty session registry.
func NewManager() *Manager {
	return &Manager{sessions: map[int]*Session{}}
}

// DefaultShell returns the OS-default shell. Used when the caller doesn't
// pin a specific one.
func DefaultShell() string {
	if runtime.GOOS == "windows" {
		// Prefer pwsh > powershell > cmd — matches the Rust frontend behavior.
		for _, candidate := range []string{
			"C:\\Program Files\\PowerShell\\7\\pwsh.exe",
			"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return os.Getenv("COMSPEC")
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	return sh
}

// Open starts a shell attached to a PTY (or plain pipes on legacy Windows).
//
// Output is forwarded to the frontend via two Wails events named
// `pty:<id>` (per-read payload) and `pty:exit:<id>` (final exit code).
// The frontend subscribes to those names directly; no Channel attachment
// IDs are needed.
func (m *Manager) Open(
	ctx context.Context,
	cols, rows int,
	cwd, workspace, shell string,
) (*Session, error) {
	if shell == "" {
		shell = DefaultShell()
	}
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}

	var cmdArgs []string
	// The shell-init integration in startChild decides the right argv per
	// shell (bash gets --rcfile -i, PowerShell gets -File profile.ps1).
	// We only pre-pend `-l` for POSIX shells that don't have an integration
	// of their own — bash on macOS is the canonical case. Anything with
	// an integration will append its own argv in startChild.
	switch runtime.GOOS {
	case "windows":
		cmdArgs = nil
	default:
		if classifyShell(shell) == "bash" {
			// bash ignores --rcfile when started with -l; rely on the
			// rcfile to source /etc/profile and ~/.bash_profile instead.
			cmdArgs = nil
		} else {
			cmdArgs = []string{"-l"}
		}
	}

	master, child, state, err := startChild(shell, cmdArgs, cwd, cols, rows)
	if err != nil {
		return nil, err
	}

	id := int(m.counter.Add(1))
	sess := &Session{
		ID:        id,
		Shell:     shell,
		Args:      cmdArgs,
		Cwd:       cwd,
		Workspace: workspace,
		master:    master,
		proc:      child,
		procState: state,
	}

	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	sess.emitEvent = func(event string, data interface{}) {
		if app := application.Get(); app != nil {
			app.Event.Emit(event, data)
		}
	}
	// Never pump here. `Start` launches the pump only after the frontend
	// has subscribed to `pty:<id>`, so early shell output (banner + first
	// prompt + OSC 7/133 markers that sync the directory tree) can't race
	// ahead of Wails's unbuffered event bus and be dropped.
	return sess, nil
}

// Start begins forwarding a session to a fresh pump goroutine. Idempotent.
// It must be called AFTER the frontend has registered its `pty:<id>` listener
// so the shell's startup bytes aren't lost on the unbuffered event bus.
func (s *Session) Start(ctx context.Context) {
	if s.started.Swap(true) {
		return
	}
	go s.pump(ctx)
}

// Start signals a session's pump. No-op when already running.
func (m *Manager) Start(ctx context.Context, id int) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %d not found", id)
	}
	s.Start(ctx)
	return nil
}

// startChild spawns the shell, attaches its stdio to a master endpoint and
// returns (master, child, state, err). The concrete backend depends on the
// platform and the kernel's ConPTY support.
func startChild(shell string, args []string, cwd string, cols, rows int) (io.ReadWriteCloser, childProc, childState, error) {
	// Resolve shell-integration args/env at the top so both the ConPTY
	// and the legacy pipeBackend paths below can apply them. The frontend
	// uses the OSC 7 cwd + OSC 133 markers these snippets emit to keep
	// the directory tree root in sync with the active terminal.
	integ, _ := integrationFor(shell)
	if len(integ.extraArgs) > 0 {
		args = append(args, integ.extraArgs...)
	}

	if runtime.GOOS == "windows" && !hasConPTY() {
		// Legacy Windows: kernel32 doesn't export CreatePseudoConsole.
		// Fall back to plain pipes; no PTY semantics.
		be, err := newPipeBackend(shell, args, cwd)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("pty start: %w", err)
		}
		if len(integ.extraEnv) > 0 {
			be.cmd.Env = append(be.cmd.Env, integ.extraEnv...)
		}
		return be, be.cmd, execState{be.cmd}, nil
	}

	// Modern path: ConPTY on Windows, openpty on Unix. Both go through
	// the same go-pty entrypoint.
	pt, err := gopty.New()
	if err != nil {
		// gopty.New() on Windows is the call that fails on 1709 builds.
		// Try the pipe fallback one more time before giving up.
		if runtime.GOOS == "windows" {
			be, perr := newPipeBackend(shell, args, cwd)
			if perr == nil {
				if len(integ.extraEnv) > 0 {
					be.cmd.Env = append(be.cmd.Env, integ.extraEnv...)
				}
				return be, be.cmd, execState{be.cmd}, nil
			}
		}
		return nil, nil, nil, fmt.Errorf("pty start: %w", err)
	}
	if err := pt.Resize(cols, rows); err != nil {
		_ = pt.Close()
		return nil, nil, nil, fmt.Errorf("pty resize: %w", err)
	}

	cmd := pt.Command(shell, args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	if len(integ.extraEnv) > 0 {
		cmd.Env = append(cmd.Env, integ.extraEnv...)
	}
	cmd.SysProcAttr = procAttr()

	if err := cmd.Start(); err != nil {
		_ = pt.Close()
		return nil, nil, nil, fmt.Errorf("pty start: %w", err)
	}
	return pt, cmd, goptyCmdState{cmd}, nil
}
// execState is a childState wrapper around *exec.Cmd.ProcessState that
// tolerates Wait() not being called on the same goroutine.
type execState struct{ cmd *exec.Cmd }

func (s execState) ExitCode() int {
	st := s.cmd.ProcessState
	if st == nil {
		return -1
	}
	return st.ExitCode()
}

// goptyCmdState wraps a go-pty Cmd and reads its ProcessState after Wait.
type goptyCmdState struct{ cmd *gopty.Cmd }

func (s goptyCmdState) ExitCode() int {
	st := s.cmd.ProcessState
	if st == nil {
		return -1
	}
	return st.ExitCode()
}

// Write forwards stdin data to the master.
func (s *Session) Write(data []byte) error {
	if s.closed.Load() {
		return errors.New("pty closed")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.master == nil {
		return errors.New("master nil")
	}
	_, err := s.master.Write(data)
	return err
}

// Resize changes the terminal geometry. No-op on the legacy pipe backend.
func (s *Session) Resize(cols, rows int) error {
	if s.closed.Load() {
		return nil
	}
	if r, ok := s.master.(interface {
		Resize(width, height int) error
	}); ok {
		return r.Resize(cols, rows)
	}
	return nil
}

// Close terminates the child and releases the master fd.
func (s *Session) Close() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	if s.master != nil {
		_ = s.master.Close()
	}
	if s.proc != nil {
		// Kick the wait loop so pump() doesn't block forever on a child
		// that's already been killed.
		if w, ok := s.proc.(interface {
			Process() *os.Process
		}); ok {
			if p := w.Process(); p != nil {
				_ = killProcessTree(p.Pid)
			}
		}
	}
}

// CloseAll signals every live session. Called on app exit / `pty_close_all`.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		s.Close()
	}
	m.sessions = map[int]*Session{}
}

// HasForeground checks whether the child is running. Simple heuristic.
func (m *Manager) HasForeground(id int) bool {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok || s == nil {
		return false
	}
	if r, ok := s.proc.(interface {
		Process() *os.Process
	}); ok {
		if p := r.Process(); p != nil {
			return probeProcess(p)
		}
	}
	return false
}

func (s *Session) pump(ctx context.Context) {
	dataEvent := fmt.Sprintf("pty:%d", s.ID)
	exitEvent := fmt.Sprintf("pty:exit:%d", s.ID)
	buf := make([]byte, 32*1024)
	for {
		n, err := s.master.Read(buf)
		if n > 0 {
			// simple-shell-style envelope: {"data": "<base64>"} so the JS
			// side can decode the exact bytes the shell wrote.
			payload := make([]byte, n)
			copy(payload, buf[:n])
			b64 := base64.StdEncoding.EncodeToString(payload)
			if s.emitEvent != nil {
				s.emitEvent(dataEvent, map[string]string{"data": b64})
			}
		}
		if err != nil {
			break
		}
	}
	s.waitAndExit(ctx, exitEvent)
}

func (s *Session) waitAndExit(ctx context.Context, exitEvent string) {
	_ = s.proc.Wait()
	code := s.procState.ExitCode()
	if s.emitEvent != nil {
		s.emitEvent(exitEvent, code)
	}
}

// Write forwards stdin bytes to a session.
func (m *Manager) Write(id int, data []byte) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %d not found", id)
	}
	return s.Write(data)
}

// Resize changes the PTY geometry of a session.
func (m *Manager) Resize(id, cols, rows int) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %d not found", id)
	}
	return s.Resize(cols, rows)
}

// Close terminates a session by ID.
func (m *Manager) Close(id int) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if ok {
		s.Close()
	}
}

// ShellName returns the basename of the shell binary for a session.
func (m *Manager) ShellName(id int) string {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return ""
	}
	return filepathBase(s.Shell)
}

// ListShells returns the shells present on this machine.
func (m *Manager) ListShells() []string {
	out := []string{}
	candidates := []string{"pwsh", "powershell", "cmd", "bash", "zsh", "fish", "sh"}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			out = append(out, c)
		}
	}
	return out
}

func filepathBase(p string) string {
	if p == "" {
		return ""
	}
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}

func procAttr() *syscall.SysProcAttr {
	return platformProcAttr()
}

func killProcessTree(pid int) error {
	if runtime.GOOS == "windows" {
		p, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return killProcessTreeWindows(p)
	}
	return unixKillGroup(pid)
}