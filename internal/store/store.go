// Package store persists the frontend's LazyStore JSON bags.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mu      sync.Mutex
	dir     string
	dirty   = map[string]bool{}
	pending = map[string][]byte{}
)

// LoadArgs is the request body for Load.
type LoadArgs struct {
	Path string `json:"path"`
}

// SaveArgs is the request body for Save.
type SaveArgs struct {
	Path string                 `json:"path"`
	Data map[string]interface{} `json:"data"`
}

// Init points the package at the per-user data dir.
func Init(d string) {
	mu.Lock()
	defer mu.Unlock()
	if d != "" {
		dir = d
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		dir = ""
	}
}

// Load reads a JSON bag from disk; returns an empty object if missing.
func Load(args LoadArgs) (map[string]interface{}, error) {
	mu.Lock()
	defer mu.Unlock()
	if dir == "" {
		return map[string]interface{}{}, nil
	}
	p, err := safePath(dir, args.Path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Save writes a JSON bag to disk.
func Save(args SaveArgs) error {
	mu.Lock()
	defer mu.Unlock()
	if dir == "" {
		return errors.New("store dir not initialized")
	}
	p, err := safePath(dir, args.Path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(args.Data, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write via temp file.
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// safePath joins dir + path.json and validates the result stays inside dir.
func safePath(dir, path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	// Reject path traversal attempts.
	if strings.Contains(path, "..") || strings.ContainsAny(path, "/\\") {
		return "", errors.New("invalid path: no separators or .. allowed")
	}
	p := filepath.Join(dir, path+".json")
	// Final check: resolved path must be inside dir.
	absDir, _ := filepath.Abs(dir)
	absP, _ := filepath.Abs(p)
	if !strings.HasPrefix(absP, absDir+string(filepath.Separator)) && absP != absDir {
		return "", errors.New("path escapes store directory")
	}
	return p, nil
}