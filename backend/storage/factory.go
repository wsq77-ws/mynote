package storage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// New 根据配置创建对应的存储后端实例
func New(cfg *Config) (Storage, error) {
	switch cfg.Storage.Type {
	case "local", "":
		dataDir := cfg.Storage.Local.DataDir
		if dataDir == "" {
			dataDir = "./data"
		}
		return NewLocalStorage(dataDir)
	case "oss":
		return NewOSSStorage(cfg.Storage.OSS)
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.Storage.Type)
	}
}

// LoadConfig 从 YAML 文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}
