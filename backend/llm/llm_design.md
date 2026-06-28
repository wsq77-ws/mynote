# MyNote 大语言模型能力接入设计文档

> 本文档描述为 MyNote 接入大语言模型（LLM）能力的完整设计方案。
> 设计遵循项目现有分层架构（`api.Handler → service → storage/meta`），
> 复用 `Storage` 接口读写笔记、复用 `models.APIResponse` 响应信封、
> 复用 `r.Group("/api")` 路由注册方式，保证与现有代码完全兼容。

---

## 1. 功能目标

| 编号 | 功能 | 触发位置 | 说明 |
|------|------|----------|------|
| F1 | 笔记内容自动补全 | 编辑器内 toggle 开关 | 开启后每 3s 触发一次，以最近 100 字符为上下文，单次补全不超过 20 字符 |
| F2 | 根据提示生成笔记内容 | 右侧 sidebar 输入框 | 用户输入提示，调用 LLM 生成对应笔记内容 |
| F3 | 总结所有笔记 | 左侧 sidebar 总结按钮 | 汇总所有笔记内容，生成 `llm_summary.md` 放入 `default` 目录 |
| F4 | LLM 配置管理 | 右侧 sidebar 配置区 | 可编辑 `system_prompt`、`api_key`、`base_url`、`model` |

---

## 2. 整体架构

### 2.1 分层结构（与现有代码一致）

```
HTTP (Gin)
  └─ api/llm_handler.go        // LLMHandler：参数校验 + 响应封装
       └─ service/llm_service.go  // LLMService：业务编排（读笔记/写总结/调用LLM）
            ├─ llm/llm.go          // LLMClient 接口 + 请求/响应结构体
            ├─ llm/openai_compat.go // OpenAI 兼容实现（适配 DeepSeek/OpenAI/Moonshot 等）
            ├─ storage.Storage      // 复用现有接口：读笔记内容、写 llm_summary.md
            └─ llm/config.go        // 配置文件读写（secret_key / system_prompt）
```

### 2.2 与现有代码的关系

| 现有约定 | LLM 模块如何遵循 |
|----------|------------------|
| `Storage` 接口（storage/storage.go） | 总结功能通过 `Storage.List` + `Storage.Read` 读取所有笔记，通过 `Storage.Write` 写入 `llm_summary.md`；不直接调用 `os.ReadFile` |
| `Meta` 接口（meta/meta.go） | 写入 `llm_summary.md` 后调用 `meta.SaveNoteMeta` + `meta.SaveNoteContent` 以纳入搜索索引 |
| `models.APIResponse` 信封 | 所有 LLM 接口返回 `{code, message, data}`，code 语义：200/400/404/500 |
| 路由注册（api/handler.go 的 `RegisterRoutes`） | LLM 路由在 `LLMHandler.RegisterRoutes(r)` 中注册到 `/api/llm/*` |
| 路径格式：`/` 分隔的相对路径 | `llm_summary.md` 写入路径为 `default/llm_summary.md`，不以 `/` 开头 |
| 错误处理：`log.Printf` 显式记录 | LLM 模块所有错误均 `log.Printf("[LLMxxx] ...")`，不使用 `_` 丢弃 |
| SQLite MaxOpenConns=1 | LLM 配置文件为独立文件存储（非 SQLite），不涉及连接池；总结功能的 `Storage.Read` 在循环中调用，需确认 LocalStorage 实现为无状态读（无连接占用） |

---

## 3. 目录与文件结构

```
backend/
├── llm/
│   ├── llm.go               // LLMClient 接口 + CompletionRequest/Response 结构体
│   ├── config.go            // LLMConfig 结构体 + 配置文件读写（secret_key.json / system_prompt.md）
│   ├── openai_compat.go     // OpenAICompatibleClient 实现（适配 DeepSeek 等兼容接口）
│   └── llm_design.md         // 本设计文档
├── api/
│   ├── handler.go            // 现有笔记 API
│   └── llm_handler.go        // LLMHandler：HTTP 处理器（当前为 stub）
├── service/
│   ├── note_service.go       // 现有笔记服务
│   └── llm_service.go        // LLMService：业务编排（当前为 stub）
└── data/                     // 运行时数据目录（已被 .gitignore 忽略）
    └── llm/                  // LLM 运行时配置目录（自动创建）
        ├── secret_key.json   // API Key + Base URL + Model（敏感，不入 git）
        └── system_prompt.md  // 系统提示词（可入 git，但默认放 data 下便于部署隔离）
```

