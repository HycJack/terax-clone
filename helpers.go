// Helper functions referenced by the App methods in app.go. Keeping them
// here avoids the main file becoming a wall of trivial glue.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"terax/internal/sysproc"

	internalsecrets "terax/internal/secrets"
	"terax/internal/store"

	internethistory "terax/internal/history"
	"terax/internal/net"
)

// =====================================================================
// Store (LazyStore persistence)
// =====================================================================

func storeInit(dir string) {
	store.Init(dir)
}

func storeLoad(path string) (map[string]interface{}, error) {
	return store.Load(store.LoadArgs{Path: path})
}

func storeSave(path string, data map[string]interface{}) error {
	return store.Save(store.SaveArgs{Path: path, Data: data})
}

// =====================================================================
// History
// =====================================================================

func historyInit(path string) {
	_ = internethistory.Init(path)
}

func historySuggest(prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	return internethistory.Suggest(internethistory.SuggestArgs{Prefix: prefix, Limit: limit})
}

func historyRecord(cmd string) error {
	return internethistory.Record(internethistory.RecordArgs{Command: cmd})
}

func historyList(limit int) ([]string, error) {
	return internethistory.List(internethistory.ListArgs{Limit: limit})
}

func historyCommands(prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	return internethistory.Commands(internethistory.CommandsArgs{Prefix: prefix, Limit: limit})
}

// =====================================================================
// Secrets (keyring)
// =====================================================================

func secretsGet(service, account string) (string, error) {
	if service != "" {
		internalsecrets.SetService(service)
	}
	return internalsecrets.Get(account)
}

func secretsSet(service, account, password string) error {
	if service != "" {
		internalsecrets.SetService(service)
	}
	return internalsecrets.Set(account, password)
}

func secretsDelete(service, account string) error {
	if service != "" {
		internalsecrets.SetService(service)
	}
	return internalsecrets.Delete(account)
}

func secretsGetAll(service string, accounts []string) ([]string, error) {
	if service != "" {
		internalsecrets.SetService(service)
	}
	return internalsecrets.GetAll(accounts)
}

// =====================================================================
// Net / AI HTTP
// =====================================================================

func netLmPing(ctx context.Context, url string) (int, error) {
	return net.LmPing(ctx, net.PingArgs{BaseURL: url})
}

func netAiHTTPRequest(ctx context.Context, args net.AiHTTPRequestArgs) (map[string]interface{}, error) {
	return net.AiHTTPRequest(ctx, args)
}

func netAiHTTPStream(ctx context.Context, args net.AiHTTPStreamArgs) error {
	return net.AiHTTPStream(ctx, args)
}

// =====================================================================
// Process / window-state / updater / autostart / opener
// =====================================================================

// processRelaunch starts a fresh copy of the binary and exits. Works on
// every OS we ship binaries for.
func processRelaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := os.Args[1:]
	cmd := exec.Command(exe, args...)
	sysproc.HideWindow(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

// autostartEnable / Disable / IsEnabled write a Run-key entry under
// HKCU\Software\Microsoft\Windows\CurrentVersion\Run. On non-Windows
// hosts we return errors so the frontend can fall back to no-op.

func autostartEnable() error {
	if runtime.GOOS != "windows" {
		return errors.New("autostart: only implemented on Windows")
	}
	return autostartSet(true)
}

func autostartDisable() error {
	if runtime.GOOS != "windows" {
		return errors.New("autostart: only implemented on Windows")
	}
	return autostartSet(false)
}

func autostartIsEnabled() bool {
	v, _ := autostartGet()
	return v
}

// autostartSet writes the registry value via `reg add`/`reg delete`.
func autostartSet(enable bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	name := "Terax"
	if enable {
		cmd := exec.Command("reg", "add", key, "/v", name, "/t", "REG_SZ", "/d", exe, "/f")
		sysproc.HideWindow(cmd)
		return cmd.Run()
	}
	cmd := exec.Command("reg", "delete", key, "/v", name, "/f")
	sysproc.HideWindow(cmd)
	_ = cmd.Run()
	return nil
}

func autostartGet() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	cmd := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Terax")
	sysproc.HideWindow(cmd)
	return cmd.Run() == nil, nil
}

// updaterCheck pings a known endpoint for an update. We don't ship a full
// release-server here; the function is a placeholder that always returns
// nil so the frontend's `update.available` UI doesn't blow up.
func updaterCheck() map[string]interface{} {
	// No remote release server configured; report "no update".
	return map[string]interface{}{
		"available":      false,
		"currentVersion": "0.0.0",
		"version":        "0.0.0",
	}
}

// openerOpenURL uses the system shell to open a URL in the default browser.
func openerOpenURL(u string) error {
	if u == "" {
		return errors.New("empty url")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") &&
		!strings.HasPrefix(u, "mailto:") && !strings.HasPrefix(u, "tel:") {
		return errors.New("unsupported scheme")
	}
	return openOS(u)
}

// openerOpenPath opens a file with the OS default app.
func openerOpenPath(p string) error {
	if p == "" {
		return errors.New("empty path")
	}
	return openOS(p)
}

// openerRevealItem opens the file explorer at `path`.
func openerRevealItem(p string) error {
	if p == "" {
		return errors.New("empty path")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return func() error {
			cmd := exec.Command("explorer.exe", "/select,", strings.ReplaceAll(abs, "/", "\\"))
			sysproc.HideWindow(cmd)
			return cmd.Start()
		}()
	case "darwin":
		return func() error {
			cmd := exec.Command("open", "-R", abs)
			sysproc.HideWindow(cmd)
			return cmd.Start()
		}()
	default:
		return func() error {
			cmd := exec.Command("xdg-open", filepath.Dir(abs))
			sysproc.HideWindow(cmd)
			return cmd.Start()
		}()
	}
}

// openOS dispatches to the host's "open this URL/file" command.
func openOS(target string) error {
	switch runtime.GOOS {
	case "windows":
		return func() error {
			cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target)
			sysproc.HideWindow(cmd)
			return cmd.Start()
		}()
	case "darwin":
		return func() error {
			cmd := exec.Command("open", target)
			sysproc.HideWindow(cmd)
			return cmd.Start()
		}()
	default:
		return func() error {
			cmd := exec.Command("xdg-open", target)
			sysproc.HideWindow(cmd)
			return cmd.Start()
		}()
	}
}

// Keep `net/http` and `fmt` referenced so the file compiles even if we
// later swap implementations.
var _ = http.Client{}
var _ = fmt.Sprintf