//go:build !windows

package sysproc

import "os/exec"

func hideWindow(cmd *exec.Cmd) {
	// No-op on non-Windows hosts.
}