### 3.1 配置文件存放位置说明

设计文档原始意图为"在 llm 目录下放置 system_prompt 和 secret_key 文件"。
经评估，将运行时配置文件放入 `data/llm/` 而非 `llm/`（源码包目录），原因：

1. **部署隔离**：构建后的二进制文件部署时，源码目录 `backend/llm/` 可能不存在；`data/` 是运行时数据目录，始终存在
2. **安全**：`backend/data/*` 已被 `.gitignore` 忽略，`secret_key.json` 自动不会被提交
3. **一致性**：与 SQLite 数据库 `data/mynote.db` 同属运行时数据

配置目录路径通过环境变量 `MYNOTE_LLM_DIR` 可覆盖，默认 `{data_dir}/llm/`。

### 3.2 .gitignore 补充

现有 `.gitignore` 已忽略 `backend/data/*`，因此 `data/llm/secret_key.json` 自动被忽略。
**无需额外修改 .gitignore**。但需在文档中明确：**严禁将 secret_key 文件移出 data 目录**。

---

## 4. LLM Client 接口设计（扩展性）

### 4.1 接口定义（llm/llm.go）

```go
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
    Content    string // 模型生成的文本
    FinishReason string // "stop" | "length" | "content_filter"
}
```

### 4.2 OpenAI 兼容实现（llm/openai_compat.go）

DeepSeek、OpenAI、Moonshot、智谱等均兼容 OpenAI Chat Completions API 格式，
因此默认提供一个通用实现：

```go
// OpenAICompatibleClient 适配所有兼容 OpenAI Chat Completions 的接口
// DeepSeek: base_url=https://api.deepseek.com  model=deepseek-v4-pro
// OpenAI:   base_url=https://api.openai.com    model=gpt-4o
// Moonshot: base_url=https://api.moonshot.cn   model=moonshot-v1-8k
type OpenAICompatibleClient struct {
    apiKey  string
    baseURL string
    model   string
    httpClient *http.Client
}
```

请求体格式（与设计文档原始示例一致）：
```json
{
  "model": "deepseek-v4-pro",
  "messages": [
    {"role": "system", "content": "<system_prompt>"},
    {"role": "user", "content": "<user_prompt>"}
  ],
  "max_tokens": 512,
  "temperature": 0.7,
  "stream": false
}
```

HTTP 调用：
```
POST {base_url}/chat/completions
Authorization: Bearer {api_key}
Content-Type: application/json
```

> 注：原始设计示例中的 `thinking` 和 `reasoning_effort` 为 DeepSeek 专有可选参数，
> 通用实现默认不携带，避免不兼容提供商报错。如需启用推理能力，可通过
> `CompletionRequest` 扩展字段（`Extra map[string]interface{}`）传入。

### 4.3 客户端工厂（扩展点）

```go
// NewClient 根据配置创建 LLM 客户端
func NewClient(cfg *Config) (LLMClient, error) {
    switch cfg.Provider {
    case "", "openai-compatible":
        return NewOpenAICompatibleClient(cfg)
    // 扩展：未来可添加 "anthropic"、"ollama" 等 case
    default:
        return nil, fmt.Errorf("不支持的 LLM 提供商: %s", cfg.Provider)
    }
}
```

新增提供商步骤：
1. 在 `llm/` 下创建新文件实现 `LLMClient` 接口
2. 在 `NewClient` 工厂函数中添加对应 `case`
3. 无需改动 service / handler 层

---

## 5. 配置管理（llm/config.go）

### 5.1 配置结构体

