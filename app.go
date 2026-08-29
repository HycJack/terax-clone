// Package main wires up the Wails app and binds every Tauri command the
// frontend invokes into a single Go method (or one of the internal/* helpers).
//
// The generated TypeScript bindings live under `frontend/wailsjs/go/main/App`.
// The frontend shim (`frontend/src/lib/wails-shim/core.ts`) routes by name.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"terax/internal/agent"
	"terax/internal/events"
	"terax/internal/fs"
	"terax/internal/git"
	"terax/internal/lsp"
	"terax/internal/mcp"
	internalpty "terax/internal/pty"
	internalshell "terax/internal/shell"
	internaltype "terax/internal/types"
	"terax/internal/winctrl"
	"terax/internal/workspace"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the Wails-bound struct.
type App struct {
	ctx       context.Context
	app       *application.App
	wailsApp  *application.App
	mainWindow application.Window
	ptyMgr    *internalpty.Manager
	fsWatcher *fs.WatcherManager
	lspMgr    *lsp.Manager
	bgMgr     *internalshell.Manager
}

// PtyManager is an alias so we can share with shell pkg.
type PtyManager = internalpty.Manager

// NewApp constructs an App with default state.
func NewApp() *App {
	return &App{
		ptyMgr:    internalpty.NewManager(),
		fsWatcher: fs.NewWatcherManager(),
		lspMgr:    lsp.NewManager(),
		bgMgr:     internalshell.NewManager(),
	}
}

// ServiceStartup is called by Wails v3 during application startup.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	a.app = application.Get()
	a.fsWatcher.BindContext(ctx)

	// Initialize per-user data dirs.
	dataDir := appDir()
	if err := os.MkdirAll(dataDir, 0o755); err == nil {
		storeInit(dataDir)
		historyInit(filepath.Join(dataDir, "history.txt"))
	}

	// Set default launch directory to the user's home directory.
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		workspace.InitLaunchCwd(homeDir)
	} else if wd, err := os.Getwd(); err == nil {
		workspace.InitLaunchCwd(wd)
	}

	// Wire the custom window-chrome close/minimise buttons.
	winctrl.Register(ctx)

	// Event-bridge for settings page: store/secrets ops via EventsEmit/EventsOn.
	events.RegisterAll(ctx)

	return nil
}

// shutdown is called when the app is exiting.
func (a *App) shutdown() {
	if a.ptyMgr != nil {
		a.ptyMgr.CloseAll()
	}
	if a.lspMgr != nil {
		a.lspMgr.KillAll()
	}
	if a.fsWatcher != nil {
		a.fsWatcher.Close()
	}
}

// =========================================================================
// Launch dir / files (matches Tauri's `LaunchDir` state)
// =========================================================================

// GetLaunchDir returns the directory the user passed at launch.
func (a *App) GetLaunchDir() *string {
	d := workspace.CurrentDir()
	if d == "" {
		return nil
	}
	return &d
}

// GetLaunchFiles returns the files passed via "Open With" / CLI args.
func (a *App) GetLaunchFiles() []string {
	// Not yet implemented: would require macOS-style file association
	// handling, which Windows / Linux deliver through argv at cold start.
	return []string{}
}

// =========================================================================
// PTY commands
// =========================================================================

// PtyOpenArgs is the request body for PtyOpen.
type PtyOpenArgs = internaltype.PtyOpenArgs

// PtyOpen starts a new PTY session.
func (a *App) PtyOpen(args PtyOpenArgs) (int, error) {
	ws := workspace.ResolveWorkspaceEnv(args.Workspace)
	cwd := args.Cwd
	if cwd == nil || *cwd == "" {
		v := workspace.CurrentDir()
		cwd = &v
	}
	if cwd == nil || *cwd == "" {
		// No cwd available from workspace or current dir; surface a clear
		// error rather than silently falling back to a hard-coded path.
		return 0, fmt.Errorf("pty open: no working directory available")
	}
	shell := ""
	if args.Shell != nil {
		shell = *args.Shell
	}
	s, err := a.ptyMgr.Open(a.ctx, args.Cols, args.Rows, *cwd, ws.Kind, shell)
	if err != nil {
		return 0, err
	}
	return s.ID, nil
}

// PtyWriteArgs is the request body for PtyWrite.
type PtyWriteArgs struct {
	ID   int    `json:"id"`
	Data []byte `json:"data"`
}

// PtyWrite sends stdin data to a PTY session.
func (a *App) PtyWrite(args PtyWriteArgs) error {
	return a.ptyMgr.Write(args.ID, args.Data)
}

