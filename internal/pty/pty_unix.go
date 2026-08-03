//go:build !windows

package pty

import (
	"os"
	"syscall"
)

func unixSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func unixKillGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

func probeProcess(p *os.Process) bool {
	return p.Signal(syscall.Signal(0)) == nil
}