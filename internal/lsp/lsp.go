// Package lsp manages language-server-protocol sessions. Each session owns
// a child process speaking JSON-RPC over stdin/stdout; we forward messages
// to the frontend via the unique event names the shim registered.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"terax/internal/sysproc"
	"terax/internal/types"
)

// Session represents a running LSP server.
type Session struct {
	ID            int
	Command       *exec.Cmd
	OnMessageEvent string
	OnExitEvent   string
	Stdin         io.WriteCloser
	mu            sync.Mutex
	closed        atomic.Bool
}

// Manager owns live LSP sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[int]*Session
	seq      atomic.Int32
}

// NewManager creates an empty registry.
func NewManager() *Manager {
	return &Manager{sessions: map[int]*Session{}}
}

// Spawn starts an LSP server child process.
func Spawn(ctx context.Context, m *Manager, args types.LspSpawnArgs) (*Session, error) {
	cmd := exec.CommandContext(ctx, args.Command, args.Args...)
	sysproc.HideWindow(cmd)
	cmd.Env = mergeEnv(args.Env)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderrBuf cappedBuffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	id := int(m.seq.Add(1))
	sess := &Session{
		ID:             id,
		Command:        cmd,
		OnMessageEvent: args.OnMessageEvent,
		OnExitEvent:    args.OnExitEvent,
		Stdin:          stdin,
	}
	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	// Reader goroutine: parse LSP messages using the standard `Content-Length`
	// header framing and emit each body as an event.
	go sess.read(ctx, stdout)
	// Wait goroutine: emits the exit info once the child terminates.
	go sess.wait(ctx, &stderrBuf)

	return sess, nil
}

// Send writes one JSON-RPC message to the LSP server. The frontend formats
// the body (with Content-Length header) before calling.
func (m *Manager) Send(id int, message string) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("lsp session %d not found", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return fmt.Errorf("lsp session %d closed", id)
	}
	_, err := io.WriteString(s.Stdin, message)
	return err
}

// Kill terminates the session.
func (m *Manager) Kill(id int) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	if s.Command != nil && s.Command.Process != nil {
		return s.Command.Process.Kill()
	}
	return nil
}

// KillAll terminates every session. Called from the app's `RunEvent::Exit`
// equivalent.
func (m *Manager) KillAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.Command != nil && s.Command.Process != nil {
			_ = s.Command.Process.Kill()
		}
		s.closed.Store(true)
	}
	m.sessions = map[int]*Session{}
}

// Detect checks whether a binary exists on PATH. The frontend calls this
// before deciding whether to enable a preset.
func Detect(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// HostPID returns the current process's PID, used by the LSP `processId`
// initialization field.
func HostPID() int {
	return getpid()
}

// ResolveRoot walks up from `path` looking for any of the supplied markers.
// If none found, returns "".
func ResolveRoot(path string, markers []string) string {
	if len(markers) == 0 {
		markers = []string{".git"}
	}
	cur := path
	for {
		for _, m := range markers {
			if _, err := stat(cur + "/" + m); err == nil {
				return cur
			}
		}
		parent := parentDir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

func (s *Session) read(ctx context.Context, r io.Reader) {
	br := bufio.NewReader(r)
	for {
		// Read headers.
		var contentLength int
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length: ") {
				n, _ := strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
				contentLength = n
			}
		}
		if contentLength <= 0 {
			continue
		}
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(br, body); err != nil {
			return
		}
		wailsruntime.EventsEmit(ctx, s.OnMessageEvent, body)
	}
}

func (s *Session) wait(ctx context.Context, stderrBuf *cappedBuffer) {
	_ = s.Command.Wait()
	s.closed.Store(true)
	tail := stderrBuf.String()
	exit := types.LspExitInfo{
		StderrTail: tail,
	}
	if s.Command.ProcessState != nil {
		c := s.Command.ProcessState.ExitCode()
		exit.Code = &c
	}
	wailsruntime.EventsEmit(ctx, s.OnExitEvent, exit)
}

func mergeEnv(extra map[string]string) []string {
	base := os_env()
	for k, v := range extra {
		base = append(base, k+"="+v)
	}
	return base
}

// cappedBuffer collects up to 64 KiB of stderr for the LspExitInfo payload.
type cappedBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		// Keep the tail only — that's what the frontend shows.
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// Avoid importing os in the package init by routing through these helpers;
// tests can stub if needed.
var (
	stat        = func(path string) (any, error) { return osStat(path) }
	parentDir   = parentPath
	os_env      = osEnv
	getpid      = osGetpid
	osStat      = defaultStat
	parentPath  = defaultParent
	osEnv       = defaultEnv
	osGetpid    = defaultGetpid
)

func defaultStat(path string) (any, error)                 { return osStatImpl(path) }
func defaultParent(p string) string                        { return parentImpl(p) }
func defaultEnv() []string                                 { return envImpl() }
func defaultGetpid() int                                   { return pidImpl() }

// EncodeError is unused (frontend does its own JSON) but kept for symmetry.
var _ = json.Marshal