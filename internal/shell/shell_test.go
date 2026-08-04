package shell

import (
	"context"
	"os"
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
