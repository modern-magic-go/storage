package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func setupTestAdapter(t *testing.T) (*LocalAdapter, func()) {
	tmpDir := filepath.Join(os.TempDir(), "storage-test")

	adapter, err := NewLocalAdapter(tmpDir)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return adapter, cleanup
}

func TestNewLocalAdapter(t *testing.T) {
	t.Run("empty path uses default", func(t *testing.T) {
		adapter, err := NewLocalAdapter("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if adapter.basePath != "var/tmp/storage" {
			t.Errorf("expected default path, got %s", adapter.basePath)
		}
	})

	t.Run("custom path", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "storage-custom")
		defer os.RemoveAll(tmpDir)

		adapter, err := NewLocalAdapter(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if adapter.basePath != tmpDir {
			t.Errorf("expected %s, got %s", tmpDir, adapter.basePath)
		}
	})
}

func TestPutAndGet(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/file.txt"
	content := []byte("Hello, World!")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
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

func TestGetNotFound(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	_, err := adapter.Get(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestDelete(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/delete.txt"
	content := []byte("to be deleted")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
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

func TestDeleteNotFound(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	err := adapter.Delete(ctx, "nonexistent.txt")
	if err != nil {
		t.Errorf("Delete should not error for nonexistent file: %v", err)
	}
}

func TestExists(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/exists.txt"
	content := []byte("exists")

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

func TestStat(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/stat.txt"
	content := []byte("stat test")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
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

func TestStatNotFound(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	_, err := adapter.Stat(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestMimeTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"image.gif", "image/gif"},
		{"image.webp", "image/webp"},
		{"document.pdf", "application/pdf"},
		{"document.doc", "application/msword"},
		{"document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectMimeType(tt.filename)
			if got != tt.expected {
				t.Errorf("detectMimeType(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, &os.PathError{Op: "read", Path: "mock", Err: os.ErrPermission}
}

func TestPutErrorScenarios(t *testing.T) {
	t.Run("reader returns error", func(t *testing.T) {
		adapter, cleanup := setupTestAdapter(t)
		defer cleanup()

		ctx := context.Background()
		key := "test/error-reader.txt"

		err := adapter.Put(ctx, key, &errorReader{}, 100)
		if err == nil {
			t.Error("expected error when reader fails")
		}
	})

	t.Run("invalid path characters", func(t *testing.T) {
		adapter, cleanup := setupTestAdapter(t)
		defer cleanup()

		ctx := context.Background()
		key := "test/invalid\x00path.txt"

		err := adapter.Put(ctx, key, bytes.NewReader([]byte("test")), 4)
		if err == nil {
			t.Error("expected error with invalid path characters")
		}
	})

	t.Run("path component is file not directory", func(t *testing.T) {
		adapter, cleanup := setupTestAdapter(t)
		defer cleanup()

		ctx := context.Background()

		firstKey := "testfile.txt"
		err := adapter.Put(ctx, firstKey, bytes.NewReader([]byte("content")), 7)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		secondKey := "testfile.txt/subdir/file.txt"
		err = adapter.Put(ctx, secondKey, bytes.NewReader([]byte("content")), 7)
		if err == nil {
			t.Error("expected error when path component is file not directory")
		}
	})
}

func TestPutReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - chmod doesn't restrict permissions")
	}

	tmpDir := filepath.Join(os.TempDir(), "storage-readonly-test")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Chmod(tmpDir, 0o555); err != nil {
		t.Fatalf("failed to make dir read-only: %v", err)
	}

	adapter, err := NewLocalAdapter(tmpDir)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()
	key := "subdir/test.txt"

	err = adapter.Put(ctx, key, bytes.NewReader([]byte("test")), 4)
	if err == nil {
		t.Error("expected error when creating directory in read-only parent")
	}

	os.Chmod(tmpDir, 0o755)
}

func TestGetPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - chmod doesn't restrict permissions")
	}

	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/permission.txt"
	content := []byte("test content")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	fullPath := filepath.Join(adapter.basePath, key)
	if err := os.Chmod(fullPath, 0o000); err != nil {
		t.Fatalf("cannot set file permissions: %v", err)
	}
	defer os.Chmod(fullPath, 0o644)

	_, err = adapter.Get(ctx, key)
	if err == nil {
		t.Error("expected error when reading file without permission")
	}
}

func TestDeletePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - chmod doesn't restrict permissions")
	}

	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/delete-permission.txt"
	content := []byte("test content")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	parentDir := filepath.Join(adapter.basePath, "test")
	if err := os.Chmod(parentDir, 0o555); err != nil {
		t.Fatalf("cannot set directory permissions: %v", err)
	}
	defer os.Chmod(parentDir, 0o755)

	err = adapter.Delete(ctx, key)
	if err == nil {
		t.Error("expected error when deleting file from read-only directory")
	}
}

func TestStatPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - chmod doesn't restrict permissions")
	}

	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/stat-permission.txt"
	content := []byte("test content")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	fullPath := filepath.Join(adapter.basePath, key)
	if err := os.Chmod(fullPath, 0o000); err != nil {
		t.Fatalf("cannot set file permissions: %v", err)
	}
	defer os.Chmod(fullPath, 0o644)

	_, err = adapter.Stat(ctx, key)
	if err == nil {
		t.Error("expected error when stating file without permission")
	}
}

func TestExistsErrorScenario(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - chmod doesn't restrict permissions")
	}

	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test/exists-permission.txt"
	content := []byte("test content")

	err := adapter.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	parentDir := filepath.Join(adapter.basePath, "test")
	if err := os.Chmod(parentDir, 0o000); err != nil {
		t.Fatalf("cannot set directory permissions: %v", err)
	}
	defer os.Chmod(parentDir, 0o755)

	_, err = adapter.Exists(ctx, key)
	if err == nil {
		t.Error("expected error when checking existence of file in inaccessible directory")
	}
}

func TestGetOtherErrors(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	key := ""
	for i := 0; i < 300; i++ {
		key += "a"
	}
	key += ".txt"

	_, err := adapter.Get(ctx, key)
	if err == nil {
		t.Error("expected error when accessing file with very long path")
	}
}

func TestDeleteOtherErrors(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	key := ""
	for i := 0; i < 300; i++ {
		key += "a"
	}
	key += ".txt"

	err := adapter.Delete(ctx, key)
	if err == nil {
		t.Error("expected error when deleting file with very long path")
	}
}

func TestStatOtherErrors(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	key := ""
	for i := 0; i < 300; i++ {
		key += "a"
	}
	key += ".txt"

	_, err := adapter.Stat(ctx, key)
	if err == nil {
		t.Error("expected error when stating file with very long path")
	}
}

func TestExistsOtherErrors(t *testing.T) {
	adapter, cleanup := setupTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	key := ""
	for i := 0; i < 300; i++ {
		key += "a"
	}
	key += ".txt"

	_, err := adapter.Exists(ctx, key)
	if err == nil {
		t.Error("expected error when checking existence of file with very long path")
	}
}
