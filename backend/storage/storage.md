# Storage 存储层

存储层提供统一的笔记存储抽象，支持多种存储后端。通过 `config.yaml` 配置文件切换存储方式，无需修改业务代码。

## 架构设计

```
┌─────────────────────────────────────────────┐
│              service/note_service.go         │
│         (业务逻辑，依赖 Storage 接口)          │
└──────────────────┬──────────────────────────┘
                   │ Storage 接口
┌──────────────────▼──────────────────────────┐
│            storage/storage.go                │
│              Storage 接口定义                 │
└──┬───────────────────────────┬──────────────┘
   │                           │
   │ LocalStorage              │ OSSStorage
   │ (本地文件系统)              │ (对象存储/S3)
   ▼                           ▼
┌──────────────┐    ┌─────────────────────────┐
│ local.go     │    │ oss.go                  │
│ os.ReadDir   │    │ AWS SDK v2 (S3)         │
│ os.ReadFile  │    │ MinIO / Aliyun OSS      │
│ os.WriteFile │    │ Tencent COS / AWS S3    │
└──────────────┘    └─────────────────────────┘
```

## 文件说明

| 文件 | 说明 |
|------|------|
| [storage.go](storage.go) | `Storage` 接口定义和 `Entry` 结构体 |
| [config.go](config.go) | 配置结构体定义（`Config`、`StorageConfig`、`LocalConfig`、`OSSConfig`） |
| [factory.go](factory.go) | 工厂函数 `New()` 和配置加载函数 `LoadConfig()` |
| [local.go](local.go) | 本地文件系统存储实现 `LocalStorage` |
| [oss.go](oss.go) | 对象存储实现 `OSSStorage`（兼容 S3 协议） |

## Storage 接口

所有存储后端都需要实现以下接口：

```go
type Storage interface {
    // List 列出指定目录下的条目（非递归）
    List(dirPath string) ([]Entry, error)

    // Read 读取文件内容，返回文件内容和修改时间
    Read(path string) (content string, modTime time.Time, err error)

    // Write 写入文件内容
    Write(path string, content string) error

    // Mkdir 创建目录
    Mkdir(path string) error

    // Delete 删除文件或目录（目录递归删除）
    Delete(path string) error

    // GetModTime 获取文件修改时间
    GetModTime(path string) (time.Time, error)

    // Exists 检查路径是否存在
    Exists(path string) (bool, error)

    // Type 返回存储类型标识
    Type() string
}
```

## 配置

配置文件位于 `backend/config.yaml`，通过修改 `storage.type` 字段切换存储后端。

### 本地文件系统（默认）

```yaml
storage:
  type: local
  local:
    data_dir: ./data
```

### 对象存储

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
    use_ssl: true
```

### 配置查找顺序

1. 环境变量 `MYNOTE_CONFIG` 指定的路径
2. 当前工作目录下的 `config.yaml`
3. 可执行文件目录下的 `config.yaml`
4. 以上都不存在则使用默认配置（本地文件系统，`./data` 目录）

### 环境变量覆盖

| 变量 | 说明 | 优先级 |
|------|------|--------|
| `MYNOTE_CONFIG` | 指定配置文件路径 | 最高 |
| `MYNOTE_DATA_DIR` | 覆盖本地存储数据目录 | 覆盖配置文件中的 `local.data_dir` |
| `MYNOTE_PORT` | 服务端口 | 独立配置 |
| `MYNOTE_DIST_DIR` | 前端静态文件目录 | 独立配置 |

## 存储后端说明

### LocalStorage（本地文件系统）

- 笔记以 `.md` 文件形式存储在指定目录
- 目录结构直接映射文件系统目录
- 适合本地开发和单机部署

### OSSStorage（对象存储）

- 兼容 S3 协议，支持多种对象存储服务：
  - **AWS S3** — 亚马逊对象存储
  - **MinIO** — 自建对象存储
  - **阿里云 OSS** — 使用 S3 兼容模式
  - **腾讯云 COS** — 使用 S3 兼容模式
- 对象存储没有真正的目录概念，通过前缀模拟目录结构
- `Mkdir` 创建以 `/` 结尾的空对象作为目录占位符
- `Delete` 删除目录时会递归删除该前缀下所有对象
- 适合云端部署和分布式场景

## 扩展新的存储后端

1. 在 `storage/` 目录下创建新文件，如 `database.go`
2. 实现 `Storage` 接口的所有方法
3. 在 `factory.go` 的 `New()` 函数中添加新的 `case` 分支
4. 在 `config.go` 中添加对应的配置结构体
5. 在 `config.yaml` 中添加对应的配置项

示例：

```go
// storage/database.go
package storage

type DatabaseStorage struct {
    // ...
}

func NewDatabaseStorage(cfg DatabaseConfig) (*DatabaseStorage, error) {
    // ...
}

// 实现 Storage 接口的所有方法...
```

```go
// storage/factory.go - 在 New() 中添加
case "database":
    return NewDatabaseStorage(cfg.Storage.Database)
```
