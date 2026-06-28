# MyNote - Online Knowledge Note System

A Markdown-based local knowledge note system with hierarchical directory structure and WYSIWYG editing experience. Currently runs locally, with cloud deployment readiness built in.

[中文文档](README-CN.md)

## Features

- **Markdown Editing** — Powered by md-editor-v3, supports code highlighting, tables, lists, and more
- **Hierarchical Directory Tree** — Sidebar tree with unlimited nesting for organizing notes and folders
- **Auto Save** — Content is automatically saved 2 seconds after editing stops, with manual save option
- **Offline Save** — Automatically caches to localStorage when offline, auto-syncs when back online
- **Context Menu** — Right-click on the tree to create notes/directories, delete nodes, or rename
- **Live Preview** — WYSIWYG editing experience
- **Keyboard Shortcuts** — `Ctrl+S` to save, `Ctrl+F` to search, `Ctrl+N` to create new note
- **Word Count** — Real-time display of word count, line count, and estimated reading time
- **Global Search** — Search note names, paths, content, **and tags**; results display match type and highlighted tags
- **Tag System** — Add tags to notes, categorize and search by tags; newly added tags are immediately searchable
- **Drag to Sort** — Drag notes and folders within the same directory to reorder
- **AI Assistant** — Integrates OpenAI-compatible LLMs (DeepSeek / OpenAI / Moonshot / Zhipu): inline autocomplete (Tab to accept / Esc to dismiss), content generation, one-click summarization of all notes, and a configurable panel (API key, model, `max_tokens`, `temperature`, system prompt). See [LLM Configuration](#llm-configuration)
- **Pluggable Storage** — Supports local filesystem and object storage (S3 compatible), switchable via config file
- **One-Click Deployment** — In production mode, the backend serves frontend static files on a single port

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Frontend** | Vue 3 + Vite + Element Plus + md-editor-v3 |
| **Backend** | Go 1.23+ / Gin |
| **Storage** | Pluggable storage layer: local filesystem / object storage (S3 compatible) |

## Prerequisites

- [Node.js](https://nodejs.org/) 18+ (includes npm)
- [Go](https://go.dev/dl/) 1.23+

## Quick Start

### Development Mode

Frontend and backend run separately with hot reload.

**Option 1: One-click startup (recommended)**

Double-click `start-dev.bat` in the project root.

**Option 2: Start separately**

```bash
# Terminal 1 - Backend
cd backend
go run main.go

# Terminal 2 - Frontend
cd frontend
npm install
npm run dev
```

Access after startup:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080

> In development mode, the frontend proxies `/api` requests to the backend on port 8080 via Vite.

### Production Build

```bash
# Using build script (PowerShell)
.\scripts\build.ps1

# Or build manually
cd frontend && npm run build
cd backend && go build -o mynote-server.exe .
```

Build output is in the `build/` directory:
- `mynote-server.exe` — Backend executable
- `dist/` — Frontend static files
- `data/` — Note data directory
- `start.bat` — Startup script

To run: double-click `build/start.bat`, or execute `mynote-server.exe` directly, then visit http://localhost:8080.

## Usage Guide

### Create a Note

1. Click the "New" button at the top of the sidebar to create a note in the default directory
2. Or right-click a directory in the tree and select "New Note" / "New Directory"
3. Use shortcut `Ctrl+N` to quickly create a new note

### Edit a Note

1. Click any note file in the left sidebar tree
2. Write Markdown content in the right editor
3. Content auto-saves 2 seconds after you stop typing, or:
   - Click "Save" button manually
   - Use shortcut `Ctrl+S` to save

### Offline Save

When the network is unavailable, the editor automatically switches to offline mode:

- **Auto fallback**: If saving to server fails, content is cached to browser localStorage
- **Status indicator**: "Offline" or "Unsynced" tag shown next to the title, status bar at the bottom
- **Auto sync**: When network recovers, all offline changes are automatically synced to the server
- **Manual sync**: Click the "Sync" button next to the title to trigger sync manually
- **Cache priority**: When loading a note, cached version takes priority if unsynced changes exist

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+S` | Save current note |
| `Ctrl+F` | Open search box |
| `Ctrl+N` | Create new note |
| `Ctrl+B` | Toggle sidebar |
| `Ctrl+L` | Toggle AI assistant panel |

### Word Count

The editor footer shows real-time stats:
- Word count (Chinese characters, English words)
- Line count
- Estimated reading time (200 words/min)

### Search Notes

1. Click the search box in the sidebar, or use `Ctrl+F`
2. Enter keywords to search note names, paths, and content
3. Click search results to open the corresponding note

### Tag Management

1. Enter tag name in the tag input area at the top of the editor
2. Press Enter to add tag, click `×` on tag to remove
3. Tags are synced automatically when saving
4. Search by tag via `/api/tags/search?tag=xxx`

### Rename & Drag to Sort

- **Rename**: Right-click a note or directory, select "Rename", enter new name
- **Drag to Sort**: Drag notes or folders within the same directory to reorder

### Delete a Note

Right-click a note or directory in the tree and select "Delete".

### AI Assistant

The AI assistant integrates any OpenAI-compatible LLM. Open the panel via the magic-wand icon (top-right of the editor) or `Ctrl+L`.

**First-time setup** — open the panel → "Config" tab → fill in API Key, Base URL, Model (e.g. `deepseek-chat`), and optionally `max_tokens`, `temperature`, and a system prompt → click "Save". The client is hot-reloaded without restarting the server.

**Inline autocomplete (F1)** — toggle the "AI" switch in the editor header. After you stop typing for 3s, the last 100 characters are sent as context and a suggestion appears as a floating bar. Press `Tab` to accept or `Esc` to dismiss. Failures are silent and never block editing.

**Generate content (F2)** — in the panel's "Generate" tab, enter a prompt and click "Generate". Insert the result into the current note, or save it as a new note.

**Summarize all notes (F3)** — click the "Summarize" button in the sidebar. It collects all `.md` notes (up to 100), asks the LLM for a structured summary, and writes it to `default/llm_summary.md` (overwrites on repeat; a confirmation dialog is shown).

## REST API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/tree?path=` | Get directory tree |
| `GET` | `/api/note?path=` | Get note content |
| `POST` | `/api/note` | Create note or directory |
| `PUT` | `/api/note?path=` | Update note content |
| `DELETE` | `/api/note?path=` | Delete note or directory |
| `GET` | `/api/search?keyword=` | Search notes (name, path, content) |
| `PUT` | `/api/rename?path=&newName=` | Rename note or directory |
| `POST` | `/api/sort` | Update sort order `{path, sortOrder}` |
| `GET` | `/api/tags?path=` | Get note tags |
| `POST` | `/api/tags` | Add tag `{path, tag}` |
| `DELETE` | `/api/tags` | Remove tag `{path, tag}` |
| `GET` | `/api/tags/search?tag=` | Search by tag |
| `GET` | `/api/tags/all` | Get all tags |
| `GET` | `/api/llm/config` | Get LLM config (api_key masked) |
| `PUT` | `/api/llm/config` | Update LLM config (partial; hot-reload client) |
| `POST` | `/api/llm/complete` | Inline autocomplete `{text}` |
| `POST` | `/api/llm/generate` | Generate note content `{prompt}` |
| `POST` | `/api/llm/summarize` | Summarize all notes → `default/llm_summary.md` |

### Request Examples

```bash
# Get directory tree
curl http://localhost:8080/api/tree

# Get note content
curl "http://localhost:8080/api/note?path=default/welcome.md"

# Create a note
curl -X POST http://localhost:8080/api/note \
  -H "Content-Type: application/json" \
  -d '{"path":"default","name":"new-note","is_dir":false,"content":"# New Note\n\n"}'

# Update a note
curl -X PUT "http://localhost:8080/api/note?path=default/new-note.md" \
  -H "Content-Type: application/json" \
  -d '{"content":"# Updated content\n\n"}'

# Delete a note
curl -X DELETE "http://localhost:8080/api/note?path=default/new-note.md"

# Search notes
curl "http://localhost:8080/api/search?keyword=note"

# Rename a note
curl -X PUT "http://localhost:8080/api/rename?path=default/old-note.md&newName=new-note"

# Add a tag
curl -X POST http://localhost:8080/api/tags \
  -H "Content-Type: application/json" \
  -d '{"path":"default/note.md","tag":"tech"}'

# Search by tag
curl "http://localhost:8080/api/tags/search?tag=tech"

# Update LLM config (partial; api_key masked value **** is ignored)
curl -X PUT http://localhost:8080/api/llm/config \
  -H "Content-Type: application/json" \
  -d '{"api_key":"sk-xxxx","base_url":"https://api.deepseek.com","model":"deepseek-chat","max_tokens":512,"temperature":0.7}'

# Inline autocomplete
curl -X POST http://localhost:8080/api/llm/complete \
  -H "Content-Type: application/json" \
  -d '{"text":"# Vue3\n\nComposition API"}'

# Generate note content
curl -X POST http://localhost:8080/api/llm/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt":"Write a study note on Vue3 reactive APIs"}'

# Summarize all notes
curl -X POST http://localhost:8080/api/llm/summarize
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MYNOTE_CONFIG` | Config file path | `config.yaml` |
| `MYNOTE_DATA_DIR` | Note data directory (overrides config file) | `./data` |
| `MYNOTE_DIST_DIR` | Frontend static files directory | `../frontend/dist/` |
| `MYNOTE_PORT` | Service port | `8080` |
| `MYNOTE_LLM_DIR` | LLM config directory (secret_key.json, system_prompt.md) | `{data_dir}/llm/` |
| `GIN_MODE` | Gin run mode (`debug`/`release`) | `debug` |

## Storage Configuration

Note data supports multiple storage backends, switchable via `backend/config.yaml`. See [storage/storage.md](file:///d:/workspace/mynote/backend/storage/storage.md) for details.

### Local Filesystem (default)

```yaml
storage:
  type: local
  local:
    data_dir: ./data
```

### Object Storage (S3 Compatible)

Supports AWS S3, MinIO, Alibaba Cloud OSS, Tencent Cloud COS, etc.:

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

## LLM Configuration

The AI assistant integrates any OpenAI-compatible LLM provider (DeepSeek, OpenAI, Moonshot, Zhipu, etc.). Configuration is stored in the `data/llm/` directory and managed via the in-app config panel (magic-wand icon → "Config" tab), or by editing the files directly.

### Config Files

| File | Description | Permissions |
|------|-------------|-------------|
| `data/llm/secret_key.json` | provider, api_key, base_url, model, max_tokens, temperature | `0600` |
| `data/llm/system_prompt.md` | System prompt (plain text) | `0644` |

> **Security**: The API key is stored in plaintext in `secret_key.json` (file permission `0600`). `GET /api/llm/config` returns a masked key (`****1234`); the masked value is ignored on update to prevent overwriting the real key. `base_url` must start with `http://` or `https://`.

### Model Parameters

| Parameter | Default | Range | Description |
|-----------|---------|-------|-------------|
| `max_tokens` | 512 | [1, 8192] | Max tokens per call (applies to autocomplete / generate / summarize) |
| `temperature` | 0.7 | (0, 2] | Sampling temperature |

> **Tip for reasoning models** (e.g. DeepSeek-reasoner): set a larger `max_tokens` (e.g. 2000+), otherwise the reasoning phase may exhaust the token budget and leave the final answer empty.

### Example `secret_key.json`

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

## Cloud Deployment

1. **Build**: Run `.\scripts\build.ps1` to generate the deployment package
2. **Upload**: Upload the contents of `build/` to your server
3. **Configure**: Set data directory, port, etc. via environment variables
4. **Run**: Execute `mynote-server` (Linux) or `mynote-server.exe` (Windows)

### Linux Deployment Example

```bash
# Cross-compile for Linux
cd backend
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o mynote-server .

# Run on server
export MYNOTE_DATA_DIR=/data/notes
export MYNOTE_DIST_DIR=/app/dist
export MYNOTE_PORT=80
./mynote-server
```

## Project Structure

```
mynote/
├── backend/                # Go backend
│   ├── main.go            # Entry point, routing, static file serving, config loading
│   ├── config.yaml        # Storage configuration file
│   ├── api/
│   │   ├── handler.go     # REST API handlers (notes/tags/search)
│   │   └── llm_handler.go # LLM API handlers (config/complete/generate/summarize)
│   ├── service/
│   │   ├── note_service.go # Note service (depends on Storage interface)
│   │   └── llm_service.go  # LLM orchestration (autocomplete/generate/summarize)
│   ├── llm/                # LLM client layer
│   │   ├── llm.go         # LLMClient interface + request/response structs
│   │   ├── config.go      # Config struct, read/write, masking, validation, NewClient factory
│   │   ├── openai_compat.go # OpenAI-compatible client (DeepSeek/OpenAI/Moonshot)
│   │   └── llm_design.md  # LLM module design doc
│   ├── storage/            # Pluggable storage layer
│   │   ├── storage.go     # Storage interface definition
│   │   ├── config.go      # Config structs
│   │   ├── factory.go     # Factory function + config loading
│   │   ├── local.go       # Local filesystem implementation
│   │   ├── oss.go         # Object storage implementation (S3 compatible)
│   │   └── storage.md     # Storage layer docs
│   ├── models/
│   │   ├── note.go        # Note data models
│   │   └── llm.go         # LLM request/response models
│   └── data/              # Note file storage directory (local storage mode)
│       └── llm/           # LLM config (secret_key.json, system_prompt.md)
├── frontend/               # Vue frontend
│   └── src/
│       ├── App.vue        # Root component
│       ├── components/
│       │   ├── Sidebar.vue     # Sidebar (directory tree + context menu + summarize)
│       │   ├── NoteEditor.vue  # Markdown editor (with AI autocomplete)
│       │   └── LLMPanel.vue    # AI assistant panel (generate + config)
│       └── api/index.js  # API request wrapper
├── scripts/
│   ├── dev.ps1            # Development mode startup
│   └── build.ps1          # Production build packaging
├── start-dev.bat          # One-click development startup
└── AGENT.md               # AI Agent project documentation
```

## License

MIT