```go
// Config LLM 运行时配置
type Config struct {
    Provider     string  `json:"provider"`     // 提供商标识，默认 "openai-compatible"
    APIKey       string  `json:"api_key"`      // API 密钥
    BaseURL      string  `json:"base_url"`     // 接口地址，如 https://api.deepseek.com
    Model        string  `json:"model"`        // 模型名，如 deepseek-v4-pro
    MaxTokens    int     `json:"max_tokens"`   // 全局最大生成 token 数，默认 512，作用于所有端点
    Temperature  float64 `json:"temperature"`  // 采样温度 0~2，默认 0.7
    SystemPrompt string  `json:"-"`            // 系统提示词（单独文件存储，不序列化到 secret_key.json）
}
```

> **可配置模型参数**：`MaxTokens` 与 `Temperature` 为全局参数，作用于补全/生成/总结三个端点。
> 默认值 `DefaultMaxTokens=512`、`DefaultTemperature=0.7`（导出常量，定义于 `llm/config.go`）。
> 旧配置文件缺失这两个字段时，`LoadConfig` 自动回填默认值。合法范围由 `ValidateModelParams` 校验：
> `max_tokens ∈ [1, 8192]`、`temperature ∈ (0, 2]`。

### 5.2 文件格式

**`data/llm/secret_key.json`**（敏感，gitignored）：
```json
{
  "provider": "openai-compatible",
  "api_key": "sk-xxxxxxxxxxxxxxxx",
  "base_url": "https://api.deepseek.com",
  "model": "deepseek-v4-pro",
  "max_tokens": 512,
  "temperature": 0.7
}
```

**`data/llm/system_prompt.md`**（纯文本）：
```
你是一个知识笔记助手。请根据用户的输入提供简洁、准确的回答。
```

### 5.3 配置读写 API

| 操作 | 函数 | 说明 |
|------|------|------|
| 加载配置 | `LoadConfig(dir string) (*Config, error)` | 读取 secret_key.json + system_prompt.md；文件不存在返回默认配置（`provider="openai-compatible"`、`MaxTokens=512`、`Temperature=0.7`、`SystemPrompt=默认提示词`，非错误） |
| 保存密钥配置 | `SaveSecretConfig(dir string, cfg *Config) error` | 写入 secret_key.json，文件权限 `0600` |
| 保存系统提示词 | `SaveSystemPrompt(dir, prompt string) error` | 写入 system_prompt.md |
| 获取脱敏配置 | `MaskedConfig(cfg *Config) *Config` | 返回 api_key 仅保留后 4 位，前缀 `****`（MaxTokens/Temperature 原值返回） |
| 判断脱敏值 | `IsMaskedKey(key string) bool` | 判断 api_key 是否以 `****` 开头（用于 PUT 时忽略脱敏回写） |
| 校验 base_url | `ValidateBaseURL(url string) error` | 必须以 `http://` 或 `https://` 开头 |
| 校验模型参数 | `ValidateModelParams(maxTokens int, temperature float64) error` | `max_tokens ∈ [1, 8192]`、`temperature ∈ (0, 2]` |

> **默认系统提示词**：当 `system_prompt.md` 不存在时，`LoadConfig` 返回内置默认值（"你是一个知识笔记助手..."），保证 LLM 开箱即用；文件存在时以文件内容为准。

---

## 6. API 接口设计

所有接口注册在 `/api/llm` 路由组下，遵循 `models.APIResponse` 信封。

