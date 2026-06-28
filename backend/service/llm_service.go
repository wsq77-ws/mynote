package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"mynote-backend/llm"
	"mynote-backend/meta"
	"mynote-backend/models"
	"mynote-backend/storage"
)

// 总结文档固定路径（硬编码，防路径穿越，遵循"笔记默认目录"约束）
const llmSummaryPath = "default/llm_summary.md"

// 总结笔记数上限（超出截断 + 日志告警）
const maxSummaryNotes = 100

// 各接口超时（max_tokens / temperature 由 cfg 提供，全局可配置）
const (
	completeTimeout  = 15 * time.Second
	generateTimeout  = 120 * time.Second
	summarizeTimeout = 180 * time.Second
)

// LLMService LLM 业务编排服务
type LLMService struct {
	client    llm.LLMClient // LLM 客户端（可热重载，可能为 nil 表示未配置）
	cfg       *llm.Config   // 当前配置（含明文 api_key，内存中保存）
	storage   storage.Storage
	meta      meta.Meta
	configDir string
	mu        sync.RWMutex // 保护 client / cfg 热重载
}

// noteContent 笔记内容缓存（仅用于总结遍历）
type noteContent struct {
	path    string
	content string
}

// NewLLMService 创建 LLM 服务
// 启动时加载配置并尝试创建客户端（api_key 为空时 client 为 nil，调用时拒绝）
func NewLLMService(s storage.Storage, m meta.Meta, configDir string) *LLMService {
	svc := &LLMService{
		storage:   s,
		meta:      m,
		configDir: configDir,
	}
	cfg, err := llm.LoadConfig(configDir)
	if err != nil {
		log.Printf("[LLMService] 加载配置失败: %v", err)
		svc.cfg = &llm.Config{
			Provider:    llm.DefaultProvider,
			MaxTokens:   llm.DefaultMaxTokens,
			Temperature: llm.DefaultTemperature,
		}
		return svc
	}
	svc.cfg = cfg
	if cfg.APIKey == "" {
		log.Printf("[LLMService] LLM 未配置 API Key，相关接口将返回 400")
		return svc
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		log.Printf("[LLMService] 创建客户端失败: %v", err)
		return svc
	}
	svc.client = client
	log.Printf("[LLMService] 客户端就绪 provider=%s model=%s", cfg.Provider, cfg.Model)
	return svc
}

// GetConfig 读取配置（api_key 脱敏）
func (s *LLMService) GetConfig() (*models.LLMConfigResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := s.cfg
	if cfg == nil {
		cfg = &llm.Config{
			Provider:    llm.DefaultProvider,
			MaxTokens:   llm.DefaultMaxTokens,
			Temperature: llm.DefaultTemperature,
		}
	}
	masked := llm.MaskedConfig(cfg)
	return &models.LLMConfigResponse{
		Provider:     masked.Provider,
		APIKey:       masked.APIKey,
		BaseURL:      masked.BaseURL,
		Model:        masked.Model,
		MaxTokens:    masked.MaxTokens,
		Temperature:  masked.Temperature,
		SystemPrompt: cfg.SystemPrompt,
		Configured:   cfg.APIKey != "",
	}, nil
}

