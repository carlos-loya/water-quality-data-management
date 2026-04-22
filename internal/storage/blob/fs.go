package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FSStore writes objects under a single base directory, using a two-level
// prefix derived from the key so directories don't balloon on large deployments.
type FSStore struct {
	baseDir string
}

func NewFSStore(baseDir string) (*FSStore, error) {
	if baseDir == "" {
		return nil, errors.New("blob: base dir required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("blob: create base dir: %w", err)
	}
	return &FSStore{baseDir: baseDir}, nil
}

func (s *FSStore) Put(_ context.Context, key, _ string, _ int64, r io.Reader) error {
	if err := validateKey(key); err != nil {
		return err
	}
	full := s.path(key)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("blob: mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".upload-*")
	if err != nil {
		return fmt.Errorf("blob: create temp: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("blob: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("blob: close: %w", err)
	}
	if err := os.Rename(tmpName, full); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("blob: rename: %w", err)
	}
	return nil
}

func (s *FSStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	f, err := os.Open(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("blob: open: %w", err)
	}
	return f, nil
}

func (s *FSStore) Delete(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := os.Remove(s.path(key)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("blob: delete: %w", err)
	}
	return nil
}

func (s *FSStore) path(key string) string {
	// Shard by the first two chars of the key so a directory listing stays tame.
	// Keys are opaque UUIDs at call sites, so this is purely a layout choice.
	prefix := key
	if len(prefix) >= 2 {
		prefix = prefix[:2]
	}
	return filepath.Join(s.baseDir, prefix, key)
}

func validateKey(key string) error {
	if key == "" {
		return errors.New("blob: empty key")
	}
	if strings.ContainsAny(key, "/\\") || strings.Contains(key, "..") {
		return fmt.Errorf("blob: invalid key %q", key)
	}
	return nil
}
