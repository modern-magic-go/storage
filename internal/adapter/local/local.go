package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/modern-magic-go/storage/internal/utils"
)

type LocalAdapter struct {
	basePath string
}

func NewLocalAdapter(basePath string) (*LocalAdapter, error) {
	if basePath == "" {
		basePath = "var/tmp/storage"
	}

	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create storage path: %w", err)
	}

	info, err := os.Stat(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat storage path: %w", err)
	}

	if info.Mode()&0o200 == 0 {
		return nil, fmt.Errorf("storage path not writable")
	}

	return &LocalAdapter{basePath: basePath}, nil
}

func (a *LocalAdapter) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	fullPath := filepath.Join(a.basePath, key)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (a *LocalAdapter) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(a.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (a *LocalAdapter) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(a.basePath, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (a *LocalAdapter) Stat(ctx context.Context, key string) (FileInfo, error) {
	fullPath := filepath.Join(a.basePath, key)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, fmt.Errorf("file not found: %s", key)
		}
		return FileInfo{}, fmt.Errorf("failed to stat file: %w", err)
	}

	mimeType := detectMimeType(info.Name())

	etag, err := computeFileETag(fullPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to compute etag for %s: %w", key, err)
	}

	return FileInfo{
		Key:       key,
		Size:      info.Size(),
		MimeType:  mimeType,
		CreatedAt: info.ModTime(),
		ETag:      etag,
	}, nil
}

func computeFileETag(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return utils.ComputeETag(f)
}

// FileInfo 文件信息
type FileInfo struct {
	Key       string
	Size      int64
	MimeType  string
	CreatedAt time.Time
	ETag      string
}

func (a *LocalAdapter) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(a.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file: %w", err)
	}

	return true, nil
}

func detectMimeType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}
