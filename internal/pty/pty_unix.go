//go:build !windows

package pty

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func platformProcAttr() *syscall.SysProcAttr {
	return unixSysProcAttr()
}

// unixSysProcAttr returns the child process attributes for the go-pty path
// on Unix. We intentionally leave it empty: go-pty's Command start() sets
// Setsid + Setctty itself (creating a new session and process group so the
// child is isolated and killProcessTree via `kill(-pid)` works). Adding
// Setpgid on top of Setsid makes the child a process-group leader before
// setsid() runs, and macOS then fails the fork/exec with EPERM.
func unixSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func unixKillGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

func probeProcess(p *os.Process) bool {
	return p.Signal(syscall.Signal(0)) == nil
}

// Windows-only helpers stubbed out on non-Windows hosts.

func hasConPTY() bool { return false }

func newPipeBackend(shell string, args []string, cwd string) (*pipeBackend, error) {
	return nil, fmt.Errorf("pipe backend is windows-only")
}

func killProcessTreeWindows(p *os.Process) error {
	return fmt.Errorf("kill process tree is windows-only")
}

// pipeBackend is defined on Windows (fallback_windows.go); this stub keeps
// the shared pty.go compiling on non-Windows hosts where it is never used.
type pipeBackend struct {
	cmd *exec.Cmd
}

func (b *pipeBackend) Read(p []byte) (int, error)  { return 0, fmt.Errorf("not used") }
func (b *pipeBackend) Write(p []byte) (int, error) { return 0, fmt.Errorf("not used") }
func (b *pipeBackend) Close() error                { return nil }