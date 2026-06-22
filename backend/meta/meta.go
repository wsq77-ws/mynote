package meta

import "time"

// NoteMeta 笔记元数据
type NoteMeta struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`       // 相对路径，如 default/笔记.md
	Name      string    `json:"name"`       // 笔记名称（不含 .md 后缀）
	IsDir     bool      `json:"is_dir"`     // 是否为目录
	Author    string    `json:"author"`     // 作者
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// Meta 元数据管理接口
type Meta interface {
	// Init 初始化数据库（建表等）
	Init() error

	// Close 关闭数据库连接
	Close() error

	// UpsertNote 创建或更新笔记元数据
	UpsertNote(meta *NoteMeta) error

	// GetNoteByPath 根据路径获取笔记元数据
	GetNoteByPath(path string) (*NoteMeta, error)

	// GetAllNotes 获取所有笔记和目录的元数据
	GetAllNotes() ([]NoteMeta, error)

	// GetChildren 获取指定目录下的子条目元数据
	GetChildren(dirPath string) ([]NoteMeta, error)

	// DeleteNote 删除笔记元数据（支持递归删除目录下的所有条目）
	DeleteNote(path string) error

	// Search 搜索笔记（按名称或路径模糊匹配）
	Search(keyword string) ([]NoteMeta, error)

	// SyncFromStorage 根据存储层的实际条目同步元数据
	// entries 是存储层列出的条目，返回同步后的元数据列表
	SyncFromStorage(entries []SyncEntry) error
}

// SyncEntry 存储层条目，用于同步元数据
type SyncEntry struct {
	Path    string
	Name    string
	IsDir   bool
	ModTime time.Time
}
