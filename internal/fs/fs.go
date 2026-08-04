// Package fs implements the filesystem commands the frontend invokes.
//
// The semantics match the Rust commands 1:1: paths are returned with forward
// slashes regardless of host OS so the frontend's string-joining logic
// keeps working. Hidden / gitignore filtering follows the same conventions
// as the original `ignore`-based backend.
package fs

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"terax/internal/sysproc"
	"terax/internal/types"
)

// WatcherManager keeps one fsnotify watcher per app session and forwards
// change events as a single "fs:changed" event to the frontend.
type WatcherManager struct {
	mu      sync.Mutex
	watcher *fsnotify.Watcher
	watched map[string]bool
	ctx     context.Context
}

// NewWatcherManager wires up the singleton watcher.
func NewWatcherManager() *WatcherManager {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fsnotify: %v\n", err)
	}
	mgr := &WatcherManager{watcher: w, watched: map[string]bool{}}
	if w != nil {
		go mgr.pump()
	}
	return mgr
}

// BindContext attaches the Wails app context so we can emit events.
func (m *WatcherManager) BindContext(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx = ctx
}

// Add starts watching the given directories.
func (m *WatcherManager) Add(paths []string) {
	if m.watcher == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if m.watched[abs] {
			continue
		}
		if err := m.watcher.Add(abs); err == nil {
			m.watched[abs] = true
		}
	}
}

// Remove stops watching.
func (m *WatcherManager) Remove(paths []string) {
	if m.watcher == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if !m.watched[abs] {
			continue
		}
		_ = m.watcher.Remove(abs)
		delete(m.watched, abs)
	}
}

func (m *WatcherManager) pump() {
	if m.watcher == nil {
		return
	}
	// Coalesce rapid bursts of events so the frontend doesn't get
	// overwhelmed on save-heavy directories.
	coalesce := 50 * time.Millisecond
	t := time.NewTimer(coalesce)
	t.Stop()
	pending := map[string]bool{}
	for {
		select {
		case ev, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			pending[ev.Name] = true
			t.Reset(coalesce)
		case <-t.C:
			if len(pending) == 0 {
				continue
			}
			paths := make([]string, 0, len(pending))
			for k := range pending {
				paths = append(paths, k)
			}
			pending = map[string]bool{}
			m.mu.Lock()
			ctx := m.ctx
			m.mu.Unlock()
			if ctx != nil {
				wailsruntime.EventsEmit(ctx, "fs:changed", paths)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "fsnotify error: %v\n", err)
		}
	}
}

// Canonicalize returns an absolute path with forward slashes.
func Canonicalize(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return strings.ReplaceAll(abs, "\\", "/")
}