// UpdateConfig 更新配置并热重载客户端
// api_key 以 **** 开头视为脱敏值回传，忽略更新
func (s *LLMService) UpdateConfig(req models.UpdateLLMConfigRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg == nil {
		s.cfg = &llm.Config{
			Provider:    llm.DefaultProvider,
			MaxTokens:   llm.DefaultMaxTokens,
			Temperature: llm.DefaultTemperature,
		}
	}

	// 应用字段更新
	if req.APIKey != nil {
		if llm.IsMaskedKey(*req.APIKey) {
			// 脱敏值回写保护：忽略
			log.Printf("[LLMService] 检测到脱敏 api_key 回传，忽略更新")
		} else {
			s.cfg.APIKey = *req.APIKey
		}
	}
	if req.BaseURL != nil {
		if err := llm.ValidateBaseURL(*req.BaseURL); err != nil {
			return err
		}
		s.cfg.BaseURL = *req.BaseURL
	}
	if req.Model != nil {
		s.cfg.Model = *req.Model
	}
	// max_tokens / temperature：指针非 nil 才更新，并校验合法性
	if req.MaxTokens != nil {
		s.cfg.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		s.cfg.Temperature = *req.Temperature
	}
	// 校验模型参数（在持久化前统一校验，避免写入非法值）
	if err := llm.ValidateModelParams(s.cfg.MaxTokens, s.cfg.Temperature); err != nil {
		return err
	}

	// 持久化密钥配置（system_prompt 单独存储）
	if err := llm.SaveSecretConfig(s.configDir, s.cfg); err != nil {
		log.Printf("[LLMService] 保存密钥配置失败: %v", err)
		return err
	}

	// 系统提示词单独存储
	if req.SystemPrompt != nil {
		if err := llm.SaveSystemPrompt(s.configDir, *req.SystemPrompt); err != nil {
			log.Printf("[LLMService] 保存系统提示词失败: %v", err)
			return err
		}
		s.cfg.SystemPrompt = *req.SystemPrompt
	}

	// 热重载客户端
	if s.cfg.APIKey == "" {
		s.client = nil
		log.Printf("[LLMService] api_key 已清空，客户端卸载")
		return nil
	}
	client, err := llm.NewClient(s.cfg)
	if err != nil {
		log.Printf("[LLMService] 热重载客户端失败: %v", err)
		return err
	}
	s.client = client
	log.Printf("[LLMService] 配置已更新并热重载 provider=%s model=%s", s.cfg.Provider, s.cfg.Model)
	return nil
}

// Complete 自动补全（F1）
func (s *LLMService) Complete(text string) (string, error) {
	s.mu.RLock()
	client, cfg := s.client, s.cfg
	s.mu.RUnlock()

	if client == nil || cfg == nil || cfg.APIKey == "" {
		return "", fmt.Errorf("LLM 未配置，请先设置 API Key")
	}

	// 截断超长文本（超出 500 字符）
	if len([]rune(text)) > 500 {
		text = string([]rune(text)[:500])
	}

	ctx, cancel := context.WithTimeout(context.Background(), completeTimeout)
	defer cancel()

	resp, err := client.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: cfg.SystemPrompt,
		UserPrompt:   "请根据以下文本内容进行补全，只返回补全部分，不要重复原文：\n" + text,
		MaxTokens:    cfg.MaxTokens,
		Temperature:  cfg.Temperature,
	})
	if err != nil {
		log.Printf("[LLMComplete] 调用失败: %v", err)
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}
	log.Printf("[LLMComplete] 完成 max_tokens=%d temperature=%.2f finish=%s len=%d",
		cfg.MaxTokens, cfg.Temperature, resp.FinishReason, len([]rune(resp.Content)))
	return strings.TrimSpace(resp.Content), nil
}

// Generate 生成笔记内容（F2）
func (s *LLMService) Generate(prompt string) (string, error) {
	s.mu.RLock()
	client, cfg := s.client, s.cfg
	s.mu.RUnlock()

	if client == nil || cfg == nil || cfg.APIKey == "" {
		return "", fmt.Errorf("LLM 未配置，请先设置 API Key")
	}

	ctx, cancel := context.WithTimeout(context.Background(), generateTimeout)
	defer cancel()

	resp, err := client.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: cfg.SystemPrompt,
		UserPrompt:   prompt,
		MaxTokens:    cfg.MaxTokens,
		Temperature:  cfg.Temperature,
	})
	if err != nil {
		log.Printf("[LLMGenerate] 调用失败: %v", err)
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}
	log.Printf("[LLMGenerate] 完成 max_tokens=%d temperature=%.2f finish=%s len=%d",
		cfg.MaxTokens, cfg.Temperature, resp.FinishReason, len([]rune(resp.Content)))
	return strings.TrimSpace(resp.Content), nil
}

