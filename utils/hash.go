package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// CalculateHash 计算 io.Reader 的 SHA256 哈希值
// 支持流式计算，适用于大文件
func CalculateHash(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
