package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a throwaway git repo in t.TempDir and returns its root.
func initTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "config", "user.email", "test@example.com")
	mustGit(t, root, "config", "user.name", "Test")
	return root
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiffContentUnstagedModified(t *testing.T) {
	root := initTestRepo(t)
	const rel = "a.txt"
	writeFile(t, root, rel, "line1\nline2\nline3\n")
	mustGit(t, root, "add", rel)
	mustGit(t, root, "commit", "-m", "add a.txt")

	// Modify the working tree (unstaged).
	writeFile(t, root, rel, "line1\nline2 changed\nline4 added\n")

	res, err := DiffContent(context.Background(), root, rel, false, "")
	if err != nil {
		t.Fatalf("DiffContent: %v", err)
	}
	if !strings.Contains(res.OriginalContent, "line3") {
		t.Errorf("original should contain the HEAD snapshot; got %q", res.OriginalContent)
	}
	if !strings.Contains(res.ModifiedContent, "line2 changed") {
		t.Errorf("modified should contain the working snapshot; got %q", res.ModifiedContent)
	}
	if res.IsBinary {
		t.Errorf("text file must not be flagged binary")
	}
}

func TestDiffContentNewUntracked(t *testing.T) {
	root := initTestRepo(t)
	const rel = "new.txt"
	writeFile(t, root, rel, "brand new\n")

	res, err := DiffContent(context.Background(), root, rel, false, "")
	if err != nil {
		t.Fatalf("DiffContent: %v", err)
	}
	if res.OriginalContent != "" {
		t.Errorf("untracked file original should be empty; got %q", res.OriginalContent)
	}
	if !strings.Contains(res.ModifiedContent, "brand new") {
		t.Errorf("untracked file modified should contain its content; got %q", res.ModifiedContent)
	}
}

func TestDiffContentStaged(t *testing.T) {
	root := initTestRepo(t)
	const rel = "a.txt"
	writeFile(t, root, rel, "v1\n")
	mustGit(t, root, "add", rel)
	mustGit(t, root, "commit", "-m", "add")

	writeFile(t, root, rel, "v2\n")
	mustGit(t, root, "add", rel)

	res, err := DiffContent(context.Background(), root, rel, true, "")
	if err != nil {
		t.Fatalf("DiffContent: %v", err)
	}
	if !strings.Contains(res.OriginalContent, "v1") {
		t.Errorf("staged original should be the HEAD snapshot; got %q", res.OriginalContent)
	}
	if !strings.Contains(res.ModifiedContent, "v2") {
		t.Errorf("staged modified should be the index snapshot; got %q", res.ModifiedContent)
	}
}

func TestCommitFileDiff(t *testing.T) {
	root := initTestRepo(t)
	const rel = "a.txt"
	writeFile(t, root, rel, "v1\n")
	mustGit(t, root, "add", rel)
	mustGit(t, root, "commit", "-m", "one")

	writeFile(t, root, rel, "v2\n")
	mustGit(t, root, "add", rel)
	mustGit(t, root, "commit", "-m", "two")

	sha := strings.TrimSpace(mustGit(t, root, "rev-parse", "HEAD"))
	res, err := CommitFileDiff(context.Background(), root, sha, rel, "")
	if err != nil {
		t.Fatalf("CommitFileDiff: %v", err)
	}
	if !strings.Contains(res.OriginalContent, "v1") {
		t.Errorf("commit original should be the parent snapshot; got %q", res.OriginalContent)
	}
	if !strings.Contains(res.ModifiedContent, "v2") {
		t.Errorf("commit modified should be the commit snapshot; got %q", res.ModifiedContent)
	}
}