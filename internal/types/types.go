// Package types holds the shared data structures used by the bound methods
// in this app. They mirror the structs the Rust backend exported so the
// generated TS bindings stay compatible with the existing frontend.
package types

// DirEntry mirrors `DirEntry` in `src/modules/explorer/lib/useFileTree.ts`.
type DirEntry struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"` // "file" | "dir" | "symlink"
	Size        int64  `json:"size"`
	Mtime       int64  `json:"mtime"`
	GitIgnored  bool   `json:"gitignored"`
}

// FsStat mirrors `fs_stat`. `Kind` is "file" | "dir" | "symlink"
// matching the Rust `StatKind` enum (serde rename_all = "lowercase").
// The frontend reads `kind` directly to pick an icon.
type FsStat struct {
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime"`
	Kind  string `json:"kind"`
}

// WorkspaceEnv represents the current shell environment (local or WSL distro).
type WorkspaceEnv struct {
	Kind    string `json:"kind"` // "local" | "wsl"
	Distro  string `json:"distro,omitempty"`
	Cwd     string `json:"cwd"`
}

// PtyOpenArgs is the request body for the `pty_open` command. The two
// trailing fields carry Channel attachment IDs produced by the frontend's
// SPECIAL handler in `wails-shim/core.ts` (which renames `onData` -> `onDataEvent`
// and `onExit` -> `onExitEvent` before crossing the bridge).
type PtyOpenArgs struct {
	Cols      int          `json:"cols"`
	Rows      int          `json:"rows"`
	Cwd       *string      `json:"cwd"`
	Workspace WorkspaceEnv `json:"workspace"`
	Blocks    bool         `json:"blocks"`
	Shell     *string      `json:"shell"`
	OnDataEvent string     `json:"onDataEvent"`
	OnExitEvent string     `json:"onExitEvent"`
}

// LspSpawnArgs is the request body for `lsp_spawn`. The two trailing fields
// carry Channel attachment IDs produced by the frontend's SPECIAL handler
// in `wails-shim/core.ts` (which renames `onMessage` -> `onMessageEvent` and
// `onExit` -> `onExitEvent` before crossing the bridge).
type LspSpawnArgs struct {
	Command          string            `json:"command"`
	Args             []string          `json:"args"`
	Env              map[string]string `json:"env"`
	Root             string            `json:"root"`
	MaxRssMb         *int              `json:"maxRssMb"`
	Workspace        WorkspaceEnv      `json:"workspace"`
	OnMessageEvent   string            `json:"onMessageEvent"`
	OnExitEvent      string            `json:"onExitEvent"`
}

// LspExitInfo mirrors `LspExitInfo` in `src/modules/lsp/lib/transport.ts`.
type LspExitInfo struct {
	Code       *int   `json:"code"`
	StderrTail string `json:"stderrTail"`
	Reason     *string `json:"reason"`
}

