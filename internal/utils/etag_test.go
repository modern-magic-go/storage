package utils

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestComputeETag(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "hello world",
			content:  "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "test content",
			content:  "test content for hash",
			expected: "ee63779c8078ead9b4c682d6bd8662edac699d851ed85cd120d3236609bb3253",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			got, err := ComputeETag(reader)
			if err != nil {
				t.Fatalf("ComputeETag failed: %v", err)
			}

			if len(got) != 64 {
				t.Errorf("expected hash length 64, got %d", len(got))
			}

			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestComputeETag_Consistency(t *testing.T) {
	content := "consistent content"
	expected := "1604d02cf3ce7a673838d7e644fa9f4e7d0490844b4e266fe117740afb9bf228"

	reader1 := strings.NewReader(content)
	hash1, err := ComputeETag(reader1)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}

	reader2 := strings.NewReader(content)
	hash2, err := ComputeETag(reader2)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("hash inconsistency: %s != %s", hash1, hash2)
	}

	if hash1 != expected {
		t.Errorf("expected %q, got %q", expected, hash1)
	}
}

func TestComputeETag_Stream(t *testing.T) {
	content := []byte("streaming content")

	reader := bytes.NewReader(content)
	hash, err := ComputeETag(reader)
	if err != nil {
		t.Fatalf("ComputeETag failed: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}
}

type etagErrorReader struct{}

func (r *etagErrorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestComputeETag_ReaderError(t *testing.T) {
	reader := &etagErrorReader{}

	hash, err := ComputeETag(reader)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if hash != "" {
		t.Errorf("expected empty string on error, got %q", hash)
	}
}
