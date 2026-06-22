package storage

// Config 存储配置
type Config struct {
	Storage StorageConfig `yaml:"storage"`
}

// StorageConfig 存储后端配置
type StorageConfig struct {
	// Type 存储类型: "local" 或 "oss"
	Type string `yaml:"type"`

	// Local 本地文件系统配置
	Local LocalConfig `yaml:"local"`

	// OSS 对象存储配置
	OSS OSSConfig `yaml:"oss"`
}

// LocalConfig 本地文件系统配置
type LocalConfig struct {
	// DataDir 数据目录路径
	DataDir string `yaml:"data_dir"`
}

// OSSConfig 对象存储配置（兼容 S3 协议）
type OSSConfig struct {
	// Endpoint 服务端点，如 https://s3.amazonaws.com 或 http://minio:9000
	Endpoint string `yaml:"endpoint"`

	// AccessKey 访问密钥
	AccessKey string `yaml:"access_key"`

	// SecretKey 秘密密钥
	SecretKey string `yaml:"secret_key"`

	// Bucket 存储桶名称
	Bucket string `yaml:"bucket"`

	// Region 区域
	Region string `yaml:"region"`

	// Prefix 键前缀，所有笔记将存储在此前缀下
	Prefix string `yaml:"prefix"`

	// UseSSL 是否使用 SSL
	UseSSL bool `yaml:"use_ssl"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Storage: StorageConfig{
			Type: "local",
			Local: LocalConfig{
				DataDir: "./data",
			},
		},
	}
}
