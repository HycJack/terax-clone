package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusParsesPathsWithoutLeadingSpace(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")
	writeFile(t, repo, "tracked.txt", "v1")
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-qm", "init")
	writeFile(t, repo, "tracked.txt", "v2")
	writeFile(t, repo, " untracked.txt", "new")

	snap, err := Status(context.Background(), repo)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	got := map[string]bool{}
	for _, f := range snap.ChangedFiles {
		got[f.Path] = true
	}
	for _, want := range []string{"tracked.txt", " untracked.txt"} {
		if !got[want] {
			t.Errorf("missing changed file %q; got %v", want, got)
		}
	}
	// Regression: porcelain v1 prefixes every path with a separator space
	// (`XY <path>`), which used to leak into the parsed pathname and broke
	// `git diff -- <path>`. `tracked.txt` must arrive without the space.
	if !got["tracked.txt"] {
		t.Fatalf("tracked.txt missing from changed files (leading separator space leaked?): %v", got)
	}
	// A filename that genuinely starts with a space must keep it: only the
	// single separator space is stripped.
	if !got[" untracked.txt"] {
		t.Errorf("literal leading-space filename lost: %v", got)
	}
}

func TestStatusParsesRenames(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")
	writeFile(t, repo, "old.txt", "v1")
	runGit(t, repo, "add", "old.txt")
	runGit(t, repo, "commit", "-qm", "init")
	runGit(t, repo, "mv", "old.txt", "new.txt")

	snap, err := Status(context.Background(), repo)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	found := false
	for _, f := range snap.ChangedFiles {
		if f.Path == "new.txt" && f.OriginalPath != nil && *f.OriginalPath == "old.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected rename new.txt <- old.txt, got %+v", snap.ChangedFiles)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v (%s)", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}