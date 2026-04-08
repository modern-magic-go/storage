package storage

import (
	"context"
	"fmt"
	"io"

	"app/pkg/storage/local"
)

// NewAdapter 根据配置创建存储适配器实例
func NewAdapter(cfg Config) (StorageAdapter, error) {
	switch cfg.Adapter {
	case "local":
		adapter, err := local.NewLocalAdapter(cfg.StoragePath)
		if err != nil {
			return nil, err
		}
		return &localAdapterWrapper{adapter: adapter}, nil
	case "minio":
		// TODO: 实现 Minio 适配器
		return nil, fmt.Errorf("minio adapter not implemented yet")
	case "oss":
		// TODO: 实现阿里云 OSS 适配器
		return nil, fmt.Errorf("oss adapter not implemented yet")
	default:
		return nil, fmt.Errorf("unknown storage adapter: %s", cfg.Adapter)
	}
}

// localAdapterWrapper 包装 local.LocalAdapter 以实现 StorageAdapter 接口
type localAdapterWrapper struct {
	adapter *local.LocalAdapter
}

func (w *localAdapterWrapper) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	return w.adapter.Put(ctx, key, reader, size)
}

func (w *localAdapterWrapper) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return w.adapter.Get(ctx, key)
}

func (w *localAdapterWrapper) Delete(ctx context.Context, key string) error {
	return w.adapter.Delete(ctx, key)
}

func (w *localAdapterWrapper) Stat(ctx context.Context, key string) (FileInfo, error) {
	localInfo, err := w.adapter.Stat(ctx, key)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Key:       localInfo.Key,
		Size:      localInfo.Size,
		MimeType:  localInfo.MimeType,
		CreatedAt: localInfo.CreatedAt,
	}, nil
}

func (w *localAdapterWrapper) Exists(ctx context.Context, key string) (bool, error) {
	return w.adapter.Exists(ctx, key)
}
