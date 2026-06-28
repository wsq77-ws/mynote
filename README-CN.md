# MyNote - 在线知识笔记系统

一个基于 Markdown 的本地知识笔记系统，支持层级目录结构，提供所见即所得的编辑体验。当前运行在本地，已为云端部署做好准备。

## 功能特性

- **Markdown 编辑** — 基于 md-editor-v3，支持代码高亮、表格、列表等丰富语法
- **层级目录树** — 左侧目录树支持无限层级嵌套，目录/笔记分类管理
- **自动保存** — 编辑内容 2 秒自动保存，也可手动保存
- **离线保存** — 网络不可用时自动保存到本地，网络恢复后自动同步到服务器
- **右键菜单** — 目录树支持右键新建笔记/目录、删除节点、重命名
- **实时预览** — 所见即所得的编辑体验
- **快捷键支持** — `Ctrl+S` 保存、`Ctrl+F` 搜索、`Ctrl+N` 新建笔记
- **字数统计** — 实时显示字数、行数、预计阅读时间
- **全局搜索** — 支持搜索笔记名称、路径、内容、**标签**，搜索结果展示匹配类型与高亮标签
- **标签系统** — 为每篇笔记添加标签，按标签分类和搜索，新增标签立即可被搜索到
- **拖拽排序** — 支持拖拽调整目录树中笔记和目录的顺序
- **AI 助手** — 集成 OpenAI 兼容大模型（DeepSeek / OpenAI / Moonshot / 智谱）：行内自动补全（Tab 接受 / Esc 拒绝）、内容生成、一键总结所有笔记，配置面板可调 API Key、模型、`max_tokens`、温度、系统提示词。详见 [LLM 配置](#llm-配置)
- **可插拔存储** — 支持本地文件系统和对象存储（S3 兼容），通过配置文件切换
- **一键部署** — 生产模式下后端自动服务前端静态文件，单端口运行

## 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | Vue 3 + Vite + Element Plus + md-editor-v3 |
| **后端** | Go 1.23+ / Gin |
| **存储** | 可插拔存储层：本地文件系统 / 对象存储（S3 兼容） |

## 环境要求

- [Node.js](https://nodejs.org/) 18+（含 npm）
- [Go](https://go.dev/dl/) 1.23+

## 快速开始

### 开发模式

前端和后端分开运行，支持热重载。

**方式一：一键启动（推荐）**

双击项目根目录的 `start-dev.bat`。

**方式二：分别启动**

```bash
# 终端1 - 启动后端
cd backend
go run main.go

# 终端2 - 启动前端
cd frontend
npm install
npm run dev
```

启动后访问：
- 前端页面：http://localhost:3000
- 后端 API：http://localhost:8080

> 开发模式下，前端通过 Vite 代理将 `/api` 请求转发到后端 8080 端口。

### 生产构建

```bash
# 使用构建脚本（PowerShell）
.\scripts\build.ps1

# 或手动构建
cd frontend && npm run build
cd backend && go build -o mynote-server.exe .
```

构建产物位于 `build/` 目录：
- `mynote-server.exe` — 后端可执行文件
- `dist/` — 前端静态文件
- `data/` — 笔记数据目录
- `start.bat` — 启动脚本

运行方式：双击 `build/start.bat`，或直接执行 `mynote-server.exe`，访问 http://localhost:8080。

## 使用指南

### 创建笔记

1. 点击侧边栏顶部「新建」按钮，在根目录创建笔记
2. 或在目录树上右键目录，选择「新建笔记」/「新建目录」
3. 使用快捷键 `Ctrl+N` 快速新建笔记

### 编辑笔记

1. 在左侧目录树点击任意笔记文件
2. 在右侧编辑器中编写 Markdown 内容
3. 内容会在停止输入 2 秒后自动保存，也可：
   - 点击「保存」按钮手动保存
   - 使用快捷键 `Ctrl+S` 快速保存

### 离线保存

当网络不可用时，编辑器会自动切换到离线模式：

- **自动降级**：保存到服务器失败时，内容自动缓存到浏览器 localStorage
- **状态提示**：标题旁显示「离线」或「待同步」标签，底部状态栏同步提示
- **自动同步**：网络恢复后，自动将所有离线修改同步到服务器
- **手动同步**：点击标题旁的「同步」按钮可手动触发同步
- **缓存优先**：加载笔记时，如果检测到本地有未同步的缓存，优先使用缓存版本

### 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+S` | 保存当前笔记 |
| `Ctrl+F` | 打开搜索框 |
| `Ctrl+N` | 新建笔记 |
| `Ctrl+B` | 切换侧边栏显示 |
| `Ctrl+L` | 切换 AI 助手面板 |

### 字数统计

编辑器底部实时显示：
- 字数（中文按字符，英文按单词）
- 行数
- 预计阅读时间（按200字/分钟）

### 搜索笔记

1. 点击侧边栏搜索框，或使用 `Ctrl+F`
2. 输入关键词搜索笔记名称、路径、内容
3. 点击搜索结果直接打开对应笔记

### 标签管理

1. 在编辑器顶部标签区域输入标签名称
2. 回车添加标签，点击标签 `×` 删除
3. 保存笔记时标签自动同步
4. 可通过 `/api/tags/search?tag=xxx` 搜索标签

### 重命名与拖拽

- **重命名**：右键笔记或目录，选择「重命名」，输入新名称
- **拖拽排序**：在目录树中拖拽笔记或目录调整顺序（同目录内）

### 删除笔记

在目录树上右键笔记或目录，选择「删除」。

### AI 助手

AI 助手集成任意 OpenAI 兼容大模型。点击编辑器右上角的魔法棒图标，或按 `Ctrl+L` 打开面板。

**首次配置** — 打开面板 →「配置」页 → 填写 API Key、Base URL、模型（如 `deepseek-chat`），可选填 `max_tokens`、温度、系统提示词 → 点击「保存」。客户端热重载，无需重启服务。

**行内自动补全（F1）** — 打开编辑器顶部的「AI」开关。停止输入 3 秒后，将最近 100 字符作为上下文请求补全，建议以浮层条展示。按 `Tab` 接受、`Esc` 拒绝。失败静默处理，绝不影响编辑。

**内容生成（F2）** — 在面板「生成」页输入提示词，点击「生成」。可将结果插入当前笔记，或另存为新笔记。

**总结所有笔记（F3）** — 点击侧边栏「总结」按钮。收集所有 `.md` 笔记（上限 100 篇），由 LLM 生成结构化总结并写入 `default/llm_summary.md`（重复调用会覆盖，前端二次确认）。

## REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/health` | 健康检查 |
| `GET` | `/api/tree?path=` | 获取目录树 |
| `GET` | `/api/note?path=` | 获取笔记内容 |
| `POST` | `/api/note` | 创建笔记或目录 |
| `PUT` | `/api/note?path=` | 更新笔记内容 |
| `DELETE` | `/api/note?path=` | 删除笔记或目录 |
| `GET` | `/api/search?keyword=` | 搜索笔记（名称、路径、内容） |
| `PUT` | `/api/rename?path=&newName=` | 重命名笔记或目录 |
| `POST` | `/api/sort` | 更新排序 `{path, sortOrder}` |
| `GET` | `/api/tags?path=` | 获取笔记标签 |
| `POST` | `/api/tags` | 添加标签 `{path, tag}` |
| `DELETE` | `/api/tags` | 删除标签 `{path, tag}` |
| `GET` | `/api/tags/search?tag=` | 按标签搜索 |
| `GET` | `/api/tags/all` | 获取所有标签 |
| `GET` | `/api/llm/config` | 获取 LLM 配置（api_key 脱敏） |
| `PUT` | `/api/llm/config` | 更新 LLM 配置（部分更新；热重载客户端） |
| `POST` | `/api/llm/complete` | 自动补全 `{text}` |
| `POST` | `/api/llm/generate` | 生成笔记内容 `{prompt}` |
| `POST` | `/api/llm/summarize` | 总结所有笔记 → `default/llm_summary.md` |

### 请求示例

```bash
# 获取目录树
curl http://localhost:8080/api/tree

# 获取笔记内容
curl "http://localhost:8080/api/note?path=default/示例笔记.md"

# 创建笔记
curl -X POST http://localhost:8080/api/note \
  -H "Content-Type: application/json" \
  -d '{"path":"default","name":"新笔记","is_dir":false,"content":"# 新笔记\n\n"}'

# 更新笔记
curl -X PUT "http://localhost:8080/api/note?path=default/新笔记.md" \
  -H "Content-Type: application/json" \
  -d '{"content":"# 更新后的内容\n\n"}'

# 删除笔记
curl -X DELETE "http://localhost:8080/api/note?path=default/新笔记.md"

# 搜索笔记
curl "http://localhost:8080/api/search?keyword=笔记"

# 重命名笔记
curl -X PUT "http://localhost:8080/api/rename?path=default/旧笔记.md&newName=新笔记"

# 添加标签
curl -X POST http://localhost:8080/api/tags \
  -H "Content-Type: application/json" \
  -d '{"path":"default/笔记.md","tag":"技术"}'

# 按标签搜索
curl "http://localhost:8080/api/tags/search?tag=技术"

# 更新 LLM 配置（部分更新；脱敏值 **** 会被忽略）
curl -X PUT http://localhost:8080/api/llm/config \
  -H "Content-Type: application/json" \
  -d '{"api_key":"sk-xxxx","base_url":"https://api.deepseek.com","model":"deepseek-chat","max_tokens":512,"temperature":0.7}'

# 自动补全
curl -X POST http://localhost:8080/api/llm/complete \
  -H "Content-Type: application/json" \
  -d '{"text":"# Vue3\n\n组合式 API"}'

# 生成笔记内容
curl -X POST http://localhost:8080/api/llm/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt":"写一篇 Vue3 响应式 API 的学习笔记"}'

# 总结所有笔记
curl -X POST http://localhost:8080/api/llm/summarize
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYNOTE_CONFIG` | 配置文件路径 | `config.yaml` |
| `MYNOTE_DATA_DIR` | 笔记数据目录（覆盖配置文件） | `./data` |
| `MYNOTE_DIST_DIR` | 前端静态文件目录 | `../frontend/dist/` |
| `MYNOTE_PORT` | 服务端口 | `8080` |
| `MYNOTE_LLM_DIR` | LLM 配置目录（secret_key.json, system_prompt.md） | `{data_dir}/llm/` |
| `GIN_MODE` | Gin 运行模式（`debug`/`release`） | `debug` |

## 存储配置

笔记数据支持多种存储后端，通过 `backend/config.yaml` 配置文件切换。详见 [storage/storage.md](file:///d:/workspace/mynote/backend/storage/storage.md)。

### 本地文件系统（默认）

```yaml
storage:
  type: local
  local:
    data_dir: ./data
```

### 对象存储（S3 兼容）

支持 AWS S3、MinIO、阿里云 OSS、腾讯云 COS 等：

```yaml
storage:
  type: oss
  oss:
    endpoint: "http://localhost:9000"
    access_key: "your-access-key"
    secret_key: "your-secret-key"
    bucket: "mynote"
    region: "us-east-1"
    prefix: "mynote/"
```

## LLM 配置

AI 助手集成任意 OpenAI 兼容大模型提供商（DeepSeek、OpenAI、Moonshot、智谱等）。配置存放在 `data/llm/` 目录，通过应用内配置面板（魔法棒图标 →「配置」页）管理，也可直接编辑文件。

### 配置文件

| 文件 | 说明 | 权限 |
|------|------|------|
| `data/llm/secret_key.json` | provider、api_key、base_url、model、max_tokens、temperature | `0600` |
| `data/llm/system_prompt.md` | 系统提示词（纯文本） | `0644` |

> **安全**：API Key 以明文存于 `secret_key.json`（文件权限 `0600`）。`GET /api/llm/config` 返回脱敏值（`****1234`）；更新时脱敏值会被忽略，避免覆盖真实密钥。`base_url` 必须以 `http://` 或 `https://` 开头。

### 模型参数

| 参数 | 默认值 | 范围 | 说明 |
|------|--------|------|------|
| `max_tokens` | 512 | [1, 8192] | 单次调用最大 token 数（补全 / 生成 / 总结通用） |
| `temperature` | 0.7 | (0, 2] | 采样温度 |

> **推理模型提示**（如 DeepSeek-reasoner）：建议设置较大的 `max_tokens`（如 2000+），否则推理阶段可能耗尽 token 预算，导致最终答案为空。

### `secret_key.json` 示例

```json
{
  "provider": "openai-compatible",
  "api_key": "sk-xxxxxxxxxxxxxxxx",
  "base_url": "https://api.deepseek.com",
  "model": "deepseek-chat",
  "max_tokens": 512,
  "temperature": 0.7
}
```

## 云端部署

1. **构建**：运行 `.\scripts\build.ps1` 生成部署包
2. **上传**：将 `build/` 目录内容上传到服务器
3. **配置**：通过环境变量配置数据目录、端口等
4. **运行**：执行 `mynote-server`（Linux）或 `mynote-server.exe`（Windows）

### Linux 部署示例

```bash
# 交叉编译 Linux 版本
cd backend
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o mynote-server .

# 服务器上运行
export MYNOTE_DATA_DIR=/data/notes
export MYNOTE_DIST_DIR=/app/dist
export MYNOTE_PORT=80
./mynote-server
```

## 项目结构

```
mynote/
├── backend/                # Go 后端
│   ├── main.go            # 入口，路由，静态文件服务，配置加载
│   ├── config.yaml        # 存储配置文件
│   ├── api/
│   │   ├── handler.go     # REST API 处理器（笔记/标签/搜索）
│   │   └── llm_handler.go # LLM API 处理器（配置/补全/生成/总结）
│   ├── service/
│   │   ├── note_service.go # 笔记服务（依赖 Storage 接口）
│   │   └── llm_service.go  # LLM 业务编排（补全/生成/总结）
│   ├── llm/                # LLM 客户端层
│   │   ├── llm.go         # LLMClient 接口 + 请求/响应结构体
│   │   ├── config.go      # 配置结构体、读写、脱敏、校验、NewClient 工厂
│   │   ├── openai_compat.go # OpenAI 兼容客户端（DeepSeek/OpenAI/Moonshot）
│   │   └── llm_design.md  # LLM 设计文档
│   ├── storage/            # 可插拔存储层
│   │   ├── storage.go     # Storage 接口定义
│   │   ├── config.go      # 配置结构体
│   │   ├── factory.go     # 工厂函数 + 配置加载
│   │   ├── local.go       # 本地文件系统实现
│   │   ├── oss.go         # 对象存储实现（S3 兼容）
│   │   └── storage.md     # 存储层文档
│   ├── models/
│   │   ├── note.go        # 笔记数据模型
│   │   └── llm.go         # LLM 请求/响应模型
│   └── data/              # 笔记文件存储目录（本地存储模式）
│       └── llm/           # LLM 运行时配置目录（secret_key.json, system_prompt.md）
├── frontend/               # Vue 前端
│   └── src/
│       ├── App.vue        # 根组件（含侧边栏切换、AI 面板挂载）
│       ├── components/
│       │   ├── Sidebar.vue     # 侧边栏（目录树+右键菜单+总结按钮）
│       │   ├── NoteEditor.vue  # Markdown 编辑器（含 F1 自动补全）
│       │   └── LLMPanel.vue    # AI 助手面板（配置/生成）
│       └── api/index.js  # API 请求封装
├── scripts/
│   ├── dev.ps1            # 开发模式启动
│   └── build.ps1          # 生产构建打包
├── start-dev.bat          # 一键开发启动
└── AGENT.md               # AI Agent 项目说明
```

## License

MIT
