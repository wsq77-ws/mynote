package models

// LLMConfigResponse LLM 配置响应（api_key 脱敏）
type LLMConfigResponse struct {
	Provider     string  `json:"provider"`
	APIKey       string  `json:"api_key"` // 脱敏：****1234
	BaseURL      string  `json:"base_url"`
	Model        string  `json:"model"`
	MaxTokens    int     `json:"max_tokens"`
	Temperature  float64 `json:"temperature"`
	SystemPrompt string  `json:"system_prompt"`
	Configured   bool    `json:"configured"`
}

// UpdateLLMConfigRequest 更新 LLM 配置请求
// 所有字段为指针类型，支持部分更新；nil 表示不更新该字段
type UpdateLLMConfigRequest struct {
	APIKey       *string  `json:"api_key,omitempty"`
	BaseURL      *string  `json:"base_url,omitempty"`
	Model        *string  `json:"model,omitempty"`
	MaxTokens    *int     `json:"max_tokens,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	SystemPrompt *string  `json:"system_prompt,omitempty"`
}

// CompleteRequest 自动补全请求
type CompleteRequest struct {
	Text string `json:"text" binding:"required"`
}

// CompleteResponse 自动补全响应
type CompleteResponse struct {
	Suggestion string `json:"suggestion"`
}

// GenerateRequest 生成笔记请求
type GenerateRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// GenerateResponse 生成笔记响应
type GenerateResponse struct {
	Content string `json:"content"`
}

// SummarizeResponse 总结响应
type SummarizeResponse struct {
	Path      string `json:"path"`
	NoteCount int    `json:"note_count"`
}