### 6.1 路由总览

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/llm/config` | 获取 LLM 配置（api_key 脱敏） |
| `PUT` | `/api/llm/config` | 更新 LLM 配置 |
| `POST` | `/api/llm/complete` | 自动补全（F1） |
| `POST` | `/api/llm/generate` | 生成笔记内容（F2） |
| `POST` | `/api/llm/summarize` | 总结所有笔记（F3） |

### 6.2 接口详情

#### `GET /api/llm/config`
获取当前 LLM 配置。**api_key 脱敏返回**（仅后 4 位）。

**响应 Data**:
```json
{
  "provider": "openai-compatible",
  "api_key": "****1234",
  "base_url": "https://api.deepseek.com",
  "model": "deepseek-v4-pro",
  "system_prompt": "你是一个知识笔记助手...",
  "configured": true
}
```
- `configured`：`api_key` 非空时为 `true`，前端据此判断是否可调用 LLM

---

#### `PUT /api/llm/config`
更新 LLM 配置。所有字段可选，仅更新提供的字段。

| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `api_key` | body | string | 否 | - | API 密钥；传 `""` 清空，传 `****` 开头则忽略（脱敏值回写保护） |
| `base_url` | body | string | 否 | - | 接口地址 |
| `model` | body | string | 否 | - | 模型名 |
| `max_tokens` | body | int | 否 | 512 | 全局最大生成 token 数，作用于所有端点；合法范围 [1, 8192] |
| `temperature` | body | float | 否 | 0.7 | 采样温度，作用于所有端点；合法范围 (0, 2] |
| `system_prompt` | body | string | 否 | - | 系统提示词 |

**安全规则**：若 `api_key` 以 `****` 开头，视为脱敏值回传，后端忽略不更新。
`max_tokens` / `temperature` 持久化前由 `ValidateModelParams` 校验，非法返回 400。

---

#### `POST /api/llm/complete`（F1 自动补全）
| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `text` | body | string | **是** | - | 当前编辑器末尾文本（前端截取最近 100 字符） |

**校验规则**：
- `text` 长度 ≤ 500 字符（超出截断）
- `max_tokens` / `temperature` 取自全局配置（默认 512 / 0.7，可在配置页修改）
- 请求超时：15s

**响应 Data**:
```json
{ "suggestion": "补充的内容" }
```

---

#### `POST /api/llm/generate`（F2 生成笔记）
| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `prompt` | body | string | **是** | - | 生成笔记的提示词 |

**校验规则**：
- `prompt` 长度 ≤ 2000 字符
- 请求超时：120s（生成内容较长）

**响应 Data**:
```json
{ "content": "# 生成的标题\n\n正文内容..." }
```

---

#### `POST /api/llm/summarize`（F3 总结所有笔记）
无请求参数。

**后端流程**：
1. 通过 `Storage.List(".")` 递归收集所有 `.md` 文件路径
2. 逐个 `Storage.Read(path)` 读取内容，拼接为 `# {path}\n\n{content}\n\n---\n` 格式
3. 以 system_prompt + 总结指令构造 `CompletionRequest`，`max_tokens` / `temperature` 取自全局配置（默认 512 / 0.7；总结长内容时建议在配置页调大 max_tokens）
4. 调用 `LLMClient.Complete`
5. 通过 `Storage.Write("default/llm_summary.md", summary)` 写入总结文档
6. 调用 `meta.SaveNoteMeta` + `meta.SaveNoteContent` 纳入元数据与搜索索引

**响应 Data**:
```json
{ "path": "default/llm_summary.md", "note_count": 12 }
```

**安全规则**：
- 总结文档路径硬编码为 `default/llm_summary.md`，不接受前端传入路径（防路径穿越）
- 单次总结笔记数上限 100，超出截断并记录日志

---

## 7. 数据模型（models/llm.go）

```go
package models

// LLMConfigResponse LLM 配置响应（api_key 脱敏）
type LLMConfigResponse struct {
    Provider     string `json:"provider"`
    APIKey       string `json:"api_key"`       // 脱敏：****1234
    BaseURL      string `json:"base_url"`
    Model        string `json:"model"`
    SystemPrompt string `json:"system_prompt"`
    Configured   bool   `json:"configured"`
}

// UpdateLLMConfigRequest 更新 LLM 配置请求
type UpdateLLMConfigRequest struct {
    APIKey       *string `json:"api_key,omitempty"`
    BaseURL      *string `json:"base_url,omitempty"`
    Model        *string `json:"model,omitempty"`
    SystemPrompt *string `json:"system_prompt,omitempty"`
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
```

---

## 8. Service 层设计（service/llm_service.go）

