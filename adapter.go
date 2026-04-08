package storage

import (
	"context"
	"io"
	"time"
)

type FileInfo struct {
	Key       string
	Size      int64
	MimeType  string
	CreatedAt time.Time
}

type StorageAdapter interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (FileInfo, error)
	Exists(ctx context.Context, key string) (bool, error)
}
