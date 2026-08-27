// Package sysproc provides cross-platform helpers for configuring child
// process attributes. On Windows the key concern is hiding the console
// window that `os/exec` otherwise flashes when spawning `cmd.exe`,
// `reg.exe`, `wsl.exe`, `git.exe`, `rg.exe`, etc. On Unix the helper is
// a no-op (or sets the process group, depending on caller needs).
package sysproc

import "os/exec"

// HideWindow configures `cmd` so that spawning it on Windows does not
// flash a console window. On non-Windows hosts it is a no-op.
//
// Usage:
//
//	cmd := exec.Command("reg", "query", ...)
//	sysproc.HideWindow(cmd)
//	out, err := cmd.Output()
func HideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	hideWindow(cmd)
}
