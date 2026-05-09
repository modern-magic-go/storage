package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		wantErr     bool
		errContains string
		errIs       error
	}{
		{
			name: "local adapter success",
			cfg: Config{
				Adapter:     "local",
				StoragePath: filepath.Join(os.TempDir(), "storage-client-factory-test"),
			},
		},
		{
			name: "local adapter with empty path uses default",
			cfg: Config{
				Adapter:     "local",
				StoragePath: "",
			},
		},
		{
			name: "minio adapter not available",
			cfg: Config{
				Adapter: "minio",
			},
			wantErr:     true,
			errContains: "not implemented yet",
			errIs:       ErrAdapterNotFound,
		},
		{
			name: "oss adapter not available",
			cfg: Config{
				Adapter: "oss",
			},
			wantErr:     true,
			errContains: "not implemented yet",
			errIs:       ErrAdapterNotFound,
		},
		{
			name: "unknown adapter type",
			cfg: Config{
				Adapter: "unknown",
			},
			wantErr:     true,
			errContains: "unknown storage adapter",
			errIs:       ErrAdapterNotFound,
		},
		{
			name: "empty adapter type",
			cfg: Config{
				Adapter: "",
			},
			wantErr:     true,
			errContains: "unknown storage adapter",
			errIs:       ErrAdapterNotFound,
		},
		{
			name: "whitespace adapter type",
			cfg: Config{
				Adapter: " ",
			},
			wantErr:     true,
			errContains: "unknown storage adapter",
			errIs:       ErrAdapterNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.StoragePath != "" {
				defer os.RemoveAll(tt.cfg.StoragePath)
			}

			client, err := New(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected error to wrap %v, got %v", tt.errIs, err)
				}
				if client != nil {
					t.Error("expected nil client on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if client == nil {
				t.Fatal("expected non-nil client")
			}
		})
	}
}

func TestNewClient_InvalidConfig(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-invalid-config-test")
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	filePath := filepath.Join(tmpDir, "base-path-file")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: filePath,
	})
	if err == nil {
		t.Fatal("expected invalid config error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected error to wrap %v, got %v", ErrInvalidConfig, err)
	}
	if client != nil {
		t.Error("expected nil client on error")
	}
}

func TestClient_Put(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-put-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "test/put.txt"
	content := []byte("test content for put")

	err := client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	filePath := filepath.Join(tmpDir, key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("file was not created by underlying adapter")
	}
}

func TestClient_Put_MaxFileSize(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-put-max-file-size-test")
	defer os.RemoveAll(tmpDir)

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: tmpDir,
		Buckets: []BucketConf{{
			Name:        "docs",
			MaxFileSize: 100,
		}},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Put(context.Background(), "docs/file.txt", bytes.NewReader([]byte("content")), 200)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected error to wrap %v, got %v", ErrFileTooLarge, err)
	}
}

func TestClient_Put_AllowedTypes(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-put-allowed-types-test")
	defer os.RemoveAll(tmpDir)

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: tmpDir,
		Buckets: []BucketConf{{
			Name:         "docs",
			AllowedTypes: []string{"jpg", "png"},
		}},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Put(context.Background(), "docs/file.pdf", bytes.NewReader([]byte("content")), 7)
	if err == nil {
		t.Fatal("expected error for disallowed file type")
	}
	if !errors.Is(err, ErrFileTypeNotAllowed) {
		t.Errorf("expected error to wrap %v, got %v", ErrFileTypeNotAllowed, err)
	}
}

func TestClient_Put_BucketValidationPasses(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-put-bucket-validation-passes-test")
	defer os.RemoveAll(tmpDir)

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: tmpDir,
		Buckets: []BucketConf{{
			Name:         "docs",
			MaxFileSize:  1024,
			AllowedTypes: []string{"txt"},
		}},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	content := []byte("valid content")
	err = client.Put(context.Background(), "docs/file.txt", bytes.NewReader(content), 100)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestClient_Put_NoBucketConfig(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-put-no-bucket-config-test")
	defer os.RemoveAll(tmpDir)

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	content := []byte("no bucket config")
	err = client.Put(context.Background(), "docs/file.pdf", bytes.NewReader(content), 200)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestClient_Put_BucketNotMatched(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-put-bucket-not-matched-test")
	defer os.RemoveAll(tmpDir)

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: tmpDir,
		Buckets: []BucketConf{{
			Name:         "images",
			MaxFileSize:  100,
			AllowedTypes: []string{"jpg", "png"},
		}},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	content := []byte("bucket not matched")
	err = client.Put(context.Background(), "other/file.pdf", bytes.NewReader(content), 200)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestClient_Get(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-get-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "test/get.txt"
	content := []byte("test content for get")

	err := client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	reader, err := client.Get(ctx, key)
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

func TestClient_GetNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-get-notfound-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	_, err := client.Get(context.Background(), "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected error to wrap %v, got %v", ErrFileNotFound, err)
	}
}

func TestClient_Delete(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-delete-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "test/delete.txt"
	content := []byte("to be deleted")

	err := client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	err = client.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = client.Get(ctx, key)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestClient_DeleteNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-delete-notfound-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	err := client.Delete(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Errorf("Delete should not error for nonexistent file: %v", err)
	}
}

func TestClient_Stat(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-stat-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "test/stat.txt"
	content := []byte("stat test content")

	err := client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := client.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Key != key {
		t.Errorf("key mismatch: expected %s, got %s", key, info.Key)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("size mismatch: expected %d, got %d", len(content), info.Size)
	}

	if info.ETag == "" {
		t.Error("ETag should not be empty")
	}
}

func TestClient_StatNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-stat-notfound-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	_, err := client.Stat(context.Background(), "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected error to wrap %v, got %v", ErrFileNotFound, err)
	}
}

