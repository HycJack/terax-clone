//go:build windows

package pty

import (
	"fmt"
	"testing"
	"time"
)

func TestZZPipeStartup(t *testing.T) {
	for _, shell := range []string{
		"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
		"C:\\Windows\\system32\\cmd.exe",
	} {
		t.Logf("=== %s ===", shell)
		integ, err := integrationFor(shell)
		if err != nil {
			t.Fatalf("integrationFor: %v", err)
		}
		args := append([]string{}, integ.extraArgs...)
		master, _, _, err := startChild(shell, args, "C:\\Windows\\Temp", 120, 30)
		if err != nil {
			t.Fatalf("startChild: %v", err)
		}
		defer master.Close()
		done := make(chan []byte, 1)
		go func() {
			var out []byte
			buf := make([]byte, 64*1024)
			for {
				n, e := master.Read(buf)
				if n > 0 {
					out = append(out, buf[:n]...)
					if len(out) > 50000 {
						break
					}
				}
				if e != nil {
					break
				}
			}
			done <- out
		}()
		var collected []byte
		select {
		case collected = <-done:
		case <-time.After(3 * time.Second):
		}
		fmt.Printf("--- output (%d bytes), first 2000 visible ---\n%s\n--- END ---\n", len(collected), vis(collected))
	}
}

func vis(b []byte) string {
	s := ""
	for _, c := range b {
		switch {
		case c == '\x1b':
			s += "\\e"
		case c == '\r':
			s += "\\r"
		case c == '\n':
			s += "\\n"
		case c < 0x20 || c == 0x7f:
			s += fmt.Sprintf("[%02X]", c)
		default:
			s += string(c)
		}
	}
	return s
}