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
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MYNOTE_CONFIG` | Config file path | `config.yaml` |
| `MYNOTE_DATA_DIR` | Note data directory (overrides config file) | `./data` |
| `MYNOTE_DIST_DIR` | Frontend static files directory | `../frontend/dist/` |
| `MYNOTE_PORT` | Service port | `8080` |
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
│   ├── api/handler.go     # REST API handlers
│   ├── service/note_service.go # Note service (depends on Storage interface)
│   ├── storage/            # Pluggable storage layer
│   │   ├── storage.go     # Storage interface definition
│   │   ├── config.go      # Config structs
│   │   ├── factory.go     # Factory function + config loading
│   │   ├── local.go       # Local filesystem implementation
│   │   ├── oss.go         # Object storage implementation (S3 compatible)
│   │   └── storage.md     # Storage layer docs
│   ├── models/note.go     # Data models
│   └── data/              # Note file storage directory (local storage mode)
├── frontend/               # Vue frontend
│   └── src/
│       ├── App.vue        # Root component
│       ├── components/
│       │   ├── Sidebar.vue     # Sidebar (directory tree + context menu)
│       │   └── NoteEditor.vue  # Markdown editor
│       └── api/index.js  # API request wrapper
├── scripts/
│   ├── dev.ps1            # Development mode startup
│   └── build.ps1          # Production build packaging
├── start-dev.bat          # One-click development startup
└── AGENT.md               # AI Agent project documentation
```

## License

MIT