// AiStreamEvent mirrors `AiStreamEvent` in `src/modules/ai/lib/proxyFetch.ts`.
type AiStreamEvent struct {
	Kind    string         `json:"kind"` // "headers" | "chunk" | "end" | "error"
	Status  int            `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Bytes   []byte         `json:"bytes,omitempty"`
	Message string         `json:"message,omitempty"`
}

// AiHttpStreamArgs is the request body for `ai_http_stream`.
type AiHttpStreamArgs struct {
	URL                string            `json:"url"`
	Method             string            `json:"method"`
	Headers            map[string]string `json:"headers"`
	Body               []byte            `json:"body"`
	AllowPrivateNetwork bool             `json:"allowPrivateNetwork"`
	OnEventEvent       string            `json:"onEventEvent"`
}

// AgentSignalKind mirrors the variants of `AgentSignal` in
// `src/modules/agents/lib/types.ts`.
type AgentSignalKind string

const (
	AgentStarted   AgentSignalKind = "started"
	AgentWorking   AgentSignalKind = "working"
	AgentAttention AgentSignalKind = "attention"
	AgentFinished  AgentSignalKind = "finished"
	AgentExited    AgentSignalKind = "exited"
)

// AgentSignal mirrors `AgentSignal`.
type AgentSignal struct {
	Kind  AgentSignalKind `json:"kind"`
	ID    string          `json:"id"` // pty id
	Agent *string         `json:"agent,omitempty"`
	Event *string         `json:"event,omitempty"`
}

// AgentHooksReady is emitted when backend hooks are wired up.
type AgentHooksReady struct {
	Agent string `json:"agent"`
}

// AgentHooksStatus mirrors the response of `agent_hooks_status`.
type AgentHooksStatus struct {
	Ready bool `json:"ready"`
}

// SecretGetArgs / SecretSetArgs / SecretDeleteArgs / SecretGetAllArgs.
type SecretGetArgs struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

type SecretSetArgs struct {
	Service  string `json:"service"`
	Account  string `json:"account"`
	Password string `json:"password"`
}

type SecretDeleteArgs struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

type SecretGetAllArgs struct {
	Service  string   `json:"service"`
	Accounts []string `json:"accounts"`
}

// ShellRunArgs runs a one-shot shell command.
type ShellRunArgs struct {
	Command   string `json:"command"`
	Cwd       string `json:"cwd"`
	TimeoutSecs int  `json:"timeoutSecs"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// ShellSessionOpenArgs starts an interactive shell session.
type ShellSessionOpenArgs struct {
	Cwd       string       `json:"cwd"`
	Workspace WorkspaceEnv `json:"workspace"`
	Shell     string       `json:"shell,omitempty"`
}

// ShellBgSpawnArgs starts a background process.
type ShellBgSpawnArgs struct {
	Command   string `json:"command"`
	Cwd       string `json:"cwd"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// ShellBgLogsArgs fetches captured logs.
type ShellBgLogsArgs struct {
	Handle      int `json:"handle"`
	SinceOffset int `json:"sinceOffset"`
}

// ShellBgKillArgs kills a running background job.
type ShellBgKillArgs struct {
	Handle int `json:"handle"`
}

// LmPingArgs pings a local model endpoint.
type LmPingArgs struct {
	BaseURL string `json:"baseUrl"`
}

// AiHttpRequestArgs is a non-streaming HTTP call.
type AiHttpRequestArgs struct {
	URL                string            `json:"url"`
	Method             string            `json:"method"`
	Headers            map[string]string `json:"headers"`
	Body               []byte            `json:"body"`
	AllowPrivateNetwork bool             `json:"allowPrivateNetwork"`
}

// GitRepoInfo mirrors the Rust backend's `GitRepoInfo`. Returned by
// `git_resolve_repo` (nullable when cwd isn't inside a repo).
type GitRepoInfo struct {
	RepoRoot   string  `json:"repoRoot"`
	Branch     string  `json:"branch"`
	Upstream   *string `json:"upstream"`
	IsDetached bool    `json:"isDetached"`
}

// GitChangedFile mirrors the Rust backend's `GitChangedFile` — one row of
// the status view.
type GitChangedFile struct {
	Path          string  `json:"path"`
	OriginalPath  *string `json:"originalPath"`
	IndexStatus   string  `json:"indexStatus"`
	WorktreeStatus string `json:"worktreeStatus"`
	Staged        bool    `json:"staged"`
	Unstaged      bool    `json:"unstaged"`
	Untracked     bool    `json:"untracked"`
	StatusLabel   string  `json:"statusLabel"`
}

// GitStatusSnapshot mirrors the Rust backend's `GitStatusSnapshot`. Returned
// by `git_status`.
type GitStatusSnapshot struct {
	RepoRoot    string           `json:"repoRoot"`
	Branch      string           `json:"branch"`
	Upstream    *string          `json:"upstream"`
	Ahead       int              `json:"ahead"`
	Behind      int              `json:"behind"`
	IsDetached  bool             `json:"isDetached"`
	Truncated   bool             `json:"truncated"`
	ChangedFiles []GitChangedFile `json:"changedFiles"`
}

// GitPanelSnapshot mirrors the Rust backend's `GitPanelSnapshot`.
// `Repo` and `Status` may be nil when cwd isn't in a repo.
type GitPanelSnapshot struct {
	Repo   *GitRepoInfo       `json:"repo"`
	Status *GitStatusSnapshot `json:"status"`
}

// GitDiffResult is the unified diff output for `git_diff` / `git_show_commit`.
type GitDiffResult struct {
	DiffText  string `json:"diffText"`
	Truncated bool   `json:"truncated"`
}

// GitDiffContentResult is the side-by-side blob content diff for
// `git_diff_content` / `git_commit_file_diff`.
type GitDiffContentResult struct {
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
	IsBinary       bool   `json:"isBinary"`
	FallbackPatch   string `json:"fallbackPatch"`
	Truncated       bool   `json:"truncated"`
}

// GitLogEntry mirrors a single row of the Rust backend's `git_log`.
type GitLogEntry struct {
	Sha           string   `json:"sha"`
	ShortSha      string   `json:"shortSha"`
	Author        string   `json:"author"`
	AuthorEmail   string   `json:"authorEmail"`
	TimestampSecs int64    `json:"timestampSecs"`
	Parents       []string `json:"parents"`
	Subject       string   `json:"subject"`
	FilesChanged  int      `json:"filesChanged"`
	Insertions    int      `json:"insertions"`
	Deletions     int      `json:"deletions"`
}

// GitCommitResult mirrors `git_commit` return shape.
type GitCommitResult struct {
	CommitSha string `json:"commitSha"`
	Summary   string `json:"summary"`
}

// GitCommitFileChange mirrors one row of the Rust backend's
// `git_commit_files` list.
type GitCommitFileChange struct {
	Path         string  `json:"path"`
	OriginalPath *string `json:"originalPath"`
	Status       string  `json:"status"`
	StatusLabel  string  `json:"statusLabel"`
	Added        int     `json:"added"`
	Removed      int     `json:"removed"`
	IsBinary     bool    `json:"isBinary"`
}

// GitDiscardEntry is one entry in `git_discard`'s request payload.
type GitDiscardEntry struct {
	Path      string `json:"path"`
	Untracked bool   `json:"untracked"`
}

// GitBranchEntry mirrors one row of the Rust backend's branch list.
type GitBranchEntry struct {
	Name         string  `json:"name"`
	Kind         string  `json:"kind"` // "local" | "worktree"
	WorktreePath *string `json:"worktreePath"`
	IsHead       bool    `json:"isHead"`
	IsDetached   bool    `json:"isDetached"`
}

// GitBranchListResult mirrors the envelope for `git_list_branches`.
type GitBranchListResult struct {
	Branches []GitBranchEntry `json:"branches"`
}

// GitPushResult mirrors `git_push` return shape.
type GitPushResult struct {
	Remote *string `json:"remote"`
	Branch *string `json:"branch"`
	Pushed bool    `json:"pushed"`
}

// HistoryRecordArgs records a command in history.
type HistoryRecordArgs struct {
	Command string `json:"command"`
}

// HistoryListArgs lists recorded commands.
type HistoryListArgs struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit"`
}

// HistorySuggestArgs provides fuzzy history suggestions.
type HistorySuggestArgs struct {
	Line  string `json:"line"`
	Limit int    `json:"limit"`
}

// StoreLoadArgs loads a JSON KV bag.
type StoreLoadArgs struct {
	Path string `json:"path"`
}

// StoreSaveArgs persists a JSON KV bag.
type StoreSaveArgs struct {
	Path string                 `json:"path"`
	Data map[string]interface{} `json:"data"`
}

// FsReadDirArgs lists entries of a directory.
type FsReadDirArgs struct {
	Path            string `json:"path"`
	ShowHidden      bool   `json:"showHidden"`
	GitDecorations  bool   `json:"gitDecorations"`
	Workspace       WorkspaceEnv `json:"workspace"`
}

// FsSearchArgs searches for files by glob/name.
type FsSearchArgs struct {
	Query     string `json:"query"`
	Root      string `json:"root"`
	MaxResults int   `json:"limit"`
	ShowHidden bool   `json:"showHidden"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsListFilesArgs returns files matching a glob.
type FsListFilesArgs struct {
	Pattern   string `json:"pattern"`
	Root      string `json:"root"`
	Limit     int    `json:"limit,omitempty"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsGrepArgs runs a content search (ripgrep equivalent).
type FsGrepArgs struct {
	Root          string   `json:"root"`
	Pattern       string   `json:"pattern"`
	Glob          []string `json:"glob,omitempty"`
	CaseInsensitive bool   `json:"caseInsensitive"`
	MaxResults    int      `json:"maxResults"`
	Workspace     WorkspaceEnv `json:"workspace"`
}

// FsGrepHit mirrors a single grep hit. Field names mirror the Rust
// backend's serialised struct (path, rel, line, text).
type FsGrepHit struct {
	Path string `json:"path"`
	Rel  string `json:"rel"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// FsGrepResponse mirrors the Rust `GrepResponse` envelope.
type FsGrepResponse struct {
	Hits         []FsGrepHit `json:"hits"`
	Truncated    bool        `json:"truncated"`
	FilesScanned int         `json:"files_scanned"`
}

// FsGlobHit mirrors a single glob match.
type FsGlobHit struct {
	Path string `json:"path"`
	Rel  string `json:"rel"`
}

// FsGlobResponse mirrors the Rust `GlobResponse` envelope.
type FsGlobResponse struct {
	Hits      []FsGlobHit `json:"hits"`
	Truncated bool        `json:"truncated"`
}

// FsListFilesResult mirrors the Rust `ListFilesResult` envelope.
type FsListFilesResult struct {
	Files     []string `json:"files"`
	Truncated bool     `json:"truncated"`
}

// FsSearchHit is one entry from the explorer search bar.
type FsSearchHit struct {
	Path  string `json:"path"`
	Rel   string `json:"rel"`
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

// FsSearchResponse mirrors the Rust explorer search envelope.
type FsSearchResponse struct {
	Hits      []FsSearchHit `json:"hits"`
	Truncated bool          `json:"truncated"`
}

// FsGlobArgs matches glob patterns.
type FsGlobArgs struct {
	Pattern    string `json:"pattern"`
	Root       string `json:"root"`
	MaxResults int    `json:"maxResults,omitempty"`
	Workspace  WorkspaceEnv `json:"workspace"`
}

// FsRenameArgs renames/moves a path.
type FsRenameArgs struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsCopyArgs copies one or more files/dirs into a destination directory.
// Mirrors the Rust backend's `fs_copy(sources, dest_dir)` signature so
// the frontend drag-drop payload ({sources, destDir}) flows through
// unchanged.
type FsCopyArgs struct {
	Sources   []string     `json:"sources"`
	DestDir   string       `json:"destDir"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsDeleteArgs deletes a file/dir.
type FsDeleteArgs struct {
	Path      string `json:"path"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsCreateArgs creates a file or directory.
type FsCreateArgs struct {
	Path      string `json:"path"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsWriteArgs writes text content to a file.
type FsWriteArgs struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Workspace WorkspaceEnv `json:"workspace"`
}

// FsWatchArgs adds/removes a directory from the watcher.
type FsWatchArgs struct {
	Paths     []string `json:"paths"`
	Workspace WorkspaceEnv `json:"workspace"`
}