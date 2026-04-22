// Package blob provides a narrow interface for external object storage.
// Implementations must be safe for concurrent use.
package blob

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound is returned by Open when the key does not exist.
var ErrNotFound = errors.New("blob: not found")

// Store is the minimum surface sample-result attachments need.
// The same interface is satisfied by the filesystem and S3 implementations.
type Store interface {
	// Put writes r under key. content is required for the S3 backend.
	Put(ctx context.Context, key, contentType string, size int64, r io.Reader) error

	// Open returns a reader for the object at key. Callers must Close it.
	// Returns ErrNotFound if key is absent.
	Open(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the object. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
}
