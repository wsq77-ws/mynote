# MyNote - 在线知识笔记系统

一个基于 Markdown 的本地知识笔记系统，支持层级目录结构，提供所见即所得的编辑体验。

## 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | Vue 3 + Vite + Element Plus + md-editor-v3 |
| **后端** | Go 1.23+ / Gin |
| **存储** | 可插拔存储层：本地文件系统 / 对象存储（S3 兼容） |

## 项目结构

```
mynote/
├── backend/                # Go 后端
│   ├── main.go            # 入口，路由，静态文件服务，配置加载
│   ├── config.yaml        # 存储配置文件（本地/对象存储切换）
│   ├── api/
│   │   └── handler.go     # REST API 处理器
│   ├── service/
│   │   └── note_service.go # 笔记服务（依赖 Storage 接口）
│   ├── storage/            # 可插拔存储层
│   │   ├── storage.go     # Storage 接口定义
│   │   ├── config.go      # 配置结构体
│   │   ├── factory.go     # 工厂函数 + 配置加载
│   │   ├── local.go       # 本地文件系统实现
│   │   ├── oss.go         # 对象存储实现（S3 兼容）
│   │   └── storage.md     # 存储层文档
│   ├── models/
│   │   └── note.go        # 数据模型
│   ├── data/              # 笔记文件存储目录（本地存储模式）
│   └── go.mod / go.sum
├── frontend/               # Vue 前端
│   ├── src/
│   │   ├── App.vue             # 根组件
│   │   ├── main.js             # 入口
│   │   ├── style.css           # 全局样式
│   │   ├── components/
│   │   │   ├── Sidebar.vue     # 侧边栏（目录树+右键菜单）
│   │   │   └── NoteEditor.vue  # Markdown 编辑器
│   │   └── api/
│   │       └── index.js        # API 请求封装
│   ├── index.html
│   ├── vite.config.js
│   └── package.json
├── scripts/
│   ├── dev.ps1            # 开发模式启动（PowerShell）
│   └── build.ps1          # 生产构建打包（PowerShell）
├── start-dev.bat          # 一键开发启动（Batch）
├── .gitignore
└── AGENT.md
```

## REST API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/health` | 健康检查 |
| `GET` | `/api/tree` | 获取目录树 |
| `GET` | `/api/note` | 获取笔记内容 |
| `POST` | `/api/note` | 创建笔记或目录 |
| `PUT` | `/api/note` | 更新笔记内容 |
| `DELETE` | `/api/note` | 删除笔记或目录 |

### 接口参数详情

#### `GET /api/health`
健康检查，无参数。

**响应**: `{ "status": "ok" }`

---

#### `GET /api/tree`
获取目录树（递归返回所有子节点）。

| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `path` | query | string | 否 | `.`（根目录） | 要获取的目录相对路径 |

**响应 Data**: `TreeNode[]`，每个节点包含 `name`、`path`、`type`（`file`/`directory`）、`children`。

---

#### `GET /api/note`
获取指定笔记的内容。

| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `path` | query | string | **是** | - | 笔记相对路径，如 `default/示例笔记.md`。可不带 `.md` 后缀 |

**响应 Data**: `Note` 对象，包含 `path`、`name`、`content`、`updated_at`。

---

#### `POST /api/note`
创建笔记或目录。

| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `path` | body | string | 否 | `default` | 父目录相对路径。为空时自动使用 `default` 目录 |
| `name` | body | string | **是** | - | 笔记或目录名称（笔记可不带 `.md` 后缀） |
| `is_dir` | body | bool | 否 | `false` | `true` 创建目录，`false` 创建笔记 |
| `content` | body | string | 否 | `""` | 笔记初始内容（创建目录时忽略） |

**请求示例**:
```json
{
  "path": "default",
  "name": "新笔记",
  "is_dir": false,
  "content": "# 新笔记\n\n"
}
```

---

#### `PUT /api/note`
更新笔记内容。

| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `path` | query | string | **是** | - | 笔记相对路径 |
| `content` | body | string | **是** | - | 笔记新内容 |

**请求 Body**:
```json
{ "content": "# 更新后的内容\n\n" }
```

---

#### `DELETE /api/note`
删除笔记或目录（目录会递归删除）。

| 参数名 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|------|--------|------|
| `path` | query | string | **是** | - | 要删除的笔记或目录相对路径 |

### 通用响应格式