```go
package service

type LLMService struct {
    client   llm.LLMClient   // LLM 客户端（可热重载）
    storage  storage.Storage // 复用笔记存储
    meta     meta.Meta       // 复用元数据
    configDir string         // 配置目录路径
    mu       sync.RWMutex    // 保护 client 热重载
}

func NewLLMService(s storage.Storage, m meta.Meta, configDir string) *LLMService

// GetConfig 读取配置（api_key 脱敏）
func (s *LLMService) GetConfig() (*models.LLMConfigResponse, error)

// UpdateConfig 更新配置并热重载客户端
func (s *LLMService) UpdateConfig(req models.UpdateLLMConfigRequest) error

// Complete 自动补全
func (s *LLMService) Complete(text string) (string, error)

// Generate 生成笔记内容
func (s *LLMService) Generate(prompt string) (string, error)

// Summarize 总结所有笔记并写入 default/llm_summary.md
func (s *LLMService) Summarize() (*models.SummarizeResponse, error)
```

### 8.1 客户端热重载

`UpdateConfig` 保存配置后，重新调用 `llm.NewClient` 创建新客户端并替换（通过 `mu` 写锁保护）。
`Complete`/`Generate`/`Summarize` 通过读锁获取当前 client。这样用户修改 API Key 后无需重启服务。

### 8.2 总结功能的笔记遍历

```go
// maxSummaryNotes 总结笔记数上限（与 6.2/11.3 安全规则一致）
const maxSummaryNotes = 100

func (s *LLMService) collectAllNotes() ([]noteContent, error) {
    var notes []noteContent
    var walk func(dir string) error
    walk = func(dir string) error {
        entries, err := s.storage.List(dir)
        if err != nil {
            return err
        }
        for _, e := range entries {
            // 达到上限后停止收集（11.3 安全规则）
            if len(notes) >= maxSummaryNotes {
                return nil
            }
            if e.IsDir {
                if err := walk(e.Path); err != nil {
                    // 递归失败必须显式记录，禁止用 _ 丢弃
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
```

> 注：LocalStorage.Read 为无状态文件读取，不存在 SQLite 连接池占用问题，可安全在循环中调用。
> **实现约束**：错误处理遵循项目硬约束——禁止用 `_` 丢弃 error，所有错误必须 `log.Printf` 显式记录。

---

## 9. Handler 层设计（api/llm_handler.go）

```go
package api

type LLMHandler struct {
    svc *service.LLMService
}

func NewLLMHandler(svc *service.LLMService) *LLMHandler

func (h *LLMHandler) RegisterRoutes(r *gin.Engine) {
    g := r.Group("/api/llm")
    {
        g.GET("/config", h.GetConfig)
        g.PUT("/config", h.UpdateConfig)
        g.POST("/complete", h.Complete)
        g.POST("/generate", h.Generate)
        g.POST("/summarize", h.Summarize)
    }
}
```

Handler 遵循现有错误处理模式：
- 参数缺失/校验失败 → `400 APIResponse{Code: 400, Message: "..."}`
- LLM 未配置（api_key 为空）→ `400 APIResponse{Code: 400, Message: "LLM 未配置，请先设置 API Key"}`
- LLM 调用失败 → `500 APIResponse{Code: 500, Message: "LLM 调用失败: " + err.Error()}`
- 成功 → `200 APIResponse{Code: 200, Message: "success", Data: ...}`

---

## 10. main.go 接入方式

在 `main.go` 中 `noteSvc` 创建之后、路由注册处添加：

```go
// 创建 LLM 服务
llmConfigDir := filepath.Join(cfg.Storage.Local.DataDir, "llm")
if envDir := os.Getenv("MYNOTE_LLM_DIR"); envDir != "" {
    llmConfigDir = envDir
}
llmSvc := service.NewLLMService(store, metaStore, llmConfigDir)
llmHandler := api.NewLLMHandler(llmSvc)
llmHandler.RegisterRoutes(r)
```

### 新增环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYNOTE_LLM_DIR` | LLM 配置文件目录 | `{data_dir}/llm/` |

---

## 11. 安全设计

### 11.1 API Key 保护