func TestClient_Exists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-exists-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "test/exists.txt"
	content := []byte("exists test")

	exists, err := client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("file should not exist yet")
	}

	err = client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err = client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("file should exist after Put")
	}
}

func TestClient_ExistsNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-exists-notfound-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	exists, err := client.Exists(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Fatalf("Exists should not error: %v", err)
	}
	if exists {
		t.Error("nonexistent file should not exist")
	}
}

func TestClient_Close(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-close-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestClient_Integration(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-integration-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "integration/workflow.txt"
	content := []byte("integration test content")

	exists, err := client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("file should not exist initially")
	}

	err = client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err = client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("file should exist after Put")
	}

	info, err := client.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("size mismatch: expected %d, got %d", len(content), info.Size)
	}
	if info.ETag == "" {
		t.Error("ETag should not be empty in integration test")
	}

	func() {
		reader, err := client.Get(ctx, key)
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

	err = client.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err = client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("file should not exist after delete")
	}
}

func TestClient_StatFileInfoConversion(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-stat-conversion-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	key := "test/conversion.txt"
	content := []byte("conversion test")

	err := client.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := client.Stat(ctx, key)
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
	if info.ETag == "" {
		t.Error("ETag should not be empty")
	}
}

func TestClient_ETag_Consistency(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-etag-consistency-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	content := []byte("same content for etag check")

	err := client.Put(ctx, "a/file.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put a failed: %v", err)
	}

	err = client.Put(ctx, "b/file.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put b failed: %v", err)
	}

	infoA, err := client.Stat(ctx, "a/file.txt")
	if err != nil {
		t.Fatalf("Stat a failed: %v", err)
	}

	infoB, err := client.Stat(ctx, "b/file.txt")
	if err != nil {
		t.Fatalf("Stat b failed: %v", err)
	}

	if infoA.ETag != infoB.ETag {
		t.Errorf("ETag mismatch for same content: %s != %s", infoA.ETag, infoB.ETag)
	}
}

func TestClient_ETag_EmptyContent(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-etag-empty-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	content := []byte("")

	err := client.Put(ctx, "empty.txt", bytes.NewReader(content), 0)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := client.Stat(ctx, "empty.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if info.ETag != expected {
		t.Errorf("empty file ETag mismatch: expected %s, got %s", expected, info.ETag)
	}
}

func TestClient_ETag_KnownValue(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-etag-known-test")
	defer os.RemoveAll(tmpDir)

	client := newTestClient(t, tmpDir)

	ctx := context.Background()
	content := []byte("hello world")

	err := client.Put(ctx, "hello.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := client.Stat(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if info.ETag != expected {
		t.Errorf("ETag mismatch for 'hello world': expected %s, got %s", expected, info.ETag)
	}
}

func TestClient_StoragePathRelative(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-relpath-test")
	defer os.RemoveAll(tmpDir)

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	content := []byte("relative path")

	err = client.Put(ctx, "file.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	info, err := client.Stat(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("size mismatch: expected %d, got %d", len(content), info.Size)
	}
}

func TestClient_StoragePathAbsolute(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "storage-client-abspath-test")
	defer os.RemoveAll(tmpDir)

	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: absPath,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	content := []byte("absolute path")

	err = client.Put(ctx, "file.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err := client.Exists(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("file should exist after Put with absolute path")
	}
}

func newTestClient(t *testing.T, storagePath string) *Client {
	t.Helper()

	client, err := New(Config{
		Adapter:     "local",
		StoragePath: storagePath,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return client
}
