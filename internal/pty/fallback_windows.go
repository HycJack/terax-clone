//go:build windows

// Fallback pty backend for Windows builds that pre-date Windows 10 1809
// (build 17763) and therefore lack `CreatePseudoConsole` in kernel32.
//
// We probe `kernel32!CreatePseudoConsole` once at process start. If it's
// missing, `hasConPTY()` flips to false and the manager routes every
// `Open()` call through `pipeBackend`, which is a plain `exec.Cmd` with
// stdio piped to anonymous pipes — no real PTY (no ANSI, no resize, no
// Ctrl+C escape sequences), but the shell still produces output the
// frontend can read.
package pty

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
)

// Lazy handles for the ConPTY procedures. Resolved at package init so the
// first Open() call doesn't pay the dll lookup cost.
var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procCreatePseudo      = kernel32.NewProc("CreatePseudoConsole")
	conPTYProbed   atomicBool
	hasConPTYValue atomicBool
)

type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (a *atomicBool) Store(v bool) { a.mu.Lock(); a.v = v; a.mu.Unlock() }
func (a *atomicBool) Load() bool    { a.mu.Lock(); defer a.mu.Unlock(); return a.v }

func init() {
	// NewProc() never errors on missing symbols; the error surfaces at the
	// first Call(). We ask the loader to find the entry point — `nil`
	// means the export exists in this build of kernel32.dll.
	conPTYProbed.Store(true)
	hasConPTYValue.Store(procCreatePseudo.Find() == nil)
}

// hasConPTY reports whether the running Windows kernel exports
// CreatePseudoConsole (i.e. is Windows 10 1809 / Windows 11 / Server 2019+).
func hasConPTY() bool { return hasConPTYValue.Load() }

// pipeBackend is the no-PTY fallback: just a child process with stdio
// connected to three anonymous pipes. The "master" is the joined
// reader+writer pair. Resize is a no-op.
type pipeBackend struct {
	stdin  *os.File
	stdout *os.File
	cmd    *exec.Cmd
	mu     sync.Mutex
}

func newPipeBackend(shell string, args []string, cwd string) (*pipeBackend, error) {
	cmd := exec.Command(shell, args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	// CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW — group isolates Ctrl+C,
	// NO_WINDOW suppresses the console flash.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x08000000}

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		_ = stdinR.Close()
		_ = stdinW.Close()
		return nil, err
	}
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stdoutW
	if err := cmd.Start(); err != nil {
		_ = stdinR.Close()
		_ = stdinW.Close()
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return nil, fmt.Errorf("pipe backend start: %w", err)
	}
	// We don't need the child-side ends of the pipes anymore.
	_ = stdinR.Close()
	_ = stdoutW.Close()

	return &pipeBackend{stdin: stdinW, stdout: stdoutR, cmd: cmd}, nil
}

func (b *pipeBackend) Read(p []byte) (int, error)  { return b.stdout.Read(p) }
func (b *pipeBackend) Write(p []byte) (int, error) { return b.stdin.Write(p) }
func (b *pipeBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var errs []error
	if err := b.stdin.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := b.stdout.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}