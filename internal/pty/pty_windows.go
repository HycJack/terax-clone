//go:build windows

package pty

import (
	"os"
	"syscall"
	"time"
)

var (
	// procGenerateCtrlC is resolved lazily from kernel32.dll (declared in
	// fallback_windows.go which also owns the `kernel32` lazy DLL handle).
	procGenerateCtrlC = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

func unixSysProcAttr() *syscall.SysProcAttr {
	// On Windows we don't need Setpgid; the procAttr already sets
	// CREATE_NEW_PROCESS_GROUP. This stub exists so the file compiles.
	return &syscall.SysProcAttr{}
}

func unixKillGroup(pid int) error {
	return nil
}

func probeProcess(p *os.Process) bool {
	// `os.FindProcess` always succeeds on Windows even when the process is
	// already gone, so signal-0 probing doesn't work. Use the running flag
	// indirectly by attempting to query the process state.
	return p != nil && p.Pid > 0
}

// sendCtrlC sends CTRL_C_EVENT to the process group identified by pid.
// The child was spawned with CREATE_NEW_PROCESS_GROUP, so its group id
// equals its pid. Returns true if the signal was sent successfully.
func sendCtrlC(pid int) bool {
	r, _, err := procGenerateCtrlC.Call(
		uintptr(syscall.CTRL_C_EVENT),
		uintptr(pid),
	)
	return r != 0 && err == nil
}

// killProcessTreeWindows attempts a graceful Ctrl+C first, then falls
// back to TerminateProcess after a short grace period.
func killProcessTreeWindows(p *os.Process) error {
	// Try Ctrl+C first — lets the child clean up (flush buffers,
	// restore terminal state, run trap handlers, etc.).
	if sendCtrlC(p.Pid) {
		// Give the process up to 500 ms to exit on its own.
		done := make(chan struct{})
		go func() {
			_, _ = p.Wait()
			close(done)
		}()
		select {
		case <-done:
			return nil // clean exit
		case <-time.After(500 * time.Millisecond):
			// Still alive — fall through to hard kill.
		}
	}
	// Hard kill: TerminateProcess.
	return p.Kill()
}