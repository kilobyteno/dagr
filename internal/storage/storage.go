// Package storage abstracts S3-compatible object storage (MinIO locally).
package storage

import (
	"context"
	"io"
)

// ObjectStore stores and retrieves file blobs.
type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// NopStore is a no-op ObjectStore used until MinIO wiring is complete.
type NopStore struct{}

func (NopStore) Put(context.Context, string, io.Reader, int64, string) error { return nil }

func (NopStore) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, &NotConfigured{Op: "Get"}
}

func (NopStore) Delete(context.Context, string) error { return nil }

// NotConfigured indicates object storage is not yet wired.
type NotConfigured struct {
	Op string
}

func (e *NotConfigured) Error() string {
	if e.Op == "" {
		return "storage: not configured"
	}
	return "storage: not configured: " + e.Op
}
