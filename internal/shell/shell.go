// Package shell runs commands and manages long-lived background jobs.
package shell

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"terax/internal/sysproc"
	"terax/internal/types"
)

// Manager tracks in-flight background processes so we can read their
// captured logs and kill them by ID.
type Manager struct {
	mu     sync.RWMutex
	jobs   map[int]*Job
	sess   map[string]*Session
	seq    atomic.Int32
}

// NewManager creates an empty job registry.
func NewManager() *Manager {
	return &Manager{jobs: map[int]*Job{}, sess: map[string]*Session{}}
}

// Session is a long-lived interactive shell process.
type Session struct {
	ID      string
	Cmd     *exec.Cmd
	Stdin   io.WriteCloser
	Stdout  io.Reader
	Stderr  io.Reader
	onData  string
	onExit  string
	mu      sync.Mutex
	closed  atomic.Bool
}

// RunInSession sends `cmd` to the session's stdin. The frontend uses this
// to drive an already-open shell session.
func RunInSession(sessionID int, cmd string) error {
	return errors.New("session not implemented")
}

// CloseSession terminates an interactive session.
func CloseSession(sessionID int) error {
	return errors.New("session not implemented")
}

// Job is a captured output buffer plus the running process handle.
type Job struct {
	ID     int
	Cmd    *exec.Cmd
	buf    bytes.Buffer
	mu     sync.Mutex
	closed atomic.Bool
}

// RunCommand executes `cmd` in the workspace's shell and returns its output.
func RunCommand(ctx context.Context, args types.ShellRunArgs) (string, error) {
	shellCmd, shellArgs := buildShellArgs(args.Command)
	cmd := exec.CommandContext(ctx, shellCmd, shellArgs...)
	sysproc.HideWindow(cmd)
	if args.Cwd != "" {
		cmd.Dir = args.Cwd
	}
	cmd.Env = os.Environ()
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return out.String() + errOut.String(), err
	}
	return out.String(), nil
}

// OpenSession starts an interactive shell session similar to PTY but without
// the pseudo-terminal — used for AI-style "give the agent a terminal"
// sessions where there's no TTY sizing.
func OpenSession(ctx context.Context, m *Manager, args types.ShellSessionOpenArgs) (int, error) {
	shellPath := args.Shell
	if shellPath == "" {
		shellPath = defaultShell()
	}
	cmd := exec.CommandContext(ctx, shellPath)
	sysproc.HideWindow(cmd)
	if args.Cwd != "" {
		cmd.Dir = args.Cwd
	}
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cmd.Stderr = cmd.Stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	id := nextJobID(m)
	job := &Job{ID: id, Cmd: cmd}
	m.mu.Lock()
	m.jobs[id] = job
	m.mu.Unlock()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				job.append(buf[:n])
				// Non-PTY sessions buffer output to the Job; callers
				// retrieve it via Run() / Logs(). No event emission.
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					// log
				}
				break
			}
		}
		_ = cmd.Wait()
	}()
	_ = stdin
	return id, nil
}

// BgSpawn starts a long-running background command and captures its output.
func BgSpawn(ctx context.Context, m *Manager, args types.ShellBgSpawnArgs) (int, error) {
	shellCmd, shellArgs := buildShellArgs(args.Command)
	cmd := exec.CommandContext(ctx, shellCmd, shellArgs...)
	sysproc.HideWindow(cmd)
	if args.Cwd != "" {
		cmd.Dir = args.Cwd
	}
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	id := nextJobID(m)
	job := &Job{ID: id, Cmd: cmd}
	m.mu.Lock()
	m.jobs[id] = job
	m.mu.Unlock()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				job.append(buf[:n])
			}
			if err != nil {
				break
			}
		}
		_ = cmd.Wait()
		job.closed.Store(true)
	}()
	return id, nil
}

// BgLogs returns the captured output for a background job.
func BgLogs(m *Manager, args types.ShellBgLogsArgs) (string, error) {
	m.mu.RLock()
	j, ok := m.jobs[args.Handle]
	m.mu.RUnlock()
	if !ok {
		return "", errors.New("job not found")
	}
	return j.snapshot(), nil
}

// BgList returns the IDs of all known jobs.
func BgList(m *Manager) []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]int, 0, len(m.jobs))
	for id := range m.jobs {
		out = append(out, id)
	}
	return out
}

// BgKill terminates a background job.
func BgKill(m *Manager, args types.ShellBgKillArgs) error {
	m.mu.RLock()
	j, ok := m.jobs[args.Handle]
	m.mu.RUnlock()
	if !ok {
		return errors.New("job not found")
	}
	if j.Cmd != nil && j.Cmd.Process != nil {
		return j.Cmd.Process.Kill()
	}
	return nil
}

func (j *Job) append(b []byte) {
	j.mu.Lock()
	j.buf.Write(b)
	j.mu.Unlock()
}

func (j *Job) snapshot() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.buf.String()
}

func nextJobID(m *Manager) int {
	return int(m.seq.Add(1))
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		if a := os.Getenv("COMSPEC"); a != "" {
			return a
		}
		return "cmd.exe"
	}
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

// buildShellArgs wraps a command string in the host shell's quoting.
func buildShellArgs(cmd string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", cmd}
	}
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash", []string{"-c", cmd}
	}
	return "/bin/sh", []string{"-c", cmd}
}

func intToStr(i int) string {
	return strconv.Itoa(i)
}

// silence unused
var _ = bytes.Buffer{}