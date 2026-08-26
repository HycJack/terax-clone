// Package git wraps the system `git` CLI. We delegate to git instead of
// using go-git because the frontend already speaks porcelain output and the
// Rust backend used git directly.
//
// Each public function returns the wire shape the Rust Tauri backend
// produced (now mirrored as Go structs in `internal/types`). Field names
// follow the camelCase JSON shape the frontend expects.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"terax/internal/sysproc"
	"terax/internal/types"
)

// run executes `git` with the args and cwd, returning combined stdout.
func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	sysproc.HideWindow(cmd)
	cmd.Dir = dir
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}

// ResolveRepo walks up from `cwd` until it finds the repo root. Returns
// nil when the path isn't inside a repo (mirrors Rust's
// `Result<Option<GitRepoInfo>, String>`).
func ResolveRepo(ctx context.Context, cwd string) (*types.GitRepoInfo, error) {
	if cwd == "" {
		return nil, nil
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	sysproc.HideWindow(cmd)
	cmd.Dir = cwd
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// Not in a repo. The Rust backend returns None in this case.
		return nil, nil
	}
	root := strings.TrimSpace(strings.ReplaceAll(out.String(), "\\", "/"))
	if root == "" {
		return nil, nil
	}
	info := &types.GitRepoInfo{RepoRoot: root}
	// Get the current branch name + upstream + detached flag.
	headerOut, err := run(ctx, root, "status", "--porcelain=v1", "-z", "--branch")
	if err == nil {
		for _, p := range strings.Split(headerOut, "\x00") {
			if strings.HasPrefix(p, "## ") {
				info.Branch, info.Upstream, info.IsDetached = parseBranchHeader(p[3:])
				break
			}
		}
	}
	return info, nil
}

func parseBranchHeader(header string) (branch string, upstream *string, detached bool) {
	if idx := strings.Index(header, "..."); idx >= 0 {
		branch = header[:idx]
		rest := header[idx+3:]
		braceIdx := strings.Index(rest, " [")
		if braceIdx < 0 {
			upstream = &rest
		} else {
			u := rest[:braceIdx]
			upstream = &u
		}
		return branch, upstream, false
	}
	if strings.Contains(header, "(detached at ") {
		start := strings.Index(header, "(detached at ")
		branch = header[:start-1]
		return branch, nil, true
	}
	return header, nil, false
}

// PanelSnapshot returns the wrapped `{repo, status}` snapshot. Either field
// may be nil when cwd isn't inside a repo.
func PanelSnapshot(ctx context.Context, cwd string) (types.GitPanelSnapshot, error) {
	if cwd == "" {
		return types.GitPanelSnapshot{}, nil
	}
	repo, err := ResolveRepo(ctx, cwd)
	if err != nil || repo == nil {
		return types.GitPanelSnapshot{Repo: repo}, nil
	}
	status, err := Status(ctx, repo.RepoRoot)
	if err != nil {
		// Surface repo info even when status parsing fails.
		return types.GitPanelSnapshot{Repo: repo}, nil
	}
	return types.GitPanelSnapshot{Repo: repo, Status: &status}, nil
}

// Status parses `git status --porcelain=v1 -z` into the Rust envelope.
func Status(ctx context.Context, repo string) (types.GitStatusSnapshot, error) {
	if repo == "" {
		return types.GitStatusSnapshot{}, fmt.Errorf("no repo")
	}
	out, err := run(ctx, repo, "status", "--porcelain=v1", "-z", "--branch")
	if err != nil {
		return types.GitStatusSnapshot{}, err
	}

	snap := types.GitStatusSnapshot{
		RepoRoot:    repo,
		ChangedFiles: []types.GitChangedFile{},
	}
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if strings.HasPrefix(p, "## ") {
			branch, upstream, detached := parseBranchHeader(p[3:])
			snap.Branch = branch
			snap.Upstream = upstream
			snap.IsDetached = detached
			continue
		}
		if len(p) < 3 {
			continue
		}
		x := p[0]
		y := p[1]
		// porcelain v1 always puts a single space between the two status
		// columns and the pathname — even with `-z` (NUL-terminated) mode:
		// `XY <path>\0`. Strip it so downstream commands (`git diff -- path`,
		// `git add -- path`) receive the exact pathname; a filename that
		// starts with a space is preserved because only the one separator
		// space is removed.
		rest := p[2:]
		path := strings.TrimPrefix(rest, " ")
		var original *string
		if x == 'R' || y == 'R' || x == 'C' || y == 'C' {
			if i+1 < len(parts) {
				// -z rename/copy: `XY <new>\0<old>\0`.
				old := parts[i+1]
				original = &old
				i++
			} else if idx := strings.Index(rest, " -> "); idx >= 0 {
				// non -z fallback: `XY <new> -> <old>`.
				old := rest[:idx]
				new := rest[idx+4:]
				original = &old
				path = new
			}
		}
		indexStatus := string(x)
		if x == ' ' {
			indexStatus = ""
		}
		worktreeStatus := string(y)
		if y == ' ' {
			worktreeStatus = ""
		}
		status, label := porcelainCode(x, y)
		staged := x != ' ' && x != '?'
		unstaged := y != ' ' && y != '?'
		untracked := x == '?' && y == '?'
		cf := types.GitChangedFile{
			Path:           strings.ReplaceAll(path, "\\", "/"),
			OriginalPath:   original,
			IndexStatus:    indexStatus,
			WorktreeStatus: worktreeStatus,
			Staged:         staged,
			Unstaged:       unstaged,
			Untracked:      untracked,
			StatusLabel:    label,
		}
		_ = status
		snap.ChangedFiles = append(snap.ChangedFiles, cf)
		if staged {
			snap.Ahead++
		}
		if unstaged {
			snap.Behind++
		}
	}
	return snap, nil
}

