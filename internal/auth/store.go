package auth

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

// ErrNotFound means a Store holds nothing under a key.
var ErrNotFound = errors.New("not found")

// Store keeps Harold's secrets.
type Store interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// keyringService names Harold's entries in the OS keyring.
const keyringService = "harold"

// Keyring is the OS keyring: Secret Service on Linux, the Keychain on macOS.
type Keyring struct{}

func (Keyring) Get(key string) (string, error) {
	value, err := keyring.Get(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return value, err
}

func (Keyring) Set(key, value string) error { return keyring.Set(keyringService, key, value) }

func (Keyring) Delete(key string) error {
	err := keyring.Delete(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// Memory is a Store in memory, for tests.
type Memory struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *Memory) Get(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (m *Memory) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[key] = value
	return nil
}

func (m *Memory) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
	return nil
}
