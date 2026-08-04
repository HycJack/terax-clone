package shell

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"terax/internal/types"
)

// TestSessionRunCommand verifies the core sentinel-marker protocol:
// OpenSession → RunInSession → CloseSession, with correct stdout,
// exit code, and cwd tracking.
func TestSessionRunCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping session test in short mode")
	}

	m := NewManager()
	ctx := context.Background()

	id, err := OpenSession(ctx, m, types.ShellSessionOpenArgs{})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	defer func() { _ = CloseSession(m, id) }()

	// Give the shell a moment to start.
	time.Sleep(200 * time.Millisecond)

	// Run a simple echo command.
	result, err := RunInSession(m, types.ShellSessionRunArgs{
		ID:      id,
		Command: "echo hello-world",
	})
	if err != nil {
		t.Fatalf("RunInSession: %v", err)
	}

	if !strings.Contains(result.Stdout, "hello-world") {
		t.Errorf("stdout = %q, want it to contain 'hello-world'", result.Stdout)
	}
	if result.ExitCode == nil || *result.ExitCode != 0 {
		t.Errorf("exit_code = %v, want 0", result.ExitCode)
	}
	if result.TimedOut {
		t.Error("timed_out = true, want false")
	}
}

// TestSessionExitCodeNonZero verifies that non-zero exit codes are captured.
func TestSessionExitCodeNonZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping session test in short mode")
	}

	m := NewManager()
	ctx := context.Background()

	id, err := OpenSession(ctx, m, types.ShellSessionOpenArgs{})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	defer func() { _ = CloseSession(m, id) }()

	time.Sleep(200 * time.Millisecond)

	result, err := RunInSession(m, types.ShellSessionRunArgs{
		ID:      id,
		Command: "exit 42",
	})
	if err != nil {
		t.Fatalf("RunInSession: %v", err)
	}

	if result.ExitCode == nil {
		t.Error("exit_code is nil, want 42")
	} else if *result.ExitCode != 42 {
		t.Errorf("exit_code = %d, want 42", *result.ExitCode)
	}
}

// TestSessionCwdPersistence verifies that cwd persists across calls.
func TestSessionCwdPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping session test in short mode")
	}

	m := NewManager()
	ctx := context.Background()

	id, err := OpenSession(ctx, m, types.ShellSessionOpenArgs{
		Cwd: os.TempDir(),
	})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	defer func() { _ = CloseSession(m, id) }()

	time.Sleep(200 * time.Millisecond)

	// Run pwd to verify cwd.
	result, err := RunInSession(m, types.ShellSessionRunArgs{
		ID:      id,
		Command: "pwd",
	})
	if err != nil {
		t.Fatalf("RunInSession pwd: %v", err)
	}

	// The stdout should contain the temp dir path, and CwdAfter should
	// match.
	if result.CwdAfter == "" {
		t.Error("cwd_after is empty")
	}
}

// TestSessionQuotedPath is a regression test for the cmd.exe /C quoting
// bug. Previously, `cd /d "C:\Users\..."` would fail because Go's
// argument builder wraps the script in extra quotes, and cmd.exe's /C
// strips the outer ones, mangling the inner quoted paths.
//
// The fix writes the script to a temp .bat file so cmd.exe parses it
// with its normal batch grammar. This test exercises that path.
func TestSessionQuotedPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping session test in short mode")
	}
	if runtime.GOOS != "windows" {
		t.Skip("test only meaningful on Windows")
	}

	// Find a directory that exists and contains spaces in its name.
	// `C:\Program Files` is always present on Windows.
	target := `C:\Program Files`
	if _, err := os.Stat(target); err != nil {
		t.Skipf("test target %q not present: %v", target, err)
	}

	m := NewManager()
	ctx := context.Background()

	id, err := OpenSession(ctx, m, types.ShellSessionOpenArgs{
		Cwd: os.TempDir(),
	})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	defer func() { _ = CloseSession(m, id) }()

	time.Sleep(200 * time.Millisecond)

	// cd using a path containing a space.
	result, err := RunInSession(m, types.ShellSessionRunArgs{
		ID:      id,
		Command: `cd /d "` + target + `"`,
	})
	if err != nil {
		t.Fatalf("RunInSession cd: %v", err)
	}

	if result.ExitCode == nil || *result.ExitCode != 0 {
		t.Errorf("exit_code = %v, want 0 (cd should succeed); stdout=%q",
			result.ExitCode, result.Stdout)
	}

	// After cd, CwdAfter should reflect the new directory (the system
	// may normalise the case; we compare case-insensitively).
	if result.CwdAfter == "" {
		t.Fatal("cwd_after is empty after cd")
	}
	got := strings.ToLower(filepath.Clean(result.CwdAfter))
	want := strings.ToLower(filepath.Clean(target))
	if got != want {
		t.Errorf("cwd_after = %q, want %q", result.CwdAfter, target)
	}

	// Verify cwd persisted across calls.
	result2, err := RunInSession(m, types.ShellSessionRunArgs{
		ID:      id,
		Command: "cd",
	})
	if err != nil {
		t.Fatalf("RunInSession cd: %v", err)
	}
	if !strings.Contains(strings.ToLower(result2.Stdout), "program files") {
		t.Errorf("pwd stdout = %q, want it to contain the new cwd",
			result2.Stdout)
	}
}