func porcelainCode(x, y byte) (status, label string) {
	if x == '?' && y == '?' {
		return "untracked", "Untracked"
	}
	if x == '!' && y == '!' {
		return "ignored", "Ignored"
	}
	if x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D') {
		return "conflicted", "Conflicted"
	}
	switch x {
	case 'A':
		return "added", "Added"
	case 'M':
		return "modified", "Modified"
	case 'D':
		return "deleted", "Deleted"
	case 'R':
		return "renamed", "Renamed"
	case 'C':
		return "copied", "Copied"
	}
	switch y {
	case 'M':
		return "modified", "Modified"
	case 'D':
		return "deleted", "Deleted"
	}
	return "unknown", "Unknown"
}

// Diff returns a unified diff envelope for the given path.
func Diff(ctx context.Context, repo, path string, staged bool) (types.GitDiffResult, error) {
	args := []string{"diff", "--no-color", "--no-ext-diff"}
	if staged {
		args = append(args, "--staged")
	}
	args = append(args, "--", path)
	text, err := run(ctx, repo, args...)
	if err != nil {
		return types.GitDiffResult{}, err
	}
	return types.GitDiffResult{DiffText: text, Truncated: false}, nil
}

// DiffContent returns the staged/unstaged diff as a JSON-shaped envelope.
// The Rust backend uses `git diff --no-color` and parses hunks; we keep the
// same shape so the frontend's diff viewer continues to work.
func DiffContent(ctx context.Context, repo, path string, staged bool, originalPath string) (types.GitDiffContentResult, error) {
	args := []string{"diff", "--no-color", "--no-ext-diff"}
	if staged {
		args = append(args, "--staged")
	}
	args = append(args, "--", path)
	text, err := run(ctx, repo, args...)
	if err != nil {
		return types.GitDiffContentResult{}, err
	}
	orig, mod := parseUnifiedDiff(text)
	return types.GitDiffContentResult{
		OriginalContent: orig,
		ModifiedContent: mod,
		IsBinary:        false,
		FallbackPatch:   text,
		Truncated:       false,
	}, nil
}

// Stage adds paths to the index.
func Stage(ctx context.Context, repo string, paths []string) error {
	args := append([]string{"add", "--"}, paths...)
	_, err := run(ctx, repo, args...)
	return err
}

// Unstage resets the index entries.
func Unstage(ctx context.Context, repo string, paths []string) error {
	args := append([]string{"reset", "HEAD", "--"}, paths...)
	_, err := run(ctx, repo, args...)
	return err
}