// ReadDir lists a directory's entries, filtering hidden / gitignored per flags.
func ReadDir(args types.FsReadDirArgs) ([]types.DirEntry, error) {
	abs, err := filepath.Abs(args.Path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	// Build a quick set of gitignored paths by deferring to `git check-ignore`
	// when decorations are requested. The Rust backend used the `ignore` crate
	// here; we trade fidelity for portability by using git's own machinery.
	ignoreSet := map[string]bool{}
	if args.GitDecorations && looksLikeRepo(abs) {
		ignoreSet = computeGitIgnored(abs, info)
	}

	out := make([]types.DirEntry, 0, len(info))
	for _, e := range info {
		name := e.Name()
		if !args.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(abs, name)
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		} else if e.Mode()&os.ModeSymlink != 0 {
			kind = "symlink"
		}
		out = append(out, types.DirEntry{
			Name:       name,
			Kind:       kind,
			Size:       e.Size(),
			Mtime:      e.ModTime().Unix(),
			GitIgnored: ignoreSet[full],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			// Directories first.
			return out[i].Kind == "dir"
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func looksLikeRepo(abs string) bool {
	_, err := os.Stat(filepath.Join(abs, ".git"))
	return err == nil
}

func computeGitIgnored(abs string, entries []os.FileInfo) map[string]bool {
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Name())
	}
	cmd := exec.Command("git", append([]string{"check-ignore", "--no-index", "--"}, paths...)...)
	sysproc.HideWindow(cmd)
	cmd.Dir = abs
	out, err := cmd.Output()
	if err != nil {
		// Exit 1 = no ignored paths; that's fine.
		if _, ok := err.(*exec.ExitError); ok {
			return map[string]bool{}
		}
		return map[string]bool{}
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	set := map[string]bool{}
	for sc.Scan() {
		// Output format: "<name>\t<line>:<col>:<name>"
		line := sc.Text()
		tab := strings.IndexByte(line, '\t')
		var name string
		if tab < 0 {
			name = strings.TrimSpace(line)
		} else {
			name = line[:tab]
		}
		if name != "" {
			set[filepath.Join(abs, name)] = true
		}
	}
	return set
}

// Stat mirrors `fs_stat`.
func Stat(path string) (types.FsStat, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return types.FsStat{Kind: ""}, nil
		}
		return types.FsStat{}, err
	}
	kind := "file"
	if info.Mode()&os.ModeSymlink != 0 {
		kind = "symlink"
	} else if info.IsDir() {
		kind = "dir"
	}
	return types.FsStat{
		Size:  info.Size(),
		Mtime: info.ModTime().Unix(),
		Kind:  kind,
	}, nil
}

// ReadFile reads a text file as UTF-8. Binary detection is best-effort.
// ReadFileResult mirrors the frontend's ReadResult envelope.
type ReadFileResult = types.ReadResult

// ReadFile reads a file and returns a ReadResult envelope so the frontend
// can distinguish text / binary / too-large.
func ReadFile(path string) (ReadFileResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ReadFileResult{}, err
	}
	const limit = 10 * 1024 * 1024 // 10 MB
	if info.Size() > limit {
		return ReadFileResult{Kind: "toolarge", Size: info.Size(), Limit: limit}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ReadFileResult{}, err
	}
	if isBinary(data) {
		return ReadFileResult{Kind: "binary", Size: int64(len(data))}, nil
	}
	return ReadFileResult{Kind: "text", Content: string(data), Size: int64(len(data))}, nil
}