// PtyResizeArgs is the request body for PtyResize.
type PtyResizeArgs struct {
	ID   int `json:"id"`
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// PtyResize changes the PTY geometry.
func (a *App) PtyResize(args PtyResizeArgs) error {
	return a.ptyMgr.Resize(args.ID, args.Cols, args.Rows)
}

// PtyStartArgs is the request body for PtyStart.
type PtyStartArgs struct {
	ID int `json:"id"`
}

// PtyStart begins forwarding PTY output. The frontend must subscribe to the
// `pty:<id>` event BEFORE calling this, so shell startup bytes don't race
// ahead of the unbuffered Wails event bus and get dropped.
func (a *App) PtyStart(args PtyStartArgs) error {
	return a.ptyMgr.Start(a.ctx, args.ID)
}

// PtyClose terminates a PTY session.
func (a *App) PtyClose(id int) error {
	a.ptyMgr.Close(id)
	return nil
}

// PtyCloseAll kills every PTY session.
func (a *App) PtyCloseAll() {
	a.ptyMgr.CloseAll()
}

// PtyHasForegroundProcess mirrors the original Rust command.
func (a *App) PtyHasForegroundProcess(id int) bool {
	return a.ptyMgr.HasForeground(id)
}

// PtyHasForegroundJob is an alias used by the AI workflow.
func (a *App) PtyHasForegroundJob(id int) bool {
	return a.ptyMgr.HasForeground(id)
}

// PtyShellName returns the executable name (pwsh, zsh, ...) for a session.
func (a *App) PtyShellName(id int) string {
	return a.ptyMgr.ShellName(id)
}

// PtyListShells returns the shells available on this machine.
func (a *App) PtyListShells() []string {
	return a.ptyMgr.ListShells()
}

// PtyReadOutput used to be a poll fallback for the Wails event bus. We
// removed the outputBuf and the polling path; the new pump emits
// `pty:<id>` events directly. The method is kept as a no-op stub so a
// stale frontend build (calling `pty_read_output`) doesn't get a 404 —
// it just returns nil bytes.
func (a *App) PtyReadOutput(id int) []byte {
	return nil
}

// =========================================================================
// FS commands
// =========================================================================

// FsReadDirArgs mirrors the request shape.
type FsReadDirArgs = internaltype.FsReadDirArgs

// FsReadDir lists directory entries.
func (a *App) FsReadDir(args FsReadDirArgs) ([]internaltype.DirEntry, error) {
	return fs.ReadDir(args)
}

// ListSubdirsArgs is the request body.
type ListSubdirsArgs struct {
	Path       string                `json:"path"`
	ShowHidden bool                  `json:"showHidden"`
	Workspace  internaltype.WorkspaceEnv `json:"workspace"`
}

// ListSubdirs returns the names of immediate subdirectories of `path`.
// Used by the status-bar breadcrumb to populate the dropdown. Hidden
// entries are filtered by default; pass `showHidden: true` to include
// dot-prefixed names.
func (a *App) ListSubdirs(args ListSubdirsArgs) ([]string, error) {
	entries, err := fs.ReadDir(internaltype.FsReadDirArgs{
		Path:       args.Path,
		ShowHidden: args.ShowHidden,
		Workspace:  args.Workspace,
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Kind == "dir" {
			out = append(out, e.Name)
		}
	}
	return out, nil
}

// FsReadFile reads a file as UTF-8 text.
func (a *App) FsReadFile(path string) (internaltype.ReadResult, error) {
	if !workspace.IsAuthorized(path) {
		return internaltype.ReadResult{}, fmt.Errorf("path not authorized: %s", path)
	}
	return fs.ReadFile(path)
}

// FsWriteFileArgs is the request body.
type FsWriteFileArgs = internaltype.FsWriteArgs

// FsWriteFile writes UTF-8 content.
func (a *App) FsWriteFile(args FsWriteFileArgs) error {
	if !workspace.IsAuthorized(args.Path) {
		return fmt.Errorf("path not authorized: %s", args.Path)
	}
	return fs.WriteFile(args)
}

// FsStat returns metadata for a path.
func (a *App) FsStat(path string) (internaltype.FsStat, error) {
	return fs.Stat(path)
}

// FsCanonicalize returns an absolute path with forward slashes.
func (a *App) FsCanonicalize(path string) string {
	return fs.Canonicalize(path)
}

// FsCreateFileArgs is the request body.
type FsCreateFileArgs = internaltype.FsCreateArgs

// FsCreateFile creates an empty file.
func (a *App) FsCreateFile(args FsCreateFileArgs) error {
	if !workspace.IsAuthorized(args.Path) {
		return fmt.Errorf("path not authorized: %s", args.Path)
	}
	return fs.CreateFile(args)
}

// FsCreateDir creates a directory recursively.
func (a *App) FsCreateDir(args FsCreateFileArgs) error {
	if !workspace.IsAuthorized(args.Path) {
		return fmt.Errorf("path not authorized: %s", args.Path)
	}
	return fs.CreateDir(args)
}

// FsRenameArgs is the request body.
type FsRenameArgs = internaltype.FsRenameArgs

// FsRename moves a path.
func (a *App) FsRename(args FsRenameArgs) error {
	if !workspace.IsAuthorized(args.From) {
		return fmt.Errorf("from path not authorized: %s", args.From)
	}
	return fs.Rename(args)
}

// FsDeleteArgs is the request body.
type FsDeleteArgs = internaltype.FsDeleteArgs

// FsDelete removes a file or directory.
func (a *App) FsDelete(args FsDeleteArgs) error {
	if !workspace.IsAuthorized(args.Path) {
		return fmt.Errorf("path not authorized: %s", args.Path)
	}
	return fs.Delete(args)
}

// FsCopyArgs is the request body.
type FsCopyArgs = internaltype.FsCopyArgs

// FsCopy copies one or more paths into a destination directory.
func (a *App) FsCopy(args FsCopyArgs) error {
	// Authorise both sides: each source must be allowed, and the
	// destination must be inside an authorised tree.
	for _, source := range args.Sources {
		if !workspace.IsAuthorized(source) {
			return fmt.Errorf("source path not authorized: %s", source)
		}
	}
	if !workspace.IsAuthorized(args.DestDir) {
		return fmt.Errorf("destination path not authorized: %s", args.DestDir)
	}
	return fs.Copy(args)
}

// FsWatchAddArgs is the request body.
type FsWatchAddArgs = internaltype.FsWatchArgs

// FsWatchAdd adds directories to the watcher.
func (a *App) FsWatchAdd(args FsWatchAddArgs) {
	a.fsWatcher.Add(args.Paths)
}

// FsWatchRemoveArgs is the request body.
type FsWatchRemoveArgs = internaltype.FsWatchArgs

// FsWatchRemove removes directories from the watcher.
func (a *App) FsWatchRemove(args FsWatchRemoveArgs) {
	a.fsWatcher.Remove(args.Paths)
}

// =========================================================================
// LSP commands
// =========================================================================

// LspDetect reports whether a binary exists.
func (a *App) LspDetect(command string) bool {
	return lsp.Detect(command)
}

// LspHostPID returns the host's process ID.
func (a *App) LspHostPID() int {
	return lsp.HostPID()
}

// LspResolveRootArgs is the request body — both path and markers must be
// passed together because Wails can only marshal a single struct, not
// multiple positional args.
type LspResolveRootArgs struct {
	Path    string   `json:"path"`
	Markers []string `json:"markers"`
}

// LspResolveRoot walks up from `path` looking for any marker.
func (a *App) LspResolveRoot(args LspResolveRootArgs) string {
	return lsp.ResolveRoot(args.Path, args.Markers)
}

// LspSpawnArgs mirrors the request shape.
type LspSpawnArgs = internaltype.LspSpawnArgs

// LspSpawn starts a language server.
func (a *App) LspSpawn(args LspSpawnArgs) (int, error) {
	s, err := lsp.Spawn(a.ctx, a.lspMgr, args)
	if err != nil {
		return 0, err
	}
	return s.ID, nil
}

// LspSendArgs is the request body.
type LspSendArgs struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
}

// LspSend writes a JSON-RPC message to the LSP server.
func (a *App) LspSend(args LspSendArgs) error {
	return a.lspMgr.Send(args.ID, args.Message)
}

// LspKill terminates a session.
func (a *App) LspKill(id int) error {
	return a.lspMgr.Kill(id)
}

// =========================================================================
// FS search commands
// =========================================================================

// FsSearchArgs is the request body.
type FsSearchArgs = internaltype.FsSearchArgs

// FsSearch finds files by name substring. Returns the Rust envelope
// ({hits: [{path, rel, name, isDir}], truncated}) so the explorer
// component renders consistently with the Tauri build.
func (a *App) FsSearch(args FsSearchArgs) (internaltype.FsSearchResponse, error) {
	if args.Root != "" && !workspace.IsAuthorized(args.Root) {
		return internaltype.FsSearchResponse{}, fmt.Errorf("path not authorized: %s", args.Root)
	}
	return fs.Search(args)
}

// FsListFilesArgs is the request body.
type FsListFilesArgs = internaltype.FsListFilesArgs

// FsListFiles walks a directory collecting file paths up to a cap. Mirrors
// the Rust backend's `ListFilesResult` envelope ({files, truncated}).
func (a *App) FsListFiles(args FsListFilesArgs) (internaltype.FsListFilesResult, error) {
	if !workspace.IsAuthorized(args.Root) {
		return internaltype.FsListFilesResult{}, fmt.Errorf("path not authorized: %s", args.Root)
	}
	return fs.ListFiles(args)
}

// FsGrepArgs is the request body.
type FsGrepArgs = internaltype.FsGrepArgs

// FsGrep runs a content search and returns the envelope the frontend
// expects: {hits, truncated, filesScanned}. The Go side prefers ripgrep
// when installed; falls back to an in-process scanner otherwise.
func (a *App) FsGrep(args FsGrepArgs) (internaltype.FsGrepResponse, error) {
	if args.Root != "" && !workspace.IsAuthorized(args.Root) {
		return internaltype.FsGrepResponse{}, fmt.Errorf("path not authorized: %s", args.Root)
	}
	return fs.Grep(args)
}

// FsGrepInteractiveArgs mirrors the streaming variant.
type FsGrepInteractiveArgs = internaltype.FsGrepArgs

// FsGrepInteractive is currently the same as FsGrep; the Rust version used
// a long-running channel but for simplicity we return the full result set.
func (a *App) FsGrepInteractive(args FsGrepInteractiveArgs) (internaltype.FsGrepResponse, error) {
	if args.Root != "" && !workspace.IsAuthorized(args.Root) {
		return internaltype.FsGrepResponse{}, fmt.Errorf("path not authorized: %s", args.Root)
	}
	return fs.Grep(args)
}

// FsGlobArgs is the request body.
type FsGlobArgs = internaltype.FsGlobArgs

// FsGlob matches a glob pattern and returns the envelope the frontend
// expects: {hits: [{path, rel}], truncated}.
func (a *App) FsGlob(args FsGlobArgs) (internaltype.FsGlobResponse, error) {
	if args.Root != "" && !workspace.IsAuthorized(args.Root) {
		return internaltype.FsGlobResponse{}, fmt.Errorf("path not authorized: %s", args.Root)
	}
	return fs.Glob(args)
}

// =========================================================================
// Git commands
// =========================================================================

// GitResolveRepoArgs is the request body for git_resolve_repo.
type GitResolveRepoArgs struct {
	Cwd string `json:"cwd"`
}

// GitResolveRepo finds the repo root for a cwd. Returns the Rust
// envelope `{repoRoot, branch, upstream, isDetached}` (nullable when
// not inside a repo).
func (a *App) GitResolveRepo(args GitResolveRepoArgs) (*internaltype.GitRepoInfo, error) {
	return git.ResolveRepo(a.ctx, args.Cwd)
}

// GitPanelSnapshotArgs is the request body.
type GitPanelSnapshotArgs struct {
	Cwd string `json:"cwd"`
}

// GitPanelSnapshot returns the parsed panel snapshot (nested repo +
// status envelope).
func (a *App) GitPanelSnapshot(args GitPanelSnapshotArgs) (internaltype.GitPanelSnapshot, error) {
	return git.PanelSnapshot(a.ctx, args.Cwd)
}

// GitStatusArgs is the request body.
type GitStatusArgs struct {
	RepoRoot string `json:"repoRoot"`
}

// GitStatus returns the parsed status snapshot.
func (a *App) GitStatus(args GitStatusArgs) (internaltype.GitStatusSnapshot, error) {
	return git.Status(a.ctx, args.RepoRoot)
}

// GitDiffArgs is the request body.
type GitDiffArgs struct {
	RepoRoot string `json:"repoRoot"`
	Path     string `json:"path"`
	Staged   bool   `json:"staged"`
}

// GitDiff returns the unified diff for a path.
func (a *App) GitDiff(args GitDiffArgs) (internaltype.GitDiffResult, error) {
	return git.Diff(a.ctx, args.RepoRoot, args.Path, args.Staged)
}

// GitDiffContentArgs is the request body.
type GitDiffContentArgs struct {
	RepoRoot     string `json:"repoRoot"`
	Path         string `json:"path"`
	Staged       bool   `json:"staged"`
	OriginalPath string `json:"originalPath,omitempty"`
}

// GitDiffContent returns the side-by-side diff body.
func (a *App) GitDiffContent(args GitDiffContentArgs) (internaltype.GitDiffContentResult, error) {
	return git.DiffContent(a.ctx, args.RepoRoot, args.Path, args.Staged, args.OriginalPath)
}

// GitStageArgs is the request body.
type GitStageArgs struct {
	RepoRoot string   `json:"repoRoot"`
	Paths    []string `json:"paths"`
}

// GitStage adds paths to the index.
func (a *App) GitStage(args GitStageArgs) error {
	return git.Stage(a.ctx, args.RepoRoot, args.Paths)
}

// GitUnstageArgs is the request body.
type GitUnstageArgs = GitStageArgs

// GitUnstage resets the index entries.
func (a *App) GitUnstage(args GitUnstageArgs) error {
	return git.Unstage(a.ctx, args.RepoRoot, args.Paths)
}

// GitDiscardArgs is the request body (entries: [{path, untracked}]).
type GitDiscardArgs struct {
	RepoRoot string                       `json:"repoRoot"`
	Entries  []internaltype.GitDiscardEntry `json:"entries"`
}

// GitDiscard reverts working-tree changes.
func (a *App) GitDiscard(args GitDiscardArgs) error {
	return git.Discard(a.ctx, args.RepoRoot, args.Entries)
}

// GitCommitArgs is the request body.
type GitCommitArgs struct {
	RepoRoot string `json:"repoRoot"`
	Message  string `json:"message"`
}

// GitCommit creates a commit. Returns {commitSha, summary} envelope.
func (a *App) GitCommit(args GitCommitArgs) (internaltype.GitCommitResult, error) {
	return git.Commit(a.ctx, args.RepoRoot, args.Message)
}

// GitFetchArgs is the request body.
type GitFetchArgs struct {
	RepoRoot string `json:"repoRoot"`
}

// GitFetch fetches from origin.
func (a *App) GitFetch(args GitFetchArgs) error {
	return git.Fetch(a.ctx, args.RepoRoot)
}

// GitPullFFOnlyArgs is the request body.
type GitPullFFOnlyArgs = GitFetchArgs

// GitPullFFOnly performs a fast-forward pull.
func (a *App) GitPullFFOnly(args GitPullFFOnlyArgs) error {
	return git.PullFFOnly(a.ctx, args.RepoRoot)
}

// GitPushArgs is the request body.
type GitPushArgs = GitFetchArgs

// GitPush pushes the current branch. Returns {remote, branch, pushed}.
func (a *App) GitPush(args GitPushArgs) (internaltype.GitPushResult, error) {
	return git.Push(a.ctx, args.RepoRoot)
}

// GitLogArgs is the request body.
type GitLogArgs struct {
	RepoRoot  string  `json:"repoRoot"`
	Limit     int     `json:"limit,omitempty"`
	BeforeSha *string `json:"beforeSha,omitempty"`
}

// GitLog returns recent commits with full per-commit stats.
func (a *App) GitLog(args GitLogArgs) ([]internaltype.GitLogEntry, error) {
	return git.Log(a.ctx, args.RepoRoot, args.Limit, args.BeforeSha)
}

// GitShowCommitArgs is the request body.
type GitShowCommitArgs struct {
	RepoRoot string `json:"repoRoot"`
	Sha      string `json:"sha"`
}

// GitShowCommit returns the unified diff for a commit.
func (a *App) GitShowCommit(args GitShowCommitArgs) (internaltype.GitDiffResult, error) {
	return git.ShowCommit(a.ctx, args.RepoRoot, args.Sha)
}

// GitCommitFilesArgs is the request body.
type GitCommitFilesArgs = GitShowCommitArgs

// GitCommitFiles returns files touched by a commit.
func (a *App) GitCommitFiles(args GitCommitFilesArgs) ([]internaltype.GitCommitFileChange, error) {
	return git.CommitFiles(a.ctx, args.RepoRoot, args.Sha)
}

// GitCommitFileDiffArgs is the request body.
type GitCommitFileDiffArgs struct {
	RepoRoot     string `json:"repoRoot"`
	Sha          string `json:"sha"`
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath,omitempty"`
}

// GitCommitFileDiff returns the diff for one file in a commit.
func (a *App) GitCommitFileDiff(args GitCommitFileDiffArgs) (internaltype.GitDiffContentResult, error) {
	return git.CommitFileDiff(a.ctx, args.RepoRoot, args.Sha, args.Path, args.OriginalPath)
}

// GitRemoteURLArgs is the request body.
type GitRemoteURLArgs struct {
	RepoRoot string `json:"repoRoot"`
	Name     string `json:"name,omitempty"`
}

// GitRemoteURL returns the remote URL (nullable when no remote).
func (a *App) GitRemoteURL(args GitRemoteURLArgs) (*string, error) {
	return git.RemoteURL(a.ctx, args.RepoRoot, args.Name)
}

// GitListBranchesArgs is the request body.
type GitListBranchesArgs = GitFetchArgs

// GitListBranches returns the local+worktree branches envelope.
func (a *App) GitListBranches(args GitListBranchesArgs) (internaltype.GitBranchListResult, error) {
	return git.ListBranches(a.ctx, args.RepoRoot)
}

// GitCheckoutBranchArgs is the request body.
type GitCheckoutBranchArgs struct {
	RepoRoot string `json:"repoRoot"`
	Branch   string `json:"branch"`
}

// GitCheckoutBranch switches branch.
func (a *App) GitCheckoutBranch(args GitCheckoutBranchArgs) error {
	return git.CheckoutBranch(a.ctx, args.RepoRoot, args.Branch)
}

// =========================================================================
// Shell commands
// =========================================================================

// ShellRunCommandArgs is the request body.
type ShellRunCommandArgs = internaltype.ShellRunArgs

// ShellRunCommand runs a one-shot shell command.
func (a *App) ShellRunCommand(args ShellRunCommandArgs) (string, error) {
	return internalshell.RunCommand(a.ctx, args)
}

// ShellSessionOpenArgs is the request body.
type ShellSessionOpenArgs = internaltype.ShellSessionOpenArgs

// ShellSessionOpen starts an interactive session.
func (a *App) ShellSessionOpen(args ShellSessionOpenArgs) (int, error) {
	return internalshell.OpenSession(a.ctx, a.bgMgr, args)
}

// ShellSessionRunArgs is the request body.
type ShellSessionRunArgs = internaltype.ShellSessionRunArgs

// ShellSessionRun sends a command to an existing session and returns the
// captured output, exit code, and post-command cwd.
func (a *App) ShellSessionRun(args ShellSessionRunArgs) (*internaltype.ShellSessionResult, error) {
	return internalshell.RunInSession(a.bgMgr, args)
}

// ShellSessionCloseArgs is the request body.
type ShellSessionCloseArgs struct {
	ID int `json:"id"`
}

// ShellSessionClose closes an interactive session.
func (a *App) ShellSessionClose(args ShellSessionCloseArgs) error {
	return internalshell.CloseSession(a.bgMgr, args.ID)
}

// ShellBgSpawnArgs is the request body.
type ShellBgSpawnArgs = internaltype.ShellBgSpawnArgs

// ShellBgSpawn starts a background process.
func (a *App) ShellBgSpawn(args ShellBgSpawnArgs) (int, error) {
	return internalshell.BgSpawn(a.ctx, a.bgMgr, args)
}

// ShellBgLogsArgs is the request body.
type ShellBgLogsArgs = internaltype.ShellBgLogsArgs

// ShellBgLogs returns captured output.
func (a *App) ShellBgLogs(args ShellBgLogsArgs) (string, error) {
	return internalshell.BgLogs(a.bgMgr, args)
}

// ShellBgKillArgs is the request body.
type ShellBgKillArgs = internaltype.ShellBgKillArgs

// ShellBgKill terminates a job.
func (a *App) ShellBgKill(args ShellBgKillArgs) error {
	return internalshell.BgKill(a.bgMgr, args)
}

// ShellBgList returns the list of job IDs.
func (a *App) ShellBgList() []internalshell.BgProcessInfo {
	return internalshell.BgList(a.bgMgr)
}

// =========================================================================
// Workspace / WSL commands
// =========================================================================

// WslListDistros returns the available WSL distros (Windows only). The
// Rust envelope is `[{name, default}]` so the frontend can highlight the
// active one.
func (a *App) WslListDistros() ([]workspace.WSLDistro, error) {
	return workspace.WSLListDistros()
}

// WslDefaultDistro returns the default WSL distro.
func (a *App) WslDefaultDistro() (string, error) {
	return workspace.WSLDefaultDistro()
}

// WslHomeArgs is the request body.
type WslHomeArgs struct {
	Distro string `json:"distro"`
}

// WslHome returns the home directory of a WSL distro.
func (a *App) WslHome(args WslHomeArgs) (string, error) {
	return workspace.WSLHome(args.Distro)
}

// WorkspaceAuthorizeArgs is the request body. `path` (not `dir`) so the
// frontend can pass a single named arg that matches every other workspace
// command's shape.
type WorkspaceAuthorizeArgs struct {
	Path string `json:"path"`
}

// WorkspaceAuthorize marks a directory as allowed.
func (a *App) WorkspaceAuthorize(args WorkspaceAuthorizeArgs) error {
	return workspace.Authorize(args.Path)
}

// WorkspaceCurrentDir returns the active cwd.
func (a *App) WorkspaceCurrentDir() string {
	return workspace.CurrentDir()
}

// WorkspaceSetCwdArgs is the request body for WorkspaceSetCwd.
type WorkspaceSetCwdArgs struct {
	Path string `json:"path"`
}

// WorkspaceSetCwd updates the global active cwd (default dir for the dir
// picker and the cwd fallback for new PTYs). Called after switching a
// workspace/space so CurrentDir follows the new home instead of staying on
// the first-launch directory.
func (a *App) WorkspaceSetCwd(args WorkspaceSetCwdArgs) error {
	return workspace.SetCwd(args.Path)
}

// WorkspacePickDirectory opens a native directory picker and returns the
// chosen path (empty string if the user cancelled). The picked directory
// is also authorised so the file explorer can browse it without a prompt.
func (a *App) WorkspacePickDirectory() (string, error) {
	if a.app == nil {
		return "", errors.New("app not initialised")
	}
	// Seed the picker with the workspace cwd or the home directory so the
	// user opens somewhere useful rather than at Documents/%USERPROFILE%.
	defaultDir := workspace.CurrentDir()
	if defaultDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			defaultDir = h
		}
	}
	path, err := a.app.Dialog.OpenFile().
		SetTitle("Select project directory").
		SetDirectory(defaultDir).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if path != "" {
		if err := workspace.Authorize(path); err != nil {
			return "", err
		}
		// Switch the global cwd to the picked directory so subsequent
		// relative path resolutions land in the new project root.
		_ = workspace.SetCwd(path)
		return path, nil
	}
	return "", nil
}

