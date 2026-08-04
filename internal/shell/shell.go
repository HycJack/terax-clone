// Package shell runs commands and manages long-lived background jobs.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"terax/internal/sysproc"
	"terax/internal/types"
)

// Manager tracks in-flight background processes so we can read their
// captured logs and kill them by ID. It also tracks interactive shell
// sessions used by the agent's `bash_run` tool.
type Manager struct {
	mu       sync.RWMutex
	jobs     map[int]*Job
	sessions map[int]*Session
	seq      atomic.Int32
}

// NewManager creates an empty job registry.
func NewManager() *Manager {
	return &Manager{jobs: map[int]*Job{}, sessions: map[int]*Session{}}
}

// Session tracks the cwd for a logical shell session. Each RunInSession
// call spawns a fresh shell process with the saved cwd — this avoids the
// command-echo pollution of interactive cmd.exe while preserving the
// user-facing semantics that cwd persists across calls.
type Session struct {
	ID     int
	cwd    string
	shell  string // "bash" | "cmd.exe" | ...
	mu     sync.Mutex
	closed atomic.Bool
}

// RunInSession executes a command in the session's shell. The session's
// cwd persists across calls (so `cd foo` then `pwd` works). Each call
// spawns a fresh shell process to avoid interactive-echo pollution.
func RunInSession(m *Manager, args types.ShellSessionRunArgs) (*types.ShellSessionResult, error) {
	m.mu.RLock()
	sess, ok := m.sessions[args.ID]
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New("session not found")
	}
	if sess.closed.Load() {
		return nil, errors.New("session closed")
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	cwd := args.Cwd
	if cwd == "" {
		cwd = sess.cwd
	}

	timeout := time.Duration(args.TimeoutSecs) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	result, err := sess.runCommand(args.Command, cwd, timeout)
	if err != nil {
		return nil, err
	}
	if result.CwdAfter != "" {
		sess.cwd = result.CwdAfter
	}
	return result, nil
}

// CloseSession terminates an interactive session.
func CloseSession(m *Manager, id int) error {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return errors.New("session not found")
	}
	delete(m.sessions, id)
	m.mu.Unlock()
	sess.closed.Store(true)
	return nil
}

// Job is a captured output buffer plus the running process handle.
type Job struct {
	ID         int
	Cmd        *exec.Cmd
	Command    string
	Cwd        string
	StartedAt  int64 // unix millis
	exitCode   int
	exitCodeOK bool
	buf        bytes.Buffer
	mu         sync.Mutex
	closed     atomic.Bool
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

// OpenSession creates a new interactive shell session. The session
// tracks cwd across multiple RunInSession calls; each call spawns a
// fresh shell process to avoid interactive-echo pollution.
func OpenSession(_ context.Context, m *Manager, args types.ShellSessionOpenArgs) (int, error) {
	shellPath := args.Shell
	if shellPath == "" {
		shellPath = defaultShell()
	}

	id := nextJobID(m)
	sess := &Session{
		ID:    id,
		cwd:   args.Cwd,
		shell: shellName(shellPath),
	}

	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

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
	job := &Job{
		ID:        id,
		Cmd:       cmd,
		Command:   args.Command,
		Cwd:       args.Cwd,
		StartedAt: time.Now().UnixMilli(),
	}
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
		err = cmd.Wait()
		job.mu.Lock()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				job.exitCode = exitErr.ExitCode()
			} else {
				job.exitCode = -1
			}
		} else {
			job.exitCode = 0
			job.exitCodeOK = true
		}
		job.mu.Unlock()
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

// BgProcessInfo is the JSON shape the frontend expects from `shell_bg_list`.
type BgProcessInfo struct {
	Handle    int    `json:"handle"`
	Command   string `json:"command"`
	Cwd       string `json:"cwd"`
	StartedAt int64  `json:"started_at_ms"`
	Exited    bool   `json:"exited"`
	ExitCode  *int   `json:"exit_code"`
}

