package pty

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestZZHiddenConsole(t *testing.T) {
	logf, _ := os.Create(os.TempDir() + "/zz_hidden_console.txt")
	defer logf.Close()
	ph := func(label string, out []byte) {
		s := "\n=== " + label + " len=" + fmt.Sprint(len(out)) + " ===\n"
		for _, c := range out {
			switch c {
			case 27:
				s += `\e`
			case '\r':
				s += `\r`
			case '\n':
				s += `\n`
			default:
				if c < 0x20 || c == 0x7f {
					s += fmt.Sprintf("[%02X]", c)
				} else {
					s += string(c)
				}
			}
		}
		logf.WriteString(s)
		logf.Sync()
	}

	const CREATE_NEW_CONSOLE = 0x00000010
	shell := "C:\\Windows\\system32\\cmd.exe"
	cmd := exec.Command(shell)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: CREATE_NEW_CONSOLE,
		HideWindow:    true,
	}
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	var out []byte
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		rr := io.MultiReader(stdout, stderr)
		for {
			n, e := rr.Read(buf)
			if n > 0 {
				out = append(out, buf[:n]...)
			}
			if e != nil {
				break
			}
		}
		close(done)
	}()

	time.Sleep(600 * time.Millisecond)
	ph("phase0-banner", out)
	io.WriteString(stdin, "echo hello-123\r")
	time.Sleep(350 * time.Millisecond)
	ph("phase1-after-echo", out)
	io.WriteString(stdin, "dir\r")
	time.Sleep(350 * time.Millisecond)
	ph("phase2-after-dir", out)
	io.WriteString(stdin, string([]byte{3}))
	time.Sleep(300 * time.Millisecond)
	ph("phase3-after-ctrlc", out)
	io.WriteString(stdin, "exit\r")
	time.Sleep(500 * time.Millisecond)
	ph("phase4-final", out)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		logf.WriteString("\n[goroutine still reading — pipe never closed]\n")
	}
	logf.Sync()
}