// =========================================================================
// Agent commands
// =========================================================================

// AgentEnableHooksArgs is the request body.
type AgentEnableHooksArgs struct {
	Agent string `json:"agent"`
}

// AgentEnableHooks announces hooks readiness for a given agent.
func (a *App) AgentEnableHooks(args AgentEnableHooksArgs) error {
	return agent.EnableHooks(a.ctx, args.Agent)
}

// AgentHooksStatusArgs is the request body.
type AgentHooksStatusArgs struct {
	Agent string `json:"agent"`
}

// AgentHooksStatus reports whether hooks are wired up. The frontend
// expects a flat boolean (`boolean`).
func (a *App) AgentHooksStatus(args AgentHooksStatusArgs) bool {
	return agent.HooksStatus(args.Agent).Ready
}

// =========================================================================
// Secrets (keyring)
// =========================================================================

// SecretsGetArgs is the request body.
type SecretsGetArgs struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

// SecretsGet reads a keyring entry.
func (a *App) SecretsGet(args SecretsGetArgs) (*string, error) {
	v, err := secretsGet(args.Service, args.Account)
	if err != nil {
		return nil, err
	}
	if v == "" {
		return nil, nil
	}
	return &v, nil
}

// SecretsSetArgs is the request body.
type SecretsSetArgs struct {
	Service  string `json:"service"`
	Account  string `json:"account"`
	Password string `json:"password"`
}

