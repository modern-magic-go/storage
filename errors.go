package storage

import "errors"

var (
	ErrInvalidConfig      = errors.New("invalid storage config")
	ErrFileNotFound       = errors.New("file not found")
	ErrFileTooLarge       = errors.New("file too large")
	ErrFileTypeNotAllowed = errors.New("file type not allowed")
	ErrOperationFailed    = errors.New("storage operation failed")
	ErrBucketNotFound     = errors.New("bucket not found")
	ErrAdapterNotFound    = errors.New("adapter not found")
)
