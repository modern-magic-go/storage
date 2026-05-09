package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func ComputeETag(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
