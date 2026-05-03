package storage

import "errors"

var (
	ErrAdapterNotFound    = errors.New("storage adapter not found")
	ErrBucketNotFound     = errors.New("storage bucket not found")
	ErrFileTooLarge       = errors.New("storage file too large")
	ErrFileNotFound       = errors.New("storage file not found")
	ErrFileTypeNotAllowed = errors.New("storage file type not allowed")
	ErrInvalidConfig      = errors.New("invalid storage config")
	ErrOperationFailed    = errors.New("storage operation failed")
)