所有接口返回统一结构：
```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

| code | 含义 |
|------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 前端开发规则

> **重要**：每次做前端改动时，都必须检查传入后端 API 的参数是否合理！

### 检查清单

在修改前端代码（尤其是涉及 API 调用的部分）时，必须逐项确认：

1. **参数完整性** — 对照上方接口参数表，确认必填参数都已传值，不能为 `undefined` 或空字符串（除非接口明确允许空值）
2. **参数类型** — 确认参数类型与接口定义一致（如 `is_dir` 是 bool 而非 string）
3. **参数位置** — 确认参数放在正确的位置（query 参数 vs body 参数），参考 `src/api/index.js` 中的封装
4. **默认值处理** — 可选参数如需使用默认值，确认是否应该传空字符串还是不传该字段
5. **路径格式** — `path` 参数使用 `/` 分隔的相对路径（如 `default/子目录/笔记.md`），不要以 `/` 开头
6. **创建笔记时** — `path` 字段不能为空字符串，前端应始终携带目标目录（默认 `default`），避免后端因校验失败

### 前端 API 封装位置

所有后端 API 调用封装在 [src/api/index.js](file:///d:/workspace/mynote/frontend/src/api/index.js)，修改接口调用时优先更新此文件。

## 后端开发规则

> **重要**：每次做后端改动时，都必须确认存储层接口的一致性！

### 检查清单

1. **Storage 接口** — 所有笔记操作必须通过 `Storage` 接口，不能直接调用 `os.ReadFile`/`os.WriteFile` 等文件系统 API
2. **路径格式** — 传给 Storage 接口的路径使用 `/` 分隔的相对路径（如 `default/子目录/笔记.md`），不要使用 `filepath.Join`
3. **新增存储后端** — 必须实现 `Storage` 接口的所有方法，并在 `factory.go` 中注册
4. **配置变更** — 修改配置结构体时，同步更新 `config.yaml` 和 `storage.md` 文档
5. **业务逻辑与存储分离** — `note_service.go` 只包含业务逻辑，不包含存储实现细节

## 快速开始

### 开发模式

前端和后端分开运行，支持热重载。

**方式一（推荐）：一键启动**
```bash
双击 start-dev.bat
```

**方式二：分别启动**
```bash
# 终端1 - 后端
cd backend && go run main.go

# 终端2 - 前端
cd frontend && npm run dev
```

开发模式下：前端访问 `http://localhost:3000`，API 代理到 `http://localhost:8080`。

### 生产构建

```bash
# 使用构建脚本
.\scripts\build.ps1

# 或手动构建
cd frontend && npm run build
cd backend && go build -o mynote-server.exe .
```

构建产物在 `build/` 目录，双击 `build/start.bat` 或直接运行 `mynote-server.exe` 即可。

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYNOTE_CONFIG` | 配置文件路径 | `config.yaml` |
| `MYNOTE_DATA_DIR` | 笔记数据目录（覆盖配置文件中的 `local.data_dir`） | `./data` |
| `MYNOTE_DIST_DIR` | 前端静态文件目录 | `../frontend/dist/` |
| `MYNOTE_PORT` | 服务端口 | `8080` |
| `GIN_MODE` | Gin 运行模式 | `debug` |

## 存储层

存储层提供统一的 `Storage` 接口，支持多种存储后端，通过 `backend/config.yaml` 切换。详见 [storage/storage.md](file:///d:/workspace/mynote/backend/storage/storage.md)。

### 配置文件

```yaml
# 本地文件系统（默认）
storage:
  type: local
  local:
    data_dir: ./data

# 对象存储（S3 兼容）
storage:
  type: oss
  oss:
    endpoint: "http://localhost:9000"
    access_key: "xxx"
    secret_key: "xxx"
    bucket: "mynote"
    region: "us-east-1"
    prefix: "mynote/"
```

### 支持的存储后端

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| `local` | 本地文件系统 | 本地开发、单机部署 |
| `oss` | 对象存储（S3 兼容） | 云端部署、分布式场景 |

### 扩展存储后端

1. 在 `storage/` 下创建新文件实现 `Storage` 接口
2. 在 `factory.go` 的 `New()` 中添加对应 `case`
3. 在 `config.go` 中添加配置结构体

## 云端部署

项目已为云端部署做好准备：

1. **构建**：运行 `.\scripts\build.ps1` 生成部署包
2. **配置**：通过环境变量 `MYNOTE_DATA_DIR` 和 `MYNOTE_DIST_DIR` 配置目录
3. **部署**：将 `build/` 目录复制到服务器，设置环境变量后运行 `mynote-server.exe`
4. **Docker**（可选）：后端二进制文件 + `dist/` 目录即可运行，无额外依赖

### 示例部署命令

```bash
# Linux 服务器
export MYNOTE_DATA_DIR=/data/notes
export MYNOTE_DIST_DIR=/app/dist
export MYNOTE_PORT=80
./mynote-server
```