// Summarize 总结所有笔记并写入 default/llm_summary.md（F3）
func (s *LLMService) Summarize() (*models.SummarizeResponse, error) {
	s.mu.RLock()
	client, cfg := s.client, s.cfg
	s.mu.RUnlock()

	if client == nil || cfg == nil || cfg.APIKey == "" {
		return nil, fmt.Errorf("LLM 未配置，请先设置 API Key")
	}

	// 1. 收集所有笔记
	notes, err := s.collectAllNotes()
	if err != nil {
		return nil, fmt.Errorf("收集笔记失败: %w", err)
	}
	if len(notes) == 0 {
		return nil, fmt.Errorf("没有可总结的笔记")
	}

	// 2. 拼接笔记内容
	var sb strings.Builder
	for _, n := range notes {
		sb.WriteString("# ")
		sb.WriteString(n.path)
		sb.WriteString("\n\n")
		sb.WriteString(n.content)
		sb.WriteString("\n\n---\n")
	}

	// 3. 调用 LLM 生成总结
	ctx, cancel := context.WithTimeout(context.Background(), summarizeTimeout)
	defer cancel()

	userPrompt := "请总结以下所有笔记的核心内容，按主题归类整理，生成一份结构化的 Markdown 总结文档：\n\n" + sb.String()
	resp, err := client.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: cfg.SystemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    cfg.MaxTokens,
		Temperature:  cfg.Temperature,
	})
	if err != nil {
		log.Printf("[LLMSummarize] 调用失败: %v", err)
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}
	log.Printf("[LLMSummarize] LLM 完成 max_tokens=%d temperature=%.2f finish=%s len=%d",
		cfg.MaxTokens, cfg.Temperature, resp.FinishReason, len([]rune(resp.Content)))

	summary := strings.TrimSpace(resp.Content)

	// 4. 写入总结文档
	if err := s.storage.Write(llmSummaryPath, summary); err != nil {
		log.Printf("[LLMSummarize] 写入总结文档失败: %v", err)
		return nil, fmt.Errorf("写入总结文档失败: %w", err)
	}

	// 5. 纳入元数据与搜索索引
	// 笔记名按约定去掉 .md 后缀
	noteName := strings.TrimSuffix("llm_summary.md", ".md")
	if err := s.meta.SaveNoteMeta(&meta.NoteMeta{
		Path:  llmSummaryPath,
		Name:  noteName,
		IsDir: false,
	}); err != nil {
		log.Printf("[LLMSummarize] 保存元数据失败: %v", err)
	}
	// FTS5 虚拟表不支持 UPSERT，SaveNoteContent 内部已用 DELETE+INSERT
	if err := s.meta.SaveNoteContent(llmSummaryPath, summary); err != nil {
		log.Printf("[LLMSummarize] 保存内容索引失败: %v", err)
	}

	log.Printf("[LLMSummarize] 总结完成，已写入 %s，共 %d 篇笔记", llmSummaryPath, len(notes))

	return &models.SummarizeResponse{
		Path:      llmSummaryPath,
		NoteCount: len(notes),
	}, nil
}

// collectAllNotes 递归收集所有 .md 笔记
// 上限 maxSummaryNotes 篇，超出截断并记录日志
func (s *LLMService) collectAllNotes() ([]noteContent, error) {
	var notes []noteContent
	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := s.storage.List(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			// 达到上限后停止收集
			if len(notes) >= maxSummaryNotes {
				return nil
			}
			if e.IsDir {
				if err := walk(e.Path); err != nil {
					// 记录递归失败，不丢弃错误（遵循错误处理约束）
					log.Printf("[LLMSummarize] 遍历子目录失败 dir=%s: %v", e.Path, err)
				}
				continue
			}
			if !strings.HasSuffix(e.Name, ".md") {
				continue
			}
			content, _, err := s.storage.Read(e.Path)
			if err != nil {
				log.Printf("[LLMSummarize] 跳过读取失败的笔记 %s: %v", e.Path, err)
				continue
			}
			notes = append(notes, noteContent{path: e.Path, content: content})
		}
		return nil
	}
	if err := walk("."); err != nil {
		return nil, err
	}

	if len(notes) >= maxSummaryNotes {
		log.Printf("[LLMSummarize] 笔记数已达上限 %d，超出部分已截断", maxSummaryNotes)
	}
	return notes, nil
}