// BgList returns metadata for every background job.
func BgList(m *Manager) []BgProcessInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]BgProcessInfo, 0, len(m.jobs))
	for id, j := range m.jobs {
		j.mu.Lock()
		var ec *int
		if j.closed.Load() {
			code := j.exitCode
			ec = &code
		}
		info := BgProcessInfo{
			Handle:    id,
			Command:   j.Command,
			Cwd:       j.Cwd,
			StartedAt: j.StartedAt,
			Exited:    j.closed.Load(),
			ExitCode:  ec,
		}
		j.mu.Unlock()
		out = append(out, info)
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

// =========================================================================
// Session helpers
// =========================================================================

// runCommand spawns a fresh shell process to execute the command. The
// command is wrapped with sentinel markers so we can capture stdout,
// exit code, and post-command cwd in a single pass.
//
// Marker protocol (all on stdout):
//
//	<command output>
//	<exitMarker><exit-code>__
//	<cwdMarker><cwd>__
//	<doneMarker>
//
// Everything before <exitMarker> is the command's stdout.
//
// Windows notes:
//   - `cmd.exe /C "<script>"` has two quoting bugs that break our marker
//     protocol:
//       1. Go's argument builder wraps the inline script in extra quotes
//          for cmd.exe; cmd.exe's /C then strips the outer quotes via its
//          special handling, mangling any inner double-quoted paths like
//          `cd /d "C:\Users\..."` (they end up with stray backslashes).
//       2. The user command itself may contain quoted paths, and these
//          get re-mangled by the same /C processing.
//     The fix is to write the script to a temp .bat file and execute
//     that instead. Batch files are parsed by cmd.exe's normal batch
//     grammar, which respects quoting correctly.
//   - cmd.exe only expands %errorlevel% inside batch files (which our
//     temp file satisfies).
//   - We use sequential lines instead of `&&` so a non-zero exit from
//     the user command doesn't skip the marker emissions.
func (s *Session) runCommand(command, cwd string, timeout time.Duration) (*types.ShellSessionResult, error) {
	const maxOutput = 4 * 1024 * 1024 // 4 MB cap per command

	id := time.Now().UnixNano()
	exitMarker := fmt.Sprintf("__TERAX_EXIT_%d__", id)
	cwdMarker := fmt.Sprintf("__TERAX_CWD_%d__", id)
	doneMarker := fmt.Sprintf("__TERAX_DONE_%d__", id)

	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var cmd *exec.Cmd
	var cleanup func()
	if s.shell == "cmd.exe" {
		// Build the script as batch-file contents. Each line is its own
		// command so failures don't short-circuit the marker emissions.
		script := strings.Join([]string{
			"@echo off",
			command,
			"call echo " + exitMarker + "%errorlevel%__",
			"echo " + cwdMarker,
			"cd",
			"echo " + doneMarker,
		}, "\r\n") + "\r\n"
		batchPath, err := writeTempBatch(script)
		if err != nil {
			return nil, err
		}
		cleanup = func() { _ = os.Remove(batchPath) }
		cmd = exec.CommandContext(ctx, batchPath)
	} else {
		// Unix: use bash -c with the inline script. No /C quoting bug here.
		cwdPrefix := ""
		if cwd != "" {
			cwdPrefix = "cd " + quoteShell(cwd) + " && "
		}
		script := fmt.Sprintf(
			"%s%s; printf '%s%%s__\\n' \"$?\"; printf '%s%%s__\\n' \"$(pwd)\"; printf '%s\\n'",
			cwdPrefix, command, exitMarker, cwdMarker, doneMarker)
		cmd = exec.CommandContext(ctx, "bash", "-c", script)
	}
	if cleanup != nil {
		defer cleanup()
	}
	sysproc.HideWindow(cmd)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // merge stderr into stdout for capture
	runErr := cmd.Run()
	timedOut := ctx.Err() == context.DeadlineExceeded

	return s.parseMarkers(out.String(), exitMarker, cwdMarker, doneMarker, timedOut, maxOutput, runErr)
}

// writeTempBatch writes `contents` to a temp .bat file and returns its
// absolute path. The file is created with restrictive permissions so the
// user command (which may contain sensitive content) doesn't leak.
func writeTempBatch(contents string) (string, error) {
	f, err := os.CreateTemp("", "terax-shell-*.bat")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if runtime.GOOS == "windows" {
		// Belt-and-suspenders: deny read on the temp file so other users
		// on the machine can't snoop on command output.
		_ = os.Chmod(f.Name(), 0o600)
	}
	if _, err := f.WriteString(contents); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// parseMarkers extracts stdout, exit code, and cwd from the shell output
// using the sentinel markers emitted by runCommand.
func (s *Session) parseMarkers(
	full, exitMarker, cwdMarker, doneMarker string,
	timedOut bool,
	maxOutput int,
	runErr error,
) (*types.ShellSessionResult, error) {
	result := &types.ShellSessionResult{TimedOut: timedOut}

	// Extract stdout: everything before the exit marker.
	exitIdx := strings.Index(full, exitMarker)
	truncated := false
	if exitIdx >= 0 {
		stdoutPart := full[:exitIdx]
		if len(stdoutPart) >= maxOutput {
			truncated = true
			stdoutPart = stdoutPart[:maxOutput]
		}
		result.Stdout = strings.TrimRight(stdoutPart, "\r\n")

		// Parse exit code: exitMarker + <code> + "__"
		afterExit := full[exitIdx+len(exitMarker):]
		if endIdx := strings.Index(afterExit, "__"); endIdx >= 0 {
			codeStr := strings.TrimSpace(strings.TrimRight(afterExit[:endIdx], "\r"))
			if code, err := strconv.Atoi(codeStr); err == nil {
				result.ExitCode = &code
			}
		}
	} else {
		// No exit marker — command may have crashed before emitting it.
		result.Stdout = strings.TrimRight(full, "\r\n")
		if runErr != nil {
			// Try to extract exit code from the run error.
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				result.ExitCode = &code
			}
		}
	}

	// Parse cwd: cwdMarker + <cwd> + "__"
	cwdIdx := strings.Index(full, cwdMarker)
	if cwdIdx >= 0 {
		afterCwd := full[cwdIdx+len(cwdMarker):]
		if endIdx := strings.Index(afterCwd, "__"); endIdx >= 0 {
			cwd := strings.TrimSpace(strings.TrimRight(afterCwd[:endIdx], "\r"))
			if cwd != "" {
				result.CwdAfter = cwd
			}
		}
	}
	if result.CwdAfter == "" {
		result.CwdAfter = s.cwd
	}

	result.Truncated = truncated
	return result, nil
}

// shellName returns the base name of the shell path (e.g. "bash", "cmd.exe").
func shellName(path string) string {
	name := path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.LastIndex(name, "\\"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

// quoteShell quotes a path for use as a cd argument. Uses double quotes
// on Windows, single quotes on Unix.
func quoteShell(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "/", "\\")
		return "\"" + path + "\""
	}
	return "'" + path + "'"
}
