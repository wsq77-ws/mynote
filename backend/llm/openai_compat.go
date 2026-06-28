package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// OpenAICompatibleClient 适配所有兼容 OpenAI Chat Completions 的接口
// DeepSeek: base_url=https://api.deepseek.com  model=deepseek-v4-pro
// OpenAI:   base_url=https://api.openai.com    model=gpt-4o
// Moonshot: base_url=https://api.moonshot.cn   model=moonshot-v1-8k
type OpenAICompatibleClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAICompatibleClient 创建 OpenAI 兼容客户端
func NewOpenAICompatibleClient(cfg *Config) (*OpenAICompatibleClient, error) {
	return &OpenAICompatibleClient{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: 0, // 超时由调用方通过 context 控制
		},
	}, nil
}

// Name 返回客户端标识
func (c *OpenAICompatibleClient) Name() string {
	return "openai-compatible"
}

// chatMessage OpenAI Chat Completions 消息体
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest OpenAI Chat Completions 请求体
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// chatChoice 响应中的选择项
type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// chatResponse OpenAI Chat Completions 响应体
type chatResponse struct {
	ID      string        `json:"id"`
	Choices []chatChoice  `json:"choices"`
	Error   *chatAPIError `json:"error,omitempty"`
}

// chatAPIError API 返回的错误结构
type chatAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Complete 发起一次补全请求
func (c *OpenAICompatibleClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("LLM 未配置，请先设置 API Key")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("LLM base_url 未配置")
	}
	if c.model == "" {
		return nil, fmt.Errorf("LLM model 未配置")
	}

	// 构造消息列表：system_prompt 为空时不发送 system 消息
	var messages []chatMessage
	if strings.TrimSpace(req.SystemPrompt) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: req.UserPrompt})

	body := chatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("构造请求体失败: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 严禁记录 api_key，仅记录 base_url 与 model
	log.Printf("[LLMClient] 发起请求 base_url=%s model=%s max_tokens=%d", c.baseURL, c.model, req.MaxTokens)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 LLM 失败: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误信息，但不暴露 api_key
		var chatResp chatResponse
		if jsonErr := json.Unmarshal(respBody, &chatResp); jsonErr == nil && chatResp.Error != nil {
			return nil, fmt.Errorf("LLM 返回错误 (status=%d): %s", resp.StatusCode, chatResp.Error.Message)
		}
		return nil, fmt.Errorf("LLM 返回非 200 状态码: %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 未返回任何内容")
	}

	choice := chatResp.Choices[0]
	return &CompletionResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
	}, nil
}