// SecretsSet writes a keyring entry.
func (a *App) SecretsSet(args SecretsSetArgs) error {
	return secretsSet(args.Service, args.Account, args.Password)
}

// SecretsDeleteArgs is the request body.
type SecretsDeleteArgs struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

// SecretsDelete removes a keyring entry.
func (a *App) SecretsDelete(args SecretsDeleteArgs) error {
	return secretsDelete(args.Service, args.Account)
}

// SecretsGetAllArgs is the request body.
type SecretsGetAllArgs struct {
	Service  string   `json:"service"`
	Accounts []string `json:"accounts"`
}

// SecretsGetAll returns the values for many accounts.
func (a *App) SecretsGetAll(args SecretsGetAllArgs) ([]*string, error) {
	values, err := secretsGetAll(args.Service, args.Accounts)
	if err != nil {
		return nil, err
	}
	out := make([]*string, len(values))
	for i, v := range values {
		if v == "" {
			out[i] = nil
		} else {
			out[i] = &v
		}
	}
	return out, nil
}

// =========================================================================
// Net / AI HTTP
// =========================================================================

// LmPingArgs is the request body.
type LmPingArgs = internaltype.LmPingArgs

// LmPing pings a local model endpoint.
func (a *App) LmPing(args LmPingArgs) (int, error) {
	return netLmPing(a.ctx, args.BaseURL)
}

