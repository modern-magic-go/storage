[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

# storage - Go 统一存储适配器

`storage` 为多种存储后端提供单一统一接口，包括本地文件系统、MinIO 和阿里云 OSS。

## 特性

- 门面模式（Facade）设计，隐藏后端复杂度
- 可插拔适配器架构
- 统一的存储操作 API
- 全操作支持 `context.Context`

## 安装

```bash
go get github.com/modern-magic-go/storage
```

## 快速开始

推荐使用 `storage.New` 创建 `*Client` 门面实例。
`storage.NewAdapter` 保留用于向后兼容。

```go
package main

import (
    "context"
    "fmt"
    "io"
    "strings"

    "github.com/modern-magic-go/storage"
)

func main() {
    client, err := storage.New(storage.Config{
        Adapter:     "local",
        StoragePath: "./data",
    })
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 上传文件
    err = client.Put(ctx, "hello.txt", strings.NewReader("hello storage"), int64(len("hello storage")))
    if err != nil {
        panic(err)
    }

    // 检查是否存在
    exists, err := client.Exists(ctx, "hello.txt")
    if err != nil {
        panic(err)
    }
    fmt.Println("exists:", exists)

    // 获取文件信息
    info, err := client.Stat(ctx, "hello.txt")
    if err != nil {
        panic(err)
    }
    fmt.Println("size:", info.Size)

    // 读取文件
    reader, err := client.Get(ctx, "hello.txt")
    if err != nil {
        panic(err)
    }
    defer reader.Close()

    content, err := io.ReadAll(reader)
    if err != nil {
        panic(err)
    }
    fmt.Println(string(content))

    // 删除文件
    err = client.Delete(ctx, "hello.txt")
    if err != nil {
        panic(err)
    }
}
```

## API

`*Client` 门面提供以下方法：

| 方法 | 说明 |
| --- | --- |
| `New(cfg Config) (*Client, error)` | 创建存储客户端 |
| `Put(ctx, key, reader, size) error` | 上传文件 |
| `Get(ctx, key) (io.ReadCloser, error)` | 读取文件 |
| `Delete(ctx, key) error` | 删除文件 |
| `Stat(ctx, key) (FileInfo, error)` | 获取文件元信息 |
| `Exists(ctx, key) (bool, error)` | 检查文件是否存在 |
| `Close() error` | 释放资源 |

底层 `StorageAdapter` 接口也对外导出，供高级场景使用：

- `NewAdapter(cfg Config) (StorageAdapter, error)` — 返回原始适配器（向后兼容）

## Bucket 校验

通过 `Config.Buckets` 可为不同逻辑 Bucket 配置上传限制，key 的第一段路径作为 Bucket 名称匹配（如 `avatars/user123.jpg` → bucket = `avatars`）。

```go
client, _ := storage.New(storage.Config{
    Adapter:     "local",
    StoragePath: "./data",
    Buckets: []storage.BucketConf{
        {
            Name:         "avatars",
            MaxFileSize:  5 * 1024 * 1024,       // 5MB
            AllowedTypes: []string{"jpg", "png"}, // 仅允许图片
        },
        {
            Name:         "docs",
            MaxFileSize:  50 * 1024 * 1024,      // 50MB
            AllowedTypes: []string{"pdf", "doc", "docx"},
        },
    },
})

// 匹配 "avatars" bucket，触发校验
client.Put(ctx, "avatars/photo.jpg", reader, size)  // ✓ 通过
client.Put(ctx, "avatars/video.mp4", reader, size)  // ✗ ErrFileTypeNotAllowed
```

| 字段 | 说明 |
| --- | --- |
| `Name` | Bucket 名称，匹配 key 的第一段路径 |
| `MaxFileSize` | 最大文件大小（字节），0 表示不限制 |
| `AllowedTypes` | 允许的扩展名列表（不含点），空表示不限制 |
| `Access` | 访问策略标记（`public_read` / `authenticated`），当前作元数据用途 |

未配置 Buckets 或 key 未匹配任何 Bucket 时，跳过校验（向后兼容）。

## 支持的存储后端

| 适配器 | 状态 |
| --- | --- |
| local（本地文件系统） | ✓ 可用 |
| minio | 待实现 |
| oss（阿里云对象存储） | 待实现 |

## 项目结构

包采用门面模式，公共接口保持精简，后端具体实现隐藏在 `internal/` 目录下。

```text
storage/
├── storage.go        # 公共门面 (*Client)
├── adapter.go        # StorageAdapter 接口 + FileInfo
├── config.go         # Config + BucketConf 配置结构
├── errors.go         # 哨兵错误定义
├── storage_test.go   # 测试
├── go.mod
├── LICENSE
├── README.md
└── internal/
    ├── adapter/
    │   ├── local/    # 本地文件系统适配器
    │   ├── minio/    # MinIO 适配器（待实现）
    │   └── oss/      # 阿里云 OSS 适配器（待实现）
    └── utils/        # 内部工具（hash 等）
```

- 根包：公共门面、配置、类型定义和错误
- `internal/`：私有实现细节，外部不可见

## License

本项目采用 MIT 许可证，详见 `LICENSE`。
