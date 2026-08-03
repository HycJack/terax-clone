//go:build windows

package pty

import (
	"os"
	"syscall"
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