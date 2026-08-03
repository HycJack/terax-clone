package lsp

import (
	"os"
	"path/filepath"
	"strings"
)

// Real implementations of the helpers in lsp.go. Kept in a separate file
// so tests can override them by reassigning the package-level vars.
func osStatImpl(path string) (any, error) {
	_, err := os.Stat(path)
	return nil, err
}

func parentImpl(p string) string {
	parent := filepath.Dir(p)
	if parent == p {
		return p
	}
	return filepath.ToSlash(strings.TrimRight(parent, string(filepath.Separator)))
}

func envImpl() []string {
	return os.Environ()
}

func pidImpl() int {
	return os.Getpid()
}