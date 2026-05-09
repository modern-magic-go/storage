package storage

type Config struct {
	Adapter     string
	StoragePath string
	Buckets     []BucketConf
}

type BucketConf struct {
	Name         string
	MaxFileSize  int64
	AllowedTypes []string
}