// AiHttpRequestArgs is the request body.
type AiHttpRequestArgs = internaltype.AiHttpRequestArgs

// AiHttpRequest performs a single HTTP call.
func (a *App) AiHttpRequest(args AiHttpRequestArgs) (map[string]interface{}, error) {
	return netAiHTTPRequest(a.ctx, args)
}

// AiHttpStreamArgs is the request body.
type AiHttpStreamArgs = internaltype.AiHttpStreamArgs

// AiHttpStream opens a streaming HTTP call. Output flows over the
// `OnEventEvent` event.
func (a *App) AiHttpStream(args AiHttpStreamArgs) error {
	return netAiHTTPStream(a.ctx, args)
}

// =========================================================================
// History commands
// =========================================================================

// HistorySuggestArgs is the request body.
type HistorySuggestArgs = internaltype.HistorySuggestArgs

// HistorySuggest returns fuzzy matches.
func (a *App) HistorySuggest(args HistorySuggestArgs) ([]string, error) {
	return historySuggest(args.Line, args.Limit)
}

// HistoryCommandsArgs is the request body.
type HistoryCommandsArgs struct {
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
}

// HistoryCommands returns recorded commands matching the prefix (or every
// command if no prefix is supplied), capped at `limit` entries.
func (a *App) HistoryCommands(args HistoryCommandsArgs) ([]string, error) {
	return historyCommands(args.Prefix, args.Limit)
}

