# MyNote - 在线知识笔记系统

一个基于 Markdown 的本地知识笔记系统，支持层级目录结构，提供所见即所得的编辑体验。

## 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | Vue 3 + Vite + Element Plus + md-editor-v3 |
| **后端** | Go 1.23+ / Gin |
| **存储** | 本地文件系统（`.md` 文件） |

## 项目结构

```
mynote/
├── backend/                # Go 后端
│   ├── main.go            # 入口，路由，静态文件服务
│   ├── api/
│   │   └── handler.go     # REST API 处理器
│   ├── service/
│   │   └── note_service.go # 笔记服务（文件存储逻辑）
│   ├── models/
│   │   └── note.go        # 数据模型
│   ├── data/              # 笔记文件存储目录
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
| `GET` | `/api/tree?path=` | 获取目录树 |
| `GET` | `/api/note?path=` | 获取笔记内容 |
| `POST` | `/api/note` | 创建笔记或目录 |
| `PUT` | `/api/note?path=` | 更新笔记内容 |
| `DELETE` | `/api/note?path=` | 删除笔记或目录 |

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
| `MYNOTE_DATA_DIR` | 笔记数据目录 | `backend/data/` |
| `MYNOTE_DIST_DIR` | 前端静态文件目录 | `../frontend/dist/` |
| `MYNOTE_PORT` | 服务端口 | `8080` |
| `GIN_MODE` | Gin 运行模式 | `debug` |

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