// Discard reverts working tree changes. `untracked` entries are removed
// with `git clean -f`, the rest with `git checkout --`.
func Discard(ctx context.Context, repo string, entries []types.GitDiscardEntry) error {
	for _, e := range entries {
		if e.Untracked {
			if _, err := run(ctx, repo, "clean", "-f", "--", e.Path); err != nil {
				return err
			}
		} else {
			if _, err := run(ctx, repo, "checkout", "--", e.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

// Commit creates a commit with the given message.
func Commit(ctx context.Context, repo, message string) (types.GitCommitResult, error) {
	out, err := run(ctx, repo, "commit", "-m", message)
	if err != nil {
		return types.GitCommitResult{}, err
	}
	sha, summary := parseCommitOutput(out)
	return types.GitCommitResult{CommitSha: sha, Summary: summary}, nil
}

// parseCommitOutput extracts the new commit SHA from `git commit` output.
// Format example:
//
//	[main abc1234] my commit message
//	 1 file changed, 1 insertion(+), 0 deletions(-)
func parseCommitOutput(out string) (sha, summary string) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			close := strings.Index(line, "]")
			if close < 0 {
				continue
			}
			parts := strings.Fields(line[1:close])
			if len(parts) >= 2 {
				sha = parts[len(parts)-1]
				summary = strings.TrimSpace(line[close+1:])
			}
			return sha, summary
		}
	}
	return sha, summary
}

// Fetch fetches from origin.
func Fetch(ctx context.Context, repo string) error {
	_, err := run(ctx, repo, "fetch")
	return err
}

// PullFFOnly performs `git pull --ff-only`.
func PullFFOnly(ctx context.Context, repo string) error {
	_, err := run(ctx, repo, "pull", "--ff-only")
	return err
}

// Push pushes the current branch. Returns the Rust envelope
// `{remote, branch, pushed}`. We can't easily detect fast-forward vs
// non-fast-forward from `git push` exit code on its own, so we report
// `pushed: true` whenever git returns exit 0.
func Push(ctx context.Context, repo string) (types.GitPushResult, error) {
	out, err := run(ctx, repo, "push")
	if err != nil {
		return types.GitPushResult{}, err
	}
	res := types.GitPushResult{Pushed: true}
	// Try to glean remote + branch from output (e.g. "To github.com:...").
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "To ") {
			res.Remote = ptr(strings.TrimSpace(line[3:]))
		}
	}
	if res.Remote == nil {
		if u, err := RemoteURL(ctx, repo, ""); err == nil {
			res.Remote = u
		}
	}
	// Get current branch from status header.
	if statusOut, err := run(ctx, repo, "status", "--porcelain=v1", "-z", "--branch"); err == nil {
		for _, p := range strings.Split(statusOut, "\x00") {
			if strings.HasPrefix(p, "## ") {
				b, _, _ := parseBranchHeader(p[3:])
				res.Branch = &b
				break
			}
		}
	}
	return res, nil
}

// Log returns recent log entries. Matches the Rust `GitLogEntry` envelope:
// sha, shortSha, author, authorEmail, timestampSecs, parents, subject,
// filesChanged, insertions, deletions.
func Log(ctx context.Context, repo string, limit int, beforeSha *string) ([]types.GitLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	// Use `--numstat` to capture per-commit insertions/deletions/files-changed.
	format := "%H%x00%h%x00%an%x00%ae%x00%s%x00%at%x00%P"
	args := []string{"log", "--pretty=format:" + format, "--numstat", "-n", strconv.Itoa(limit)}
	if beforeSha != nil && *beforeSha != "" {
		args = append(args, *beforeSha+"^")
	}
	out, err := run(ctx, repo, args...)
	if err != nil {
		return nil, err
	}
	// `git log --pretty --numstat` separates commits with blank lines; for
	// each commit, the first line is the formatted header and the
	// following lines are `<added>\t<removed>\t<path>` until a blank line.
	var commits []types.GitLogEntry
	var current *types.GitLogEntry
	flush := func() {
		if current != nil {
			commits = append(commits, *current)
		}
		current = nil
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			flush()
			continue
		}
		if strings.Contains(line, "\x00") {
			parts := strings.Split(line, "\x00")
			if len(parts) < 7 {
				continue
			}
			var ts int64
			fmt.Sscanf(parts[5], "%d", &ts)
			parents := []string{}
			if parts[6] != "" {
				parents = strings.Fields(parts[6])
			}
			flush()
			current = &types.GitLogEntry{
				Sha:           parts[0],
				ShortSha:      parts[1],
				Author:        parts[2],
				AuthorEmail:   parts[3],
				Subject:       parts[4],
				TimestampSecs: ts,
				Parents:       parents,
			}
			continue
		}
		// numstat line: `<added>\t<removed>\t<path>` (binary: `-` `-` path)
		if current == nil {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		current.FilesChanged++
		add, _ := strconv.Atoi(fields[0])
		rem, _ := strconv.Atoi(fields[1])
		current.Insertions += add
		current.Deletions += rem
	}
	flush()
	return commits, nil
}

// ShowCommit returns the unified diff for a single commit.
func ShowCommit(ctx context.Context, repo, sha string) (types.GitDiffResult, error) {
	text, err := run(ctx, repo, "show", "--no-color", "--no-ext-diff", sha)
	if err != nil {
		return types.GitDiffResult{}, err
	}
	return types.GitDiffResult{DiffText: text, Truncated: false}, nil
}

