package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config LLM 运行时配置
type Config struct {
	Provider     string  `json:"provider"`    // 提供商标识，默认 "openai-compatible"
	APIKey       string  `json:"api_key"`     // API 密钥
	BaseURL      string  `json:"base_url"`    // 接口地址，如 https://api.deepseek.com
	Model        string  `json:"model"`       // 模型名，如 deepseek-v4-pro
	MaxTokens    int     `json:"max_tokens"`  // 全局最大生成 token 数，默认 512，作用于所有端点
	Temperature  float64 `json:"temperature"` // 采样温度 0~2，默认 0.7
	SystemPrompt string  `json:"-"`           // 系统提示词（单独文件存储，不序列化到 secret_key.json）
}

// secretKeyFileName 密钥配置文件名
const secretKeyFileName = "secret_key.json"

// systemPromptFileName 系统提示词文件名
const systemPromptFileName = "system_prompt.md"

// defaultSystemPrompt 默认系统提示词（文件不存在时使用，开箱即用）
const defaultSystemPrompt = "你是一个知识笔记助手。请根据用户的输入提供简洁、准确的回答。"

// DefaultProvider 默认提供商标识
const DefaultProvider = "openai-compatible"

// DefaultMaxTokens 默认最大生成 token 数（全局，作用于补全/生成/总结）
const DefaultMaxTokens = 512

// DefaultTemperature 默认采样温度
const DefaultTemperature = 0.7

// maxModelTokensUpperBound max_tokens 合法上界（防止过大请求拖垮服务）
const maxModelTokensUpperBound = 8192

// LoadConfig 读取 secret_key.json + system_prompt.md
// 文件不存在返回空配置（非错误），并补齐默认 provider、默认 system_prompt 与默认模型参数
func LoadConfig(dir string) (*Config, error) {
	cfg := &Config{
		Provider:     DefaultProvider,
		MaxTokens:    DefaultMaxTokens,
		Temperature:  DefaultTemperature,
		SystemPrompt: defaultSystemPrompt,
	}

	// 读取密钥配置
	secretPath := filepath.Join(dir, secretKeyFileName)
	if data, err := os.ReadFile(secretPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", secretPath, err)
		}
		if cfg.Provider == "" {
			cfg.Provider = DefaultProvider
		}
		// 兼容旧配置文件（无 max_tokens/temperature 字段时回填默认值）
		if cfg.MaxTokens <= 0 {
			cfg.MaxTokens = DefaultMaxTokens
		}
		if cfg.Temperature <= 0 {
			cfg.Temperature = DefaultTemperature
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取 %s 失败: %w", secretPath, err)
	}

	// 读取系统提示词（覆盖默认值，文件存在时以文件内容为准）
	promptPath := filepath.Join(dir, systemPromptFileName)
	if data, err := os.ReadFile(promptPath); err == nil {
		cfg.SystemPrompt = string(data)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取 %s 失败: %w", promptPath, err)
	}

	return cfg, nil
}

// SaveSecretConfig 写入 secret_key.json，文件权限 0600
// SystemPrompt 字段不参与序列化（json:"-"）
func SaveSecretConfig(dir string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	secretPath := filepath.Join(dir, secretKeyFileName)
	if err := os.WriteFile(secretPath, data, 0600); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", secretPath, err)
	}
	return nil
}

// SaveSystemPrompt 写入 system_prompt.md（纯文本）
func SaveSystemPrompt(dir, prompt string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	promptPath := filepath.Join(dir, systemPromptFileName)
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", promptPath, err)
	}
	return nil
}

// MaskedConfig 返回 api_key 仅保留后 4 位、前缀 **** 的脱敏配置副本
func MaskedConfig(cfg *Config) *Config {
	masked := *cfg
	masked.APIKey = maskKey(cfg.APIKey)
	return &masked
}

// maskKey 对 api_key 脱敏：长度 <=4 全部掩码，否则保留后 4 位
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// IsMaskedKey 判断 api_key 是否为脱敏值（以 **** 开头）
// 用于 PUT /config 时忽略脱敏值回写
func IsMaskedKey(key string) bool {
	return strings.HasPrefix(key, "****")
}

// ValidateBaseURL 校验 base_url 必须以 http:// 或 https:// 开头
// 禁止 file:// 等协议，防止 SSRF 风险
func ValidateBaseURL(baseURL string) error {
	if baseURL == "" {
		return nil // 允许为空，由前端/默认值处理
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return fmt.Errorf("base_url 必须以 http:// 或 https:// 开头")
	}
	return nil
}

// ValidateModelParams 校验 max_tokens 与 temperature
// max_tokens: 1 ~ maxModelTokensUpperBound
// temperature: (0, 2]（0 视为未设置，由调用方回填默认值；此处拒绝 ≤0 与 >2）
func ValidateModelParams(maxTokens int, temperature float64) error {
	if maxTokens < 1 || maxTokens > maxModelTokensUpperBound {
		return fmt.Errorf("max_tokens 必须在 1~%d 之间", maxModelTokensUpperBound)
	}
	if temperature <= 0 || temperature > 2 {
		return fmt.Errorf("temperature 必须在 (0, 2] 之间")
	}
	return nil
}

// NewClient 根据配置创建 LLM 客户端
// 扩展：未来可添加 "anthropic"、"ollama" 等 case
func NewClient(cfg *Config) (LLMClient, error) {
	switch cfg.Provider {
	case "", DefaultProvider:
		return NewOpenAICompatibleClient(cfg)
	default:
		return nil, fmt.Errorf("不支持的 LLM 提供商: %s", cfg.Provider)
	}
}
