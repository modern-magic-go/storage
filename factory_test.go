package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAdapter(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		wantErr     bool
		errContains string
		wantWrapper bool
	}{
		{
			name: "local adapter success",
			cfg: Config{
				Adapter:     "local",
				StoragePath: filepath.Join(os.TempDir(), "storage-factory-test"),
			},
			wantErr:     false,
			wantWrapper: true,
		},
		{
			name: "local adapter with empty path uses default",
			cfg: Config{
				Adapter:     "local",
				StoragePath: "",
			},
			wantErr:     false,
			wantWrapper: true,
		},
		{
			name: "minio adapter not implemented",
			cfg: Config{
				Adapter: "minio",
			},
			wantErr:     true,
			errContains: "not implemented yet",
		},
		{
			name: "oss adapter not implemented",
			cfg: Config{
				Adapter: "oss",
			},
			wantErr:     true,
			errContains: "not implemented yet",
		},
		{
			name: "unknown adapter type",
			cfg: Config{
				Adapter: "unknown",
			},
			wantErr:     true,
			errContains: "unknown storage adapter",
		},
		{
			name: "empty adapter type",
			cfg: Config{
				Adapter: "",
			},
			wantErr:     true,
			errContains: "unknown storage adapter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.StoragePath != "" {
				defer os.RemoveAll(tt.cfg.StoragePath)
			}

			adapter, err := NewAdapter(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				if adapter != nil {
					t.Error("expected nil adapter on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if adapter == nil {
					t.Fatal("expected non-nil adapter")
				}
				if tt.wantWrapper {
					if _, ok := adapter.(*localAdapterWrapper); !ok {
						t.Errorf("expected *localAdapterWrapper, got %T", adapter)
					}
				}
			}
		})
	}
}

func TestLocalAdapterWrapper_Put(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-put-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "test/put.txt"
	content := []byte("test content for put")

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	filePath := filepath.Join(tmpDir, key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("file was not created by underlying adapter")
	}
}

func TestLocalAdapterWrapper_Get(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-get-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "test/get.txt"
	content := []byte("test content for get")

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	reader, err := adapter.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch: expected %q, got %q", content, buf.Bytes())
	}
}

func TestLocalAdapterWrapper_GetNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-get-notfound-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()

	_, err = adapter.Get(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLocalAdapterWrapper_Delete(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-delete-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "test/delete.txt"
	content := []byte("to be deleted")

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	err = adapter.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = adapter.Get(ctx, key)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestLocalAdapterWrapper_DeleteNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-delete-notfound-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()

	err = adapter.Delete(ctx, "nonexistent.txt")
	if err != nil {
		t.Errorf("Delete should not error for nonexistent file: %v", err)
	}
}

func TestLocalAdapterWrapper_Stat(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-stat-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "test/stat.txt"
	content := []byte("stat test content")

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := adapter.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Key != key {
		t.Errorf("key mismatch: expected %s, got %s", key, info.Key)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("size mismatch: expected %d, got %d", len(content), info.Size)
	}
}

func TestLocalAdapterWrapper_StatNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-stat-notfound-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()

	_, err = adapter.Stat(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLocalAdapterWrapper_Exists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-exists-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "test/exists.txt"
	content := []byte("exists test")

	exists, err := adapter.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("file should not exist yet")
	}

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err = adapter.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("file should exist after Put")
	}
}

func TestLocalAdapterWrapper_ExistsNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-exists-notfound-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()

	exists, err := adapter.Exists(ctx, "nonexistent.txt")
	if err != nil {
		t.Fatalf("Exists should not error: %v", err)
	}
	if exists {
		t.Error("nonexistent file should not exist")
	}
}

func TestLocalAdapterWrapper_Integration(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-integration-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "integration/workflow.txt"
	content := []byte("integration test content")

	exists, err := adapter.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("file should not exist initially")
	}

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err = adapter.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("file should exist after Put")
	}

	info, err := adapter.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("size mismatch: expected %d, got %d", len(content), info.Size)
	}

	func() {
		reader, err := adapter.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		defer reader.Close()

		var buf bytes.Buffer
		_, err = buf.ReadFrom(reader)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), content) {
			t.Errorf("content mismatch: expected %q, got %q", content, buf.Bytes())
		}
	}()

	err = adapter.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err = adapter.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("file should not exist after delete")
	}
}

func TestLocalAdapterWrapper_StatFileInfoConversion(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-wrapper-stat-conversion-test")
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "test/conversion.txt"
	content := []byte("conversion test")

	err = adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := adapter.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Key != key {
		t.Errorf("Key mismatch: expected %s, got %s", key, info.Key)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("Size mismatch: expected %d, got %d", len(content), info.Size)
	}
	if info.MimeType == "" {
		t.Error("MimeType should not be empty")
	}
	if info.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}
