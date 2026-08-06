// Shell init script handling.
//
// The frontend tracks the shell's current working directory and command
// boundaries via OSC 7 (`\e]7;file://cwd\e\\`) and OSC 133 (`\e]133;A/B/C/D`)
// escape sequences. The local shell only emits those when it has our
// integration snippets installed — bash/PowerShell don't emit them by
// default.
//
// We write per-shell init files into `~/.cache/terax/shell-integration/<shell>/`
// at session-open time and pass them to the shell as `--rcfile` (bash) or
// `-File` (PowerShell). The shell then sources these snippets at startup,
// emitting OSC 7 from the precmd/PROMPT_COMMAND hook so the frontend can
// keep the directory tree root in sync with the active terminal.
//
// The scripts are short and stable; they're written once per cache-version
// and reused for every subsequent session to avoid disk churn.

package pty

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Scripts embedded from disk at build time. Keeping them as `embed` instead
// of `include_str!` lets Go cross-compile without us pinning the relative
// path the way Tauri's `include_str!` would.
import _ "embed"

//go:embed scripts/bashrc.bash
var bashrcScript string

//go:embed scripts/profile.ps1
var profilePSScript string

// integrationDir returns the per-user directory where shell init snippets
// live. We write once per script and only re-write when the embedded
// version changes.
func integrationDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".cache", "terax", "shell-integration")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// writeIfChanged is the atomic replace helper used by all prepare* helpers:
// a parallel shell startup must never source a half-written file.
func writeIfChanged(path, content string) error {
	if existing, err := os.ReadFile(path); err == nil && string(existing) == content {
		return nil
	}
	tmp := path + ".__terax_tmp__"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// prepareBashRcfile writes the OSC 7/133 emitting bashrc to a per-user
// cache directory and returns its path (a forward-slash form for cygwin-style
// environments). The path is suitable for bash's `--rcfile` argument.
func prepareBashRcfile() (string, error) {
	dir, err := integrationDir()
	if err != nil {
		return "", err
	}
	bashDir := filepath.Join(dir, "bash")
	if err := os.MkdirAll(bashDir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", bashDir, err)
	}
	rc := filepath.Join(bashDir, "bashrc")
	if err := writeIfChanged(rc, bashrcScript); err != nil {
		return "", err
	}
	// On Windows, bash expects forward-slash paths (MSYS convention).
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(rc, `\`, "/"), nil
	}
	return rc, nil
}

// preparePsProfile writes the OSC 7/133 emitting PowerShell profile.ps1 and
// returns its path. PowerShell reads it via `-File` and runs the script at
// session start.
func preparePsProfile() (string, error) {
	dir, err := integrationDir()
	if err != nil {
		return "", err
	}
	psDir := filepath.Join(dir, "powershell")
	if err := os.MkdirAll(psDir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", psDir, err)
	}
	profile := filepath.Join(psDir, "profile.ps1")
	if err := writeIfChanged(profile, profilePSScript); err != nil {
		return "", err
	}
	return profile, nil
}

// classifyShell returns a coarse shell kind for the given binary path.
// We only need this for picking which integration to apply; the actual
// detection of pwsh vs powershell vs bash etc. is path-based.
func classifyShell(shellPath string) string {
	// basename case-insensitive
	base := filepath.Base(shellPath)
	lower := strings.ToLower(base)
	switch lower {
	case "bash.exe", "bash":
		return "bash"
	case "pwsh.exe", "pwsh":
		return "pwsh"
	case "powershell.exe", "powershell":
		return "powershell"
	case "cmd.exe", "cmd":
		return "cmd"
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	default:
		return "other"
	}
}

// integrationConfig describes the per-shell integration commands that need
// to be appended to the shell's argv. `extraEnv` holds additional
// environment variables that the integration expects (e.g. CHERE_INVOKING
// to suppress git-bash's /etc/profile default-cd).
type integrationConfig struct {
	extraArgs []string
	extraEnv  []string
}

// integrationFor returns the integration config for the shell at shellPath.
// If the shell has no integration, it returns nil and the caller should
// just spawn the shell directly.
func integrationFor(shellPath string) (integrationConfig, error) {
	switch classifyShell(shellPath) {
	case "bash":
		rc, err := prepareBashRcfile()
		if err != nil {
			return integrationConfig{}, err
		}
		return integrationConfig{
			// bash ignores --rcfile when started with -l; we use -i instead
			// and replicate login init inside the rcfile.
			extraArgs: []string{"--rcfile", rc, "-i"},
			// git-bash's /etc/profile cd's to $HOME unless CHERE_INVOKING is
			// set; keep the cwd the caller configured.
			extraEnv: []string{"CHERE_INVOKING=1"},
		}, nil
	case "pwsh", "powershell":
		profile, err := preparePsProfile()
		if err != nil {
			return integrationConfig{}, err
		}
		return integrationConfig{
			extraArgs: []string{"-NoLogo", "-NoExit", "-ExecutionPolicy", "Bypass", "-File", profile},
		}, nil
	default:
		// cmd, zsh, fish, "other" — no integration for now. Future shells
		// can be added here without touching the rest of the codebase.
		return integrationConfig{}, nil
	}
}