| 威胁 | 防护措施 |
|------|----------|
| Key 泄露到 Git | 配置文件存放于 `data/llm/`，已被 `.gitignore` 的 `backend/data/*` 规则忽略 |
| Key 通过 API 返回泄露 | `GET /api/llm/config` 返回脱敏值（`****1234`），仅后 4 位可见 |
| 脱敏值回写覆盖真实 Key | `PUT /api/llm/config` 检测 `api_key` 以 `****` 开头时忽略更新 |
| Key 出现在日志 | LLM 客户端日志仅记录 `base_url` 和 `model`，**严禁记录 api_key** |
| 配置文件权限 | `secret_key.json` 创建时设权限 `0600`（仅所有者可读写） |

### 11.2 路径安全

| 场景 | 防护措施 |
|------|----------|
| 总结文档路径 | 硬编码 `default/llm_summary.md`，不接受前端传入路径 |
| 笔记遍历 | 仅读取 `Storage.List` 返回的相对路径，不拼接用户输入 |

### 11.3 输入校验

| 场景 | 限制 |
|------|------|
| 补全 `text` | ≤ 500 字符（超出截断） |
| 生成 `prompt` | ≤ 2000 字符（超出返回 400） |
| 总结笔记数 | ≤ 100 篇（超出截断 + 日志告警） |

### 11.4 调用安全

| 场景 | 限制 |
|------|------|
| 超时控制 | 补全 15s / 生成 120s / 总结 180s，通过 `context.WithTimeout` |
| 客户端校验 | api_key 为空时拒绝调用，返回 `400 LLM 未配置` |
| base_url 校验 | 必须以 `http://` 或 `https://` 开头，禁止 `file://` 等协议 |

---

## 12. 扩展性设计

### 12.1 新增 LLM 提供商

1. 在 `llm/` 下新建文件（如 `anthropic.go`）实现 `LLMClient` 接口
2. 在 `llm/config.go` 的 `NewClient` 工厂添加 `case "anthropic"`
3. 在 `secret_key.json` 中设置 `"provider": "anthropic"`
4. 无需改动 service / handler / 前端

### 12.2 流式输出（未来扩展）

当前 `CompletionRequest` 不含 `Stream` 字段（默认非流式）。
未来如需流式输出（打字机效果），可：

1. 扩展 `LLMClient` 接口添加 `StreamComplete(ctx, req) (<-chan StreamChunk, error)`
2. Handler 使用 Server-Sent Events (SSE) 推送
3. `OpenAICompatibleClient` 设置 `"stream": true` 并逐行解析

### 12.3 补全模型独立配置（未来扩展）

当前 system_prompt 全局共享。未来可为补全、生成、总结分别配置独立提示词：
```
data/llm/
├── secret_key.json
├── system_prompt.md          // 默认
├── complete_prompt.md       // 补全专用
├── generate_prompt.md       // 生成专用
└── summarize_prompt.md      // 总结专用
```

---

## 13. 前端集成（概要）

### 13.1 API 封装（frontend/src/api/index.js 新增）

```js
// LLM 配置
export function getLLMConfig() {
  return api.get('/llm/config')
}
export function updateLLMConfig(data) {
  return api.put('/llm/config', data)
}
// 自动补全（不参与重试，独立短超时）
export function llmComplete(text) {
  return api.post('/llm/complete', { text }, { timeout: 15000, __noRetry: true })
}
// 生成笔记
export function llmGenerate(prompt) {
  return api.post('/llm/generate', { prompt }, { timeout: 120000 })
}
// 总结所有笔记
export function llmSummarize() {
  return api.post('/llm/summarize', {}, { timeout: 180000 })
}
```

### 13.2 组件改动