// HistoryRecordArgs is the request body.
type HistoryRecordArgs = internaltype.HistoryRecordArgs

// HistoryRecord appends a command.
func (a *App) HistoryRecord(args HistoryRecordArgs) error {
	return historyRecord(args.Command)
}

// HistoryListArgs is the request body.
type HistoryListArgs = internaltype.HistoryListArgs

// HistoryList returns the last N entries.
func (a *App) HistoryList(args HistoryListArgs) ([]string, error) {
	return historyList(args.Limit)
}

// =========================================================================
// Process / window-state / updater / autostart / opener
// =========================================================================

// ProcessRelaunch re-launches the binary with the same args.
func (a *App) ProcessRelaunch() error {
	return processRelaunch()
}

// ProcessExitArgs is the request body.
type ProcessExitArgs struct {
	Code int `json:"code"`
}

// ProcessExit terminates the app.
func (a *App) ProcessExit(args ProcessExitArgs) {
	if a.app != nil {
		a.app.Quit()
	}
}

// AutostartEnable enables OS-level autostart.
func (a *App) AutostartEnable() error { return autostartEnable() }

// AutostartDisable disables it.
func (a *App) AutostartDisable() error { return autostartDisable() }

// AutostartIsEnabled reports current state.
func (a *App) AutostartIsEnabled() bool { return autostartIsEnabled() }

