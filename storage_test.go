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
