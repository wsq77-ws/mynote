package llm

import "context"

// LLMClient 大语言模型客户端接口
// 不同提供商（DeepSeek、OpenAI、Moonshot 等）需实现此接口
type LLMClient interface {
	// Complete 发起一次补全请求
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	// Name 返回客户端标识（如 "openai-compatible"）
	Name() string
}

// CompletionRequest 补全请求
type CompletionRequest struct {
	SystemPrompt string  // 系统提示词（定义模型行为）
	UserPrompt   string  // 用户输入
	MaxTokens    int     // 最大生成 token 数
	Temperature  float64 // 采样温度，0~2
}

// CompletionResponse 补全响应
type CompletionResponse struct {
	Content      string // 模型生成的文本
	FinishReason string // "stop" | "length" | "content_filter"
}