// UpdaterCheck returns the latest release info (or null when up to date).
func (a *App) UpdaterCheck() map[string]interface{} { return updaterCheck() }

// OpenerOpenURLArgs is the request body.
type OpenerOpenURLArgs struct {
	URL string `json:"url"`
}

// OpenerOpenURL opens a URL in the default browser.
func (a *App) OpenerOpenURL(args OpenerOpenURLArgs) error {
	return openerOpenURL(args.URL)
}

// OpenerOpenPathArgs is the request body.
type OpenerOpenPathArgs struct {
	Path string `json:"path"`
}

// OpenerOpenPath opens a path with the OS default app.
func (a *App) OpenerOpenPath(args OpenerOpenPathArgs) error {
	return openerOpenPath(args.Path)
}

// OpenerRevealItemArgs is the request body.
type OpenerRevealItemArgs = OpenerOpenPathArgs

// OpenerRevealItem opens the file explorer at `path`.
func (a *App) OpenerRevealItem(args OpenerRevealItemArgs) error {
	return openerRevealItem(args.Path)
}

// =========================================================================
// Store (LazyStore persistence)
// =========================================================================

// StoreLoadArgs is the request body.
type StoreLoadArgs = internaltype.StoreLoadArgs

// StoreLoad reads a JSON bag.
func (a *App) StoreLoad(args StoreLoadArgs) (map[string]interface{}, error) {
	return storeLoad(args.Path)
}

