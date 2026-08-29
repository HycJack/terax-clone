// Package lsp manages language-server-protocol sessions. Each session owns
// a child process speaking JSON-RPC over stdin/stdout; we forward messages
// to the frontend via the unique event names the shim registered.
package lsp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
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
	bin, err := lookPath(args.Command)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args.Args...)
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
	var stderrBuf = newCappedBuffer(64 * 1024)
	cmd.Stderr = stderrBuf
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
	go sess.wait(ctx, stderrBuf)

	return sess, nil
}

// Send writes one JSON-RPC message to the LSP server. The LSP wire protocol
// frames every message with a `Content-Length` header; codemirror-
// languageserver hands the transport the bare JSON body, so we add the
// framing here before writing to the child's stdin.
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
	_, err := io.WriteString(s.Stdin, frameMessage(message))
	return err
}

// frameMessage adds the LSP `Content-Length` framing header. Already-framed
// messages pass through untouched so a future client-side framing change
// stays safe.
func frameMessage(message string) string {
	if strings.HasPrefix(message, "Content-Length:") {
		return message
	}
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(message), message)
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

// Detect checks whether a binary exists on the augmented PATH. The frontend
// calls this before deciding whether to enable a preset.
func Detect(command string) bool {
	_, err := lookPath(command)
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
	// LSP headers are case-insensitive per the spec (RFC 9110 / LSP), so match
	// on the lowercased name. Guard the declared body size: a misbehaving or
	// hostile server could otherwise ask us to allocate an unbounded buffer.
	const maxBodyBytes = 32 * 1024 * 1024 // 32 MB — far above any real LSP message
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
			header := line
			idx := -1
			if i := strings.IndexByte(line, ':'); i >= 0 {
				header = line[:i]
				idx = i
			}
			if strings.EqualFold(strings.TrimSpace(header), "Content-Length") && idx >= 0 {
				value := strings.TrimSpace(line[idx+1:])
				if n, err := strconv.Atoi(value); err == nil && n > 0 {
					contentLength = n
				}
			}
		}
		if contentLength <= 0 {
			continue
		}
		if contentLength > maxBodyBytes {
			// Give up on the connection rather than allocating an absurd buffer.
			return
		}
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(br, body); err != nil {
			return
		}
		if app := application.Get(); app != nil {
			app.Event.Emit(s.OnMessageEvent, body)
		}
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
	if app := application.Get(); app != nil {
		app.Event.Emit(s.OnExitEvent, exit)
	}
}

func mergeEnv(extra map[string]string) []string {
	base := os_env()
	path := augmentedPath()
	replaced := false
	for i, kv := range base {
		k, _, ok := strings.Cut(kv, "=")
		if ok && k == "PATH" {
			base[i] = "PATH=" + path
			replaced = true
			break
		}
	}
	if !replaced {
		base = append(base, "PATH="+path)
	}
	for k, v := range extra {
		base = append(base, k+"="+v)
	}
	return base
}

// toolDirs returns directories that commonly hold language-server binaries
// (gopls, clangd, typescript-language-server, rust-analyzer, ...) but that
// are frequently absent from the PATH of a GUI-launched app or a minimal
// shell. GOPATH/bin and $HOME/go/bin cover the canonical Go tool install.
func toolDirs() []string {
	home, _ := homeDir()
	dirs := []string{}
	add := func(d string) {
		if d != "" && d != "." {
			dirs = append(dirs, d)
		}
	}
	add(getenv("GOPATH") + "/bin")
	add(home + "/go/bin")
	add("/usr/local/go/bin")
	add("/opt/homebrew/bin")
	add("/usr/local/bin")
	add(home + "/.cargo/bin")
	add(home + "/.local/bin")
	add(home + "/.bun/bin")
	add(home + "/.npm-global/bin")
	return dirs
}

// augmentedPath prepends toolDirs entries to the process PATH, keeping any
// directory that is already present in place (no duplicates).
func augmentedPath() string {
	path := getenv("PATH")
	parts := strings.Split(path, string(os.PathListSeparator))
	has := func(dir string) bool {
		for _, p := range parts {
			if p == dir {
				return true
			}
		}
		return false
	}
	for _, d := range toolDirs() {
		if !has(d) {
			path = d + string(os.PathListSeparator) + path
		}
	}
	return path
}

// lookPath resolves `command` against the augmented PATH. Slash-containing
// paths (absolute or relative) are used as-is.
func lookPath(command string) (string, error) {
	if strings.ContainsRune(command, '/') {
		if _, err := stat(command); err == nil {
			return command, nil
		}
		return "", fmt.Errorf("lsp: command %q not found", command)
	}
	for _, dir := range strings.Split(augmentedPath(), string(os.PathListSeparator)) {
		if dir == "" {
			continue
		}
		candidate := dir + "/" + command
		if _, err := stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("lsp: %q not found on PATH", command)
}

// cappedBuffer collects up to `max` bytes of stderr, keeping only the tail
// once it fills up. The tail is what the frontend shows on LSP exit.
type cappedBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func newCappedBuffer(max int) *cappedBuffer {
	return &cappedBuffer{max: max}
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
	getenv      = defaultGetenv
	homeDir     = defaultHomeDir
)

func defaultStat(path string) (any, error)                 { return osStatImpl(path) }
func defaultParent(p string) string                        { return parentImpl(p) }
func defaultEnv() []string                                 { return envImpl() }
func defaultGetpid() int                                   { return pidImpl() }
func defaultGetenv(key string) string                      { return os.Getenv(key) }
func defaultHomeDir() (string, error)                      { return os.UserHomeDir() }