| 功能 | 组件 | 改动 |
|------|------|------|
| F1 自动补全 | NoteEditor.vue | 新增 `el-switch` 开关（AI）；`watch(content)` 防抖 3s 调用 `llmComplete`；建议以浮层建议条展示，Tab 接受 / Esc 拒绝 |
| F2 生成笔记 | 新增 LLMPanel.vue（右侧 sidebar） | 输入框 + 生成按钮；生成内容可插入编辑器（`insertContent`）或另存为新笔记 |
| F3 总结 | Sidebar.vue | 新增"总结"按钮（二次确认）；调用后刷新目录树以显示 `llm_summary.md`，并向 App.vue 发送 `summarize-done` 事件 |
| F4 配置 | LLMPanel.vue | system_prompt 文本域 + api_key/base_url/model/max_tokens/temperature 输入框；保存调用 `updateLLMConfig`；脱敏值未修改时不回传；max_tokens/temperature 前端校验范围 [1,8192] / (0,2] |
| 面板挂载 | App.vue | 新增可折叠右侧 `.llm-sidebar` 容器承载 LLMPanel；快捷键 `Ctrl+L` 切换 |

### 13.3 自动补全防抖逻辑

```js
let completeTimer = null
let completeInFlight = false
let skipNextComplete = false // 切换笔记加载的首次变化不触发

watch(content, (val) => {
  if (skipNextComplete) { skipNextComplete = false; return }
  if (!autoCompleteEnabled.value) return
  if (completeInFlight) return            // 上一次请求在途，跳过
  if (currentSuggestion.value) return     // 有未接受建议，跳过
  if (!val || val.length < 5) return

  if (completeTimer) clearTimeout(completeTimer)
  completeTimer = setTimeout(() => fetchSuggestion(), 3000)
})

async function fetchSuggestion() {
  const tail = content.value.slice(-100)  // 最近 100 字符
  completeInFlight = true
  try {
    const res = await llmComplete(tail)
    if (res.data.code === 200) {
      const suggestion = (res.data.data.suggestion || '').trim()
      // 仅当内容未在请求期间变化时展示，避免错位
      if (suggestion && content.value.slice(-100) === tail) {
        currentSuggestion.value = suggestion
      }
    }
    // 失败静默忽略（15.5 离线降级）
  } catch (err) {
    console.debug('[LLMComplete] 失败:', err)
  } finally {
    completeInFlight = false
  }
}
```

> **实现说明（F1 视觉呈现）**：设计原始意图为"灰色幽灵文本"内联于编辑器。
> md-editor-v3 不原生支持内联 ghost text，强行 hack 其内部 textarea/codeMirror 脆弱且维护成本高。
> 实际采用**浮层建议条**（编辑器右下角，灰色文本 + `Tab 接受` / `Esc 拒绝` 按钮），
> 全局 `keydown` 监听 Tab/Esc。核心行为（toggle、3s 防抖、最近 100 字符上下文、Tab 接受、Esc 拒绝、失败静默）完全保留。

---

## 14. 实现优先级

| 阶段 | 内容 | 依赖 |
|------|------|------|
| P1 | `llm/llm.go` 接口 + `llm/config.go` 配置读写 + `llm/openai_compat.go` 客户端 | 无 |
| P2 | `service/llm_service.go` 业务编排 + `api/llm_handler.go` 路由 + `main.go` 接入 | P1 |
| P3 | 前端 `api/index.js` 封装 + LLMPanel.vue（F2 + F4） | P2 |
| P4 | NoteEditor.vue 自动补全（F1） | P3 |
| P5 | Sidebar.vue 总结按钮（F3） | P3 |

---

## 15. 注意事项

1. **接口幂等性**：`PUT /api/llm/config` 为部分更新，未提供的字段保持原值
2. **错误暴露**：所有错误通过 `log.Printf("[LLMxxx] ...")` 记录，handler 层将错误信息透传到 `message` 字段
3. **总结文档覆盖**：重复调用总结会覆盖 `default/llm_summary.md`，前端应二次确认
4. **FTS5 兼容**：写入 `llm_summary.md` 后调用 `meta.SaveNoteContent` 纳入搜索索引，使用 UPSERT 语义（注意 FTS5 虚拟表不支持 UPSERT，需先 DELETE 再 INSERT，参考 `sqlite.go` 现有实现）
5. **离线降级**：LLM 调用失败时不应影响笔记编辑功能；自动补全失败静默忽略，生成/总结失败返回错误提示
6. **成本控制**：自动补全每 3s 一次频率较高，前端应在用户停止输入后才触发，避免无谓调用
