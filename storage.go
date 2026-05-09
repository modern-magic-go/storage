package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/modern-magic-go/storage/internal/adapter/local"
)

// Client is the public facade for storage operations.
type Client struct {
	adapter StorageAdapter
	buckets map[string]BucketConf
}

// New creates a storage client from public configuration.
func New(cfg Config) (*Client, error) {
	adapter, err := newAdapter(cfg)
	if err != nil {
		return nil, err
	}

	var buckets map[string]BucketConf
	if len(cfg.Buckets) > 0 {
		buckets = make(map[string]BucketConf, len(cfg.Buckets))
		for _, bucket := range cfg.Buckets {
			buckets[bucket.Name] = bucket
		}
	}

	return &Client{adapter: adapter, buckets: buckets}, nil
}

// NewAdapter preserves the legacy adapter-based constructor.
func NewAdapter(cfg Config) (StorageAdapter, error) {
	return newAdapter(cfg)
}

// Put stores content at the given key.
func (c *Client) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	if bucket := c.resolveBucket(key); bucket != nil {
		if bucket.MaxFileSize > 0 && size > bucket.MaxFileSize {
			return fmt.Errorf("file size %d exceeds bucket %q limit %d: %w", size, bucket.Name, bucket.MaxFileSize, ErrFileTooLarge)
		}

		if len(bucket.AllowedTypes) > 0 {
			ext := extractExtension(key)
			if !isTypeAllowed(ext, bucket.AllowedTypes) {
				return fmt.Errorf("file type %q not allowed for bucket %q: %w", ext, bucket.Name, ErrFileTypeNotAllowed)
			}
		}
	}

	if err := c.adapter.Put(ctx, key, reader, size); err != nil {
		return wrapOperationError("put", key, err)
	}
	return nil
}

// Get opens content for the given key.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := c.adapter.Get(ctx, key)
	if err != nil {
		return nil, wrapOperationError("get", key, err)
	}
	return reader, nil
}

// Delete removes content for the given key.
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.adapter.Delete(ctx, key); err != nil {
		return wrapOperationError("delete", key, err)
	}
	return nil
}

// Stat returns metadata for the given key.
func (c *Client) Stat(ctx context.Context, key string) (FileInfo, error) {
	info, err := c.adapter.Stat(ctx, key)
	if err != nil {
		return FileInfo{}, wrapOperationError("stat", key, err)
	}
	return info, nil
}

// Exists reports whether content exists for the given key.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := c.adapter.Exists(ctx, key)
	if err != nil {
		return false, wrapOperationError("exists", key, err)
	}
	return exists, nil
}

// Close releases adapter resources when supported.
func (c *Client) Close() error {
	if closer, ok := c.adapter.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("close storage client: %w", ErrOperationFailed)
		}
	}
	return nil
}

func (c *Client) resolveBucket(key string) *BucketConf {
	if len(c.buckets) == 0 {
		return nil
	}

	bucketName, _, _ := strings.Cut(key, "/")
	bucket, ok := c.buckets[bucketName]
	if !ok {
		return nil
	}

	return &bucket
}

// extractExtension returns the file extension (without dot) from a key.
// Returns empty string if no extension found.
func extractExtension(key string) string {
	idx := strings.LastIndex(key, ".")
	if idx == -1 || idx == len(key)-1 {
		return ""
	}
	return strings.ToLower(key[idx+1:])
}

// isTypeAllowed checks if ext is in the allowedTypes list (case-insensitive).
func isTypeAllowed(ext string, allowedTypes []string) bool {
	for _, t := range allowedTypes {
		if strings.EqualFold(ext, t) {
			return true
		}
	}
	return false
}

func newAdapter(cfg Config) (StorageAdapter, error) {
	if strings.TrimSpace(cfg.Adapter) == "" {
		return nil, fmt.Errorf("unknown storage adapter: %w", ErrAdapterNotFound)
	}

	switch cfg.Adapter {
	case "local":
		adapter, err := local.NewLocalAdapter(cfg.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("create local adapter: %w", ErrInvalidConfig)
		}
		return &localAdapterWrapper{adapter: adapter}, nil
	case "minio", "oss":
		return nil, fmt.Errorf("adapter %q is not implemented yet: %w", cfg.Adapter, ErrAdapterNotFound)
	default:
		return nil, fmt.Errorf("unknown storage adapter %q: %w", cfg.Adapter, ErrAdapterNotFound)
	}
}

type localAdapterWrapper struct {
	adapter *local.LocalAdapter
}

func (a *localAdapterWrapper) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	return a.adapter.Put(ctx, key, reader, size)
}

func (a *localAdapterWrapper) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return a.adapter.Get(ctx, key)
}

func (a *localAdapterWrapper) Delete(ctx context.Context, key string) error {
	return a.adapter.Delete(ctx, key)
}

func (a *localAdapterWrapper) Stat(ctx context.Context, key string) (FileInfo, error) {
	info, err := a.adapter.Stat(ctx, key)
	if err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Key:       info.Key,
		Size:      info.Size,
		MimeType:  info.MimeType,
		CreatedAt: info.CreatedAt,
		ETag:      info.ETag,
	}, nil
}

func (a *localAdapterWrapper) Exists(ctx context.Context, key string) (bool, error) {
	return a.adapter.Exists(ctx, key)
}

func wrapOperationError(operation, key string, err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "file not found") {
		return fmt.Errorf("%s %q: %w", operation, key, ErrFileNotFound)
	}

	return fmt.Errorf("%s %q: %w", operation, key, ErrOperationFailed)
}
