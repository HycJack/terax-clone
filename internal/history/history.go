// Package history records shell commands the user runs and exposes simple
// fuzzy / prefix matching back to the frontend. We persist the history as
// one command per line under the user's app data directory.
package history

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	mu     sync.Mutex
	limit  = 5000
	file   string
	loaded = false
	cache  []string
)

// RecordArgs is the request body for Record.
type RecordArgs struct {
	Command string `json:"command"`
}

// ListArgs is the request body for List.
type ListArgs struct {
	Limit int `json:"limit"`
}

// SuggestArgs is the request body for Suggest.
type SuggestArgs struct {
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
}

// Init points the package at a per-user file path. Safe to call multiple times.
func Init(path string) error {
	mu.Lock()
	defer mu.Unlock()
	if path == "" {
		return errors.New("empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file = path
	loaded = false
	return nil
}

func ensureLoaded() error {
	if loaded {
		return nil
	}
	if file == "" {
		file = filepath.Join(defaultDir(), "history")
	}
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			cache = []string{}
			loaded = true
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	cache = []string{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			cache = append(cache, line)
		}
	}
	loaded = true
	return nil
}

// Record appends a command to the history (in memory and on disk).
func Record(args RecordArgs) error {
	if strings.TrimSpace(args.Command) == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoaded(); err != nil {
		return err
	}
	if len(cache) > 0 && cache[len(cache)-1] == args.Command {
		return nil
	}
	cache = append(cache, args.Command)
	if len(cache) > limit {
		cache = cache[len(cache)-limit:]
	}
	return persist()
}

// List returns the last N commands (most recent last).
func List(args ListArgs) ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	n := args.Limit
	if n <= 0 || n > len(cache) {
		n = len(cache)
	}
	out := make([]string, n)
	copy(out, cache[len(cache)-n:])
	return out, nil
}

// Suggest returns commands whose prefix matches.
func Suggest(args SuggestArgs) ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	prefix := strings.ToLower(args.Prefix)
	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	seen := map[string]bool{}
	out := []string{}
	for i := len(cache) - 1; i >= 0 && len(out) < limit; i-- {
		cmd := cache[i]
		if !strings.HasPrefix(strings.ToLower(cmd), prefix) {
			continue
		}
		if seen[cmd] {
			continue
		}
		seen[cmd] = true
		out = append(out, cmd)
	}
	sort.Strings(out)
	return out, nil
}

// CommandsArgs is the request body for Commands.
type CommandsArgs struct {
	Prefix string `json:"prefix,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// Commands returns the unique set of commands ever run (de-duped). If a
// prefix is supplied, only commands starting with that prefix are
// returned; limit caps the result list.
func Commands(args CommandsArgs) ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	prefix := args.Prefix
	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	seen := map[string]bool{}
	out := []string{}
	for _, c := range cache {
		if prefix != "" && !strings.HasPrefix(c, prefix) {
			continue
		}
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func persist() error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, c := range cache {
		_, _ = w.WriteString(c)
		_ = w.WriteByte('\n')
	}
	return w.Flush()
}

func defaultDir() string {
	if d, err := os.UserHomeDir(); err == nil {
		return filepath.Join(d, ".config", "terax")
	}
	return ".terax"
}