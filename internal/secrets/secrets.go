// Package secrets stores AI provider keys in the OS keyring. We use
// `github.com/zalando/go-keyring`, which transparently selects the right
// backend per OS (Credential Manager on Windows, Keychain on macOS, Secret
// Service on Linux).
package secrets

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

var (
	mu      sync.Mutex
	service = "terax-ai"
)

// SetService overrides the service name (used by the frontend for
// `KEYRING_SERVICE`).
func SetService(s string) {
	mu.Lock()
	defer mu.Unlock()
	if s != "" {
		service = s
	}
}

// Service returns the currently configured service name.
func Service() string {
	mu.Lock()
	defer mu.Unlock()
	return service
}

// Get returns the stored password or "" if missing.
func Get(account string) (string, error) {
	v, err := keyring.Get(service, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

// Set stores a password under (service, account).
func Set(account, password string) error {
	return keyring.Set(service, account, password)
}

// Delete removes the password under (service, account).
func Delete(account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// GetAll returns the values for many accounts in one call. Missing entries
// surface as empty strings, matching the frontend's expectation.
func GetAll(accounts []string) ([]string, error) {
	out := make([]string, len(accounts))
	for i, a := range accounts {
		v, err := Get(a)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}