// CommitFiles lists files touched by a commit, including add/remove
// counts and rename metadata.
func CommitFiles(ctx context.Context, repo, sha string) ([]types.GitCommitFileChange, error) {
	out, err := run(ctx, repo, "show", "--numstat", "--pretty=format:", "--no-renames", sha)
	if err != nil {
		return nil, err
	}
	var files []types.GitCommitFileChange
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		added, _ := strconv.Atoi(fields[0])
		removed, _ := strconv.Atoi(fields[1])
		path := fields[2]
		// numstat with `--no-renames` uses the new path; we don't get
		// the original path so leave it nil.
		files = append(files, types.GitCommitFileChange{
			Path:        path,
			Status:      "modified",
			StatusLabel: "Modified",
			Added:       added,
			Removed:     removed,
			IsBinary:    fields[0] == "-" && fields[1] == "-",
		})
	}
	return files, nil
}

// CommitFileDiff returns the diff for a single file in a commit.
func CommitFileDiff(ctx context.Context, repo, sha, path, originalPath string) (types.GitDiffContentResult, error) {
	if originalPath != "" && originalPath != path {
		text, err := run(ctx, repo, "show", "--no-color", "--no-ext-diff", sha, "--", originalPath, path)
		if err != nil {
			return types.GitDiffContentResult{}, err
		}
		orig, mod := parseUnifiedDiff(text)
		return types.GitDiffContentResult{
			OriginalContent: orig,
			ModifiedContent: mod,
			FallbackPatch:   text,
			Truncated:       false,
		}, nil
	}
	text, err := run(ctx, repo, "show", "--no-color", "--no-ext-diff", sha, "--", path)
	if err != nil {
		return types.GitDiffContentResult{}, err
	}
	orig, mod := parseUnifiedDiff(text)
	return types.GitDiffContentResult{
		OriginalContent: orig,
		ModifiedContent: mod,
		FallbackPatch:   text,
		Truncated:       false,
	}, nil
}

// RemoteURL returns the configured remote URL. `name` is "origin" by
// default. Returns nil when no remote is configured.
func RemoteURL(ctx context.Context, repo, name string) (*string, error) {
	if name == "" {
		name = "origin"
	}
	out, err := run(ctx, repo, "config", "--get", "remote."+name+".url")
	if err != nil {
		// No remote configured — the Rust backend returns None here.
		return nil, nil
	}
	url := strings.TrimSpace(out)
	if url == "" {
		return nil, nil
	}
	return &url, nil
}

// ListBranches returns the local + worktree branches envelope.
func ListBranches(ctx context.Context, repo string) (types.GitBranchListResult, error) {
	if repo == "" {
		return types.GitBranchListResult{Branches: []types.GitBranchEntry{}}, nil
	}
	out, err := run(ctx, repo, "for-each-ref",
		"--format=%(refname:short)%00%(upstream:short)%00%(HEAD)%00%(worktreepath)",
		"refs/heads/")
	if err != nil {
		return types.GitBranchListResult{}, err
	}
	var entries []types.GitBranchEntry
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\x00")
		if len(parts) < 4 || parts[0] == "" {
			continue
		}
		entry := types.GitBranchEntry{
			Name:       parts[0],
			Kind:       "local",
			IsHead:     parts[2] == "*",
			IsDetached: false,
		}
		if parts[3] != "" {
			wt := parts[3]
			entry.Kind = "worktree"
			entry.WorktreePath = &wt
		}
		entries = append(entries, entry)
	}
	return types.GitBranchListResult{Branches: entries}, nil
}

// CheckoutBranch switches to a local branch.
func CheckoutBranch(ctx context.Context, repo, name string) error {
	_, err := run(ctx, repo, "checkout", name)
	return err
}

// ptr returns a pointer to the literal string. Tiny helper for optional
// fields so the struct literal can stay readable.
func ptr(s string) *string {
	return &s
}

// parseUnifiedDiff splits a unified diff into original and modified content.
// Lines starting with '-' go to original, '+' to modified, and context lines
// (starting with ' ' or no prefix) go to both. The diff header and hunk
// headers are skipped.
func parseUnifiedDiff(text string) (original, modified string) {
	var origLines, modLines []string
	for _, line := range strings.Split(text, "\n") {
		// Skip diff headers and hunk headers.
		if strings.HasPrefix(line, "diff --git ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "@@ ") {
			continue
		}
		// "\ No newline at end of file" — skip.
		if line == `\ No newline at end of file` {
			continue
		}
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '-':
			// Removed line — goes to original only.
			origLines = append(origLines, line[1:])
		case '+':
			// Added line — goes to modified only.
			modLines = append(modLines, line[1:])
		default:
			// Context line (starts with ' ' or is empty after header skips).
			content := line
			if line[0] == ' ' {
				content = line[1:]
			}
			origLines = append(origLines, content)
			modLines = append(modLines, content)
		}
	}
	return strings.Join(origLines, "\n"), strings.Join(modLines, "\n")
}