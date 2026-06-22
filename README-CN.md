# MyNote - 在线知识笔记系统

一个基于 Markdown 的本地知识笔记系统，支持层级目录结构，提供所见即所得的编辑体验。当前运行在本地，已为云端部署做好准备。

## 功能特性

- **Markdown 编辑** — 基于 md-editor-v3，支持代码高亮、表格、列表等丰富语法
- **层级目录树** — 左侧目录树支持无限层级嵌套，目录/笔记分类管理
- **自动保存** — 编辑内容 2 秒自动保存，也可手动保存
- **右键菜单** — 目录树支持右键新建笔记/目录、删除节点
- **实时预览** — 所见即所得的编辑体验
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

### 编辑笔记

1. 在左侧目录树点击任意笔记文件
2. 在右侧编辑器中编写 Markdown 内容
3. 内容会在停止输入 2 秒后自动保存，也可点击「保存」按钮手动保存

### 删除笔记

在目录树上右键笔记或目录，选择「删除」。

## REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/health` | 健康检查 |
| `GET` | `/api/tree?path=` | 获取目录树 |
| `GET` | `/api/note?path=` | 获取笔记内容 |
| `POST` | `/api/note` | 创建笔记或目录 |
| `PUT` | `/api/note?path=` | 更新笔记内容 |
| `DELETE` | `/api/note?path=` | 删除笔记或目录 |

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
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYNOTE_CONFIG` | 配置文件路径 | `config.yaml` |
| `MYNOTE_DATA_DIR` | 笔记数据目录（覆盖配置文件） | `./data` |
| `MYNOTE_DIST_DIR` | 前端静态文件目录 | `../frontend/dist/` |
| `MYNOTE_PORT` | 服务端口 | `8080` |
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
│   ├── api/handler.go     # REST API 处理器
│   ├── service/note_service.go # 笔记服务（依赖 Storage 接口）
│   ├── storage/            # 可插拔存储层
│   │   ├── storage.go     # Storage 接口定义
│   │   ├── config.go      # 配置结构体
│   │   ├── factory.go     # 工厂函数 + 配置加载
│   │   ├── local.go       # 本地文件系统实现
│   │   ├── oss.go         # 对象存储实现（S3 兼容）
│   │   └── storage.md     # 存储层文档
│   ├── models/note.go     # 数据模型
│   └── data/              # 笔记文件存储目录（本地存储模式）
├── frontend/               # Vue 前端
│   └── src/
│       ├── App.vue        # 根组件
│       ├── components/
│       │   ├── Sidebar.vue     # 侧边栏（目录树+右键菜单）
│       │   └── NoteEditor.vue  # Markdown 编辑器
│       └── api/index.js  # API 请求封装
├── scripts/
│   ├── dev.ps1            # 开发模式启动
│   └── build.ps1          # 生产构建打包
├── start-dev.bat          # 一键开发启动
└── AGENT.md               # AI Agent 项目说明
```

## License

MIT