// isBinary returns true if the byte slice contains a NUL byte in the first
// 8 KB — a heuristic that matches git/GitHub's binary detection.
func isBinary(data []byte) bool {
	n := len(data)
	if n > 8192 {
		n = 8192
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// WriteFile writes UTF-8 text to a path. Parent directories are created.
func WriteFile(args types.FsWriteArgs) error {
	if err := os.MkdirAll(filepath.Dir(args.Path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(args.Path, []byte(args.Content), 0o644)
}

// CreateFile creates an empty file.
func CreateFile(args types.FsCreateArgs) error {
	if err := os.MkdirAll(filepath.Dir(args.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(args.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

// CreateDir creates a directory (recursively).
func CreateDir(args types.FsCreateArgs) error {
	return os.MkdirAll(args.Path, 0o755)
}

// Rename moves/renames a path.
func Rename(args types.FsRenameArgs) error {
	if err := os.MkdirAll(filepath.Dir(args.To), 0o755); err != nil {
		return err
	}
	return os.Rename(args.From, args.To)
}

// Delete removes a file or directory recursively.
func Delete(args types.FsDeleteArgs) error {
	return os.RemoveAll(args.Path)
}

// Copy copies one or more source paths into the destination directory,
// recursively for dirs. Mirrors the Rust backend's `fs_copy` which takes
// a batch of sources. Refuses to overwrite existing entries (the file
// manager will surface the error to the user via the toast).
func Copy(args types.FsCopyArgs) error {
	dest := args.DestDir
	if dest == "" {
		return errors.New("empty destination")
	}
	for _, source := range args.Sources {
		info, err := os.Stat(source)
		if err != nil {
			return err
		}
		name := filepath.Base(source)
		target := filepath.Join(dest, name)
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("already exists: %s", target)
		}
		if info.IsDir() {
			if err := copyDir(source, target); err != nil {
				return err
			}
		} else {
			if err := copyFile(source, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		childSrc := filepath.Join(src, e.Name())
		childDst := filepath.Join(dst, e.Name())
		info, err := os.Stat(childSrc)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := copyDir(childSrc, childDst); err != nil {
				return err
			}
		} else {
			if err := copyFile(childSrc, childDst); err != nil {
				return err
			}
		}
	}
	return nil
}

// Search walks the directory tree looking for `query` as a substring of
// the filename. Frontend uses this for the explorer search bar.
func Search(args types.FsSearchArgs) (types.FsSearchResponse, error) {
	root, err := filepath.Abs(args.Root)
	if err != nil {
		return types.FsSearchResponse{}, err
	}
	q := strings.ToLower(args.Query)
	max := args.MaxResults
	if max <= 0 {
		max = 500
	}
	var hits []types.FsSearchHit
	truncated := false
	rootClean := strings.ReplaceAll(root, "\\", "/")
	rootPrefix := rootClean
	if !strings.HasSuffix(rootPrefix, "/") {
		rootPrefix += "/"
	}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return nil
			}
			return nil
		}
		if !args.ShowHidden && d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			if path != root {
				return filepath.SkipDir
			}
		}
		if len(hits) >= max {
			truncated = true
			return filepath.SkipAll
		}
		if strings.Contains(strings.ToLower(d.Name()), q) {
			clean := strings.ReplaceAll(path, "\\", "/")
			hits = append(hits, types.FsSearchHit{
				Path:   clean,
				Rel:    strings.TrimPrefix(clean, rootPrefix),
				Name:   d.Name(),
				IsDir:  d.IsDir(),
			})
		}
		return nil
	})
	return types.FsSearchResponse{Hits: hits, Truncated: truncated}, err
}

// Glob is a small wrapper around filepath.Glob. Returns hits with both
// absolute and root-relative paths so the frontend can render them in
// its tree view without re-computing.
func Glob(args types.FsGlobArgs) (types.FsGlobResponse, error) {
	pattern := args.Pattern
	if args.Root != "" && !filepath.IsAbs(pattern) {
		pattern = filepath.Join(args.Root, pattern)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return types.FsGlobResponse{}, err
	}
	rootClean := strings.ReplaceAll(args.Root, "\\", "/")
	rootPrefix := rootClean
	if !strings.HasSuffix(rootPrefix, "/") {
		rootPrefix += "/"
	}
	max := args.MaxResults
	if max <= 0 {
		max = 500
	}
	hits := make([]types.FsGlobHit, 0, len(matches))
	truncated := false
	for _, m := range matches {
		clean := strings.ReplaceAll(m, "\\", "/")
		hits = append(hits, types.FsGlobHit{
			Path: clean,
			Rel:  strings.TrimPrefix(clean, rootPrefix),
		})
		if len(hits) >= max {
			truncated = true
			break
		}
	}
	return types.FsGlobResponse{Hits: hits, Truncated: truncated}, nil
}

// ListFiles walks `root` collecting regular file paths up to a cap. Mirrors
// the Rust backend's `ListFilesResult` envelope so the frontend can show
// "X files (truncated)" hints.
func ListFiles(args types.FsListFilesArgs) (types.FsListFilesResult, error) {
	root := args.Root
	if root == "" {
		root = "."
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}
	files := make([]string, 0, limit)
	truncated := false
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, strings.ReplaceAll(path, "\\", "/"))
		if len(files) >= limit {
			truncated = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return types.FsListFilesResult{}, err
	}
	return types.FsListFilesResult{Files: files, Truncated: truncated}, nil
}

// Grep runs a content search via the system `grep`/`rg` if available,
// otherwise falls back to an in-Go line scanner. The frontend uses this for
// `find in files`.
func Grep(args types.FsGrepArgs) (types.FsGrepResponse, error) {
	root := args.Root
	if root == "" {
		root = "."
	}
	max := args.MaxResults
	if max <= 0 {
		max = 1000
	}

	// Prefer rg when installed.
	if _, err := exec.LookPath("rg"); err == nil {
		return grepRg(args, root, max)
	}
	return grepGo(args, root, max)
}

func grepRg(args types.FsGrepArgs, root string, max int) (types.FsGrepResponse, error) {
	argv := []string{
		"--no-heading",
		"--line-number",
		"--no-messages",
	}
	// Frontend uses `caseInsensitive: true` to mean "ignore case" (the
	// everyday, opt-in semantics). rg's `-s` / `-i` flags are
	// case-sensitive-default; flip them to match.
	if args.CaseInsensitive {
		argv = append(argv, "-i")
	} else {
		argv = append(argv, "-s")
	}
	// Treat the pattern as a literal substring (not a regex) — matches the
	// Rust backend's behavior. Users who want regex can still pass `regex`.
	if len(args.Glob) > 0 {
		for _, g := range args.Glob {
			argv = append(argv, "--glob", g)
		}
	}
	argv = append(argv, "--fixed-strings")
	argv = append(argv, "--", args.Pattern, root)
	cmd := exec.Command("rg", argv...)
	sysproc.HideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return types.FsGrepResponse{Hits: []types.FsGrepHit{}}, nil
		}
		return types.FsGrepResponse{}, err
	}
	rootClean := strings.ReplaceAll(root, "\\", "/")
	rootPrefix := rootClean
	if !strings.HasSuffix(rootPrefix, "/") {
		rootPrefix += "/"
	}
	var hits []types.FsGrepHit
	truncated := false
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		colon1 := strings.IndexByte(line, ':')
		if colon1 < 0 {
			continue
		}
		path := strings.ReplaceAll(line[:colon1], "\\", "/")
		rest := line[colon1+1:]
		colon2 := strings.IndexByte(rest, ':')
		if colon2 < 0 {
			continue
		}
		var lineNum int
		fmt.Sscanf(rest[:colon2], "%d", &lineNum)
		text := rest[colon2+1:]
		rel := strings.TrimPrefix(path, rootPrefix)
		hits = append(hits, types.FsGrepHit{Path: path, Rel: rel, Line: lineNum, Text: text})
		if len(hits) >= max {
			truncated = true
			break
		}
	}
	// `files_scanned` is best-effort — rg doesn't print it without extra
	// flags, so we report the count of unique file paths we found.
	unique := map[string]struct{}{}
	for _, h := range hits {
		unique[h.Path] = struct{}{}
	}
	return types.FsGrepResponse{
		Hits:         hits,
		Truncated:    truncated,
		FilesScanned: len(unique),
	}, nil
}

