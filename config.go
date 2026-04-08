package storage

// Config 存储配置
type Config struct {
	Adapter     string       // 适配器类型：local, minio, oss
	StoragePath string       // 本地存储路径（adapter=local 时使用）
	Buckets     []BucketConf // Bucket 配置列表
}

// BucketConf 逻辑 Bucket 配置
type BucketConf struct {
	Name         string   // Bucket 名称
	Access       string   // 访问策略：public_read, authenticated
	MaxFileSize  int64    // 最大文件大小（字节）
	AllowedTypes []string // 允许的文件类型（扩展名列表）
}
