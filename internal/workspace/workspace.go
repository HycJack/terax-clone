// Package workspace owns the global "allowed directories" list and the
// per-workspace shell environment (local vs WSL distro).
package workspace

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"terax/internal/sysproc"
	"terax/internal/types"
)

var (
	mu        sync.Mutex
	allowed   = map[string]bool{}
	cwd       = ""
	launchDir = ""
)

// canonicalizePath resolves a path to its symlink-free absolute form. For
// not-yet-existing targets (about to be created) it canonicalizes the deepest
// existing ancestor and re-appends the remainder, so a write under a
// symlinked root (e.g. macOS /tmp -> /private/tmp) still matches.
func canonicalizePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	dir := abs
	var tail []string
	for {
		if r, err := filepath.EvalSymlinks(dir); err == nil {
			parts := append([]string{r}, tail...)
			return filepath.Join(parts...)
		}
		base := filepath.Base(dir)
		tail = append([]string{base}, tail...)
		parent := filepath.Dir(dir)
		if parent == dir {
			return abs
		}
		dir = parent
	}
}

// InitLaunchCwd sets the default cwd used when no workspace is selected.
// It also automatically authorises the directory so the user can browse
// files without going through the authorisation dialog on first launch.
func InitLaunchCwd(dir string) {
	mu.Lock()
	defer mu.Unlock()
	if dir != "" {
		launchDir = dir
		cwd = canonicalizePath(dir)
		allowed[cwd] = true
	} else if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = canonicalizePath(wd)
			allowed[cwd] = true
		}
	}
}

// Authorize marks a directory (recursively) as allowed.
func Authorize(dir string) error {
	if dir == "" {
		return errors.New("empty dir")
	}
	abs := canonicalizePath(dir)
	mu.Lock()
	defer mu.Unlock()
	allowed[abs] = true
	if cwd == "" {
		cwd = abs
	}
	return nil
}

// CurrentDir returns the active cwd.
func CurrentDir() string {
	mu.Lock()
	defer mu.Unlock()
	if cwd != "" {
		return strings.ReplaceAll(cwd, "\\", "/")
	}
	return ""
}

// SetCwd updates the active cwd if it's already authorized.
func SetCwd(dir string) error {
	abs := canonicalizePath(dir)
	mu.Lock()
	defer mu.Unlock()
	if !allowed[abs] && !allowed[filepath.Dir(abs)] {
		return errors.New("dir not authorized")
	}
	cwd = abs
	return nil
}

// WSL functions only operate on Windows; on other OSes they return empty.
func WSLListDistros() ([]WSLDistro, error) {
	if runtime.GOOS != "windows" {
		return nil, nil
	}
	// Use `-l -v` so we can also see whether each distro is running.
	verboseCmd := exec.Command("wsl.exe", "-l", "-v")
	sysproc.HideWindow(verboseCmd)
	verboseOut, _ := verboseCmd.Output()
	verboseLines := strings.Split(string(verboseOut), "\n")
	defaultName := ""
	runningSet := map[string]bool{}
	// Format: "  NAME      STATE           VERSION"
	for i, line := range verboseLines {
		line = strings.TrimRight(line, "\r")
		if i == 0 || strings.HasPrefix(line, "Windows Subsystem for Linux") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSuffix(fields[0], "*")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.HasSuffix(fields[0], "*") {
			defaultName = name
		}
		if fields[1] == "Running" {
			runningSet[name] = true
		}
	}
	// Use the bare `-l -q` to enumerate names only.
	listCmd := exec.Command("wsl.exe", "-l", "-q")
	sysproc.HideWindow(listCmd)
	out, err := listCmd.Output()
	if err != nil {
		return nil, err
	}
	var distros []WSLDistro
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || line == "Windows Subsystem for Linux Distributions:" {
			continue
		}
		distros = append(distros, WSLDistro{
			Name:    line,
			Default: line == defaultName,
			Running: runningSet[line],
		})
	}
	return distros, nil
}

// WSLDistro mirrors the Rust backend's `WslDistro` (used by the frontend).
type WSLDistro struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
	Running bool   `json:"running"`
}

func WSLDefaultDistro() (string, error) {
	if runtime.GOOS != "windows" {
		return "", nil
	}
	verboseCmd2 := exec.Command("wsl.exe", "-l", "-v")
	sysproc.HideWindow(verboseCmd2)
	out, err := verboseCmd2.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return "", errors.New("wsl not installed")
	}
	// The default distro has "*" as the first character of its row.
	for _, line := range lines[1:] {
		if strings.HasPrefix(strings.TrimSpace(line), "*") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}
	// Fallback: pick the first listed distro.
	fields := strings.Fields(lines[1])
	if len(fields) >= 1 {
		return fields[0], nil
	}
	return "", errors.New("no distros")
}

func WSLHome(distro string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", nil
	}
	homeCmd := exec.Command("wsl.exe", "-d", distro, "sh", "-c", "echo $HOME")
	sysproc.HideWindow(homeCmd)
	out, err := homeCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// TranslateWSLPath converts `\\wsl$\<distro>\path` to a /mnt style path.
// Not strictly needed for the Wails build but kept for parity with the
// Rust backend's `to_canon`.
func TranslateWSLPath(p string) string {
	if !strings.HasPrefix(p, "\\\\wsl$\\") && !strings.HasPrefix(p, "\\\\wsl.localhost\\") {
		return p
	}
	parts := strings.SplitN(strings.TrimPrefix(strings.TrimPrefix(p, "\\\\wsl$\\"), "\\\\wsl.localhost\\"), "\\", 2)
	if len(parts) < 2 || parts[1] == "" {
		return p
	}
	_ = parts[0]
	rest := strings.ReplaceAll(parts[1], "\\", "/")
	if rest == "" {
		return p
	}
	return "/mnt/" + strings.ToLower(string(rest[0])) + rest[1:]
}

// IsAuthorized reports whether a path is in the registry.
func IsAuthorized(p string) bool {
	// Resolve symlinks so a link inside an authorized tree that points
	// outside (e.g. `proj/link -> /etc/passwd`) can't smuggle reads/writes
	// out of the sandbox. Authorized roots are stored canonical too, so
	// symlink aliases of a workspace (macOS /tmp -> /private/tmp) still
	// match.
	check := canonicalizePath(p)
	mu.Lock()
	defer mu.Unlock()
	if allowed[check] {
		return true
	}
	sep := string(filepath.Separator)
	for a := range allowed {
		if strings.HasPrefix(check, a+sep) || check == a {
			return true
		}
	}
	return false
}

// ResolveWorkspaceEnv normalizes the workspace env the frontend passes in.
func ResolveWorkspaceEnv(e types.WorkspaceEnv) types.WorkspaceEnv {
	if e.Kind == "" {
		e.Kind = "local"
	}
	if e.Cwd != "" {
		e.Cwd = strings.ReplaceAll(e.Cwd, "\\", "/")
	}
	return e
}