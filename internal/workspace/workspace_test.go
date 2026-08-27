package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

// reset clears the package-level registry so tests are independent.
func reset() {
	mu.Lock()
	defer mu.Unlock()
	allowed = map[string]bool{}
	cwd = ""
	launchDir = ""
}

func TestAuthorizeAndCheck(t *testing.T) {
	reset()
	root := t.TempDir()
	if err := Authorize(root); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	inside := filepath.Join(root, "src", "app.ts")
	if !IsAuthorized(inside) {
		t.Errorf("IsAuthorized(%q) = false, want true", inside)
	}
	outside := filepath.Join(os.TempDir(), "unrelated-"+t.Name(), "x.txt")
	if IsAuthorized(outside) {
		t.Errorf("IsAuthorized(%q) = true, want false", outside)
	}
}

func TestSymlinkEscapesAuthorization(t *testing.T) {
	reset()
	root := t.TempDir()
	if err := Authorize(root); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	// Secret outside the workspace.
	secretDir := t.TempDir()
	secretFile := filepath.Join(secretDir, "passwd")
	if err := os.WriteFile(secretFile, []byte("root:x:0:0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Symlink INSIDE the workspace pointing at the secret.
	link := filepath.Join(root, "innocent.txt")
	if err := os.Symlink(secretFile, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if IsAuthorized(link) {
		t.Errorf("IsAuthorized(%q) = true, want false: symlink escapes the workspace", link)
	}
}

func TestSymlinkAliasOfWorkspaceStillAuthorized(t *testing.T) {
	reset()
	root := t.TempDir()
	if err := Authorize(root); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	// Reading via the alias resolves to the real workspace root — must pass.
	if !IsAuthorized(filepath.Join(alias, "file.txt")) {
		t.Errorf("IsAuthorized(%q) = false, want true (alias of workspace)", filepath.Join(alias, "file.txt"))
	}
}

func TestSetCwdRequiresAuthorization(t *testing.T) {
	reset()
	root := t.TempDir()
	if err := Authorize(root); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if err := SetCwd(root); err != nil {
		t.Errorf("SetCwd(%q): %v", root, err)
	}
	if CurrentDir() != filepath.ToSlash(canonicalizePath(root)) {
		t.Errorf("CurrentDir() = %q, want %q", CurrentDir(), filepath.ToSlash(canonicalizePath(root)))
	}
	other := t.TempDir()
	if err := SetCwd(other); err == nil {
		t.Errorf("SetCwd(%q) succeeded, want error (not authorized)", other)
	}
}