// StoreSaveArgs is the request body.
type StoreSaveArgs = internaltype.StoreSaveArgs

// StoreSave writes a JSON bag.
func (a *App) StoreSave(args StoreSaveArgs) error {
	return storeSave(args.Path, args.Data)
}

// =========================================================================
// Misc — Settings window opener
// =========================================================================

// OpenSettingsWindowArgs is the request body.
type OpenSettingsWindowArgs struct {
	Tab *string `json:"tab"`
}

// OpenSettingsWindow opens the settings page in a new window.
// If the settings window already exists, it is focused instead.
func (a *App) OpenSettingsWindow(args OpenSettingsWindowArgs) error {
	tab := ""
	if args.Tab != nil {
		tab = *args.Tab
	}

	// If settings window already exists, focus it and switch tab via event
	if a.app != nil {
		if w, ok := a.app.Window.GetByName("settings"); ok {
			w.Focus()
			a.app.Event.Emit("terax:settings-tab", tab)
			return nil
		}
	}

	// Create a new settings window
	if a.app != nil {
		url := "settings.html"
		if tab != "" {
			url = "settings.html?tab=" + tab
		}
		a.app.Window.NewWithOptions(application.WebviewWindowOptions{
			Name:  "settings",
			Title: "Terax Settings",
			URL:   url,
			Width: 800,
			Height: 600,
			Mac: application.MacWindow{
				TitleBar: application.MacTitleBar{
					AppearsTransparent: true,
					FullSizeContent:    true,
					HideTitle:          true,
				},
			},
			BackgroundColour: application.NewRGBA(27, 38, 54, 1),
		})
	}
	return nil
}

// =========================================================================
// AppDir helpers
// =========================================================================

func appDir() string {
	if runtime.GOOS == "windows" {
		if a := os.Getenv("APPDATA"); a != "" {
			return filepath.Join(a, "terax")
		}
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".config", "terax")
	}
	return ".terax"
}

// AppConfigDir returns the per-user config directory (same as appDir).
func (a *App) AppConfigDir() string {
	return appDir()
}

// AppDataDir returns the per-user data directory.
func (a *App) AppDataDir() string {
	return appDir()
}

// AppHomeDir returns the user's home directory.
func (a *App) AppHomeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

// ── Multi-window helpers ───────────────────────────────────────────────

// WindowNew creates a new application window.
func (a *App) WindowNew(title string, width, height int) {
	a.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  title,
		Width:  width,
		Height: height,
		BackgroundColour: application.NewRGBA(27, 38, 54, 1),
	})
}

// WindowGetAll returns the titles of all open windows.
func (a *App) WindowGetAll() []string {
	windows := a.wailsApp.Window.GetAll()
	names := make([]string, 0, len(windows))
	for _, w := range windows {
		names = append(names, w.Name())
	}
	return names
}

// WindowClose closes the window with the given name.
func (a *App) WindowClose(name string) {
	if w, ok := a.wailsApp.Window.GetByName(name); ok {
		w.Close()
	}
}

// WindowFocus brings the named window to focus.
func (a *App) WindowFocus(name string) {
	if w, ok := a.wailsApp.Window.GetByName(name); ok {
		w.Focus()
	}
}

// ── Application info ───────────────────────────────────────────────────

// AppVersion returns the application version.
func (a *App) AppVersion() string {
	return "0.8.6"
}

// AppName returns the application name.
func (a *App) AppName() string {
	return "terax"
}

// ── MCP Server ─────────────────────────────────────────────────────────

// McpListTools returns the list of available MCP tools.
func (a *App) McpListTools() []map[string]any {
	srv := mcp.NewServer()
	tools := make([]map[string]any, 0, len(srv.Tools()))
	for _, t := range srv.Tools() {
		tools = append(tools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return tools
}

// McpCallTool calls an MCP tool by name with the given arguments.
func (a *App) McpCallTool(name string, args map[string]any) (any, error) {
	srv := mcp.NewServer()
	b, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	return srv.CallTool(a.ctx, name, json.RawMessage(b))
}