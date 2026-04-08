package storage

import "errors"

var (
	ErrAdapterNotFound = errors.New("storage adapter not found")
	ErrFileNotFound    = errors.New("storage file not found")
	ErrInvalidConfig   = errors.New("invalid storage config")
	ErrOperationFailed = errors.New("storage operation failed")
)
