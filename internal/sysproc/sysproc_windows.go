//go:build windows

package sysproc

import (
	"os/exec"
	"syscall"
)

// CREATE_NO_WINDOW is the Windows process creation flag that suppresses
// the console window. Constant value 0x08000000 (CREATE_NO_WINDOW).
// See https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
const CREATE_NO_WINDOW = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	// Preserve any existing flags the caller may have set.
	var flags uint32
	if cmd.SysProcAttr != nil {
		flags = cmd.SysProcAttr.CreationFlags
	}
	flags |= CREATE_NO_WINDOW
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags}
}