func grepGo(args types.FsGrepArgs, root string, max int) (types.FsGrepResponse, error) {
	var hits []types.FsGrepHit
	pattern := args.Pattern
	caseInsensitive := args.CaseInsensitive
	if caseInsensitive {
		pattern = strings.ToLower(pattern)
	}
	rootClean := strings.ReplaceAll(root, "\\", "/")
	rootPrefix := rootClean
	if !strings.HasSuffix(rootPrefix, "/") {
		rootPrefix += "/"
	}
	filesScanned := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		filesScanned++
		if len(args.Glob) > 0 {
			matched := false
			for _, g := range args.Glob {
				if m, _ := filepath.Match(g, d.Name()); m {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		lineNum := 0
		for sc.Scan() {
			lineNum++
			text := sc.Text()
			hay := text
			if caseInsensitive {
				hay = strings.ToLower(text)
			}
			if strings.Contains(hay, pattern) {
				cleanPath := strings.ReplaceAll(path, "\\", "/")
				hits = append(hits, types.FsGrepHit{
					Path: cleanPath,
					Rel:  strings.TrimPrefix(cleanPath, rootPrefix),
					Line: lineNum,
					Text: text,
				})
				if len(hits) >= max {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	// If we hit max in the Go walker, the result is truncated.
	truncated := len(hits) >= max
	return types.FsGrepResponse{
		Hits:         hits,
		Truncated:    truncated,
		FilesScanned: filesScanned,
	}, err
}

// AppDir returns the per-user data directory the runtime expects.
func AppDir() string {
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