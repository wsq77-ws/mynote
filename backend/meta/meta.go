package meta

import "time"

// NoteMeta 笔记元数据
type NoteMeta struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	IsDir     bool      `json:"is_dir"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Tag 标签
type Tag struct {
	ID        int64     `json:"id"`
	NotePath  string    `json:"note_path"`
	TagName   string    `json:"tag_name"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Path      string   `json:"path"`
	Name      string   `json:"name"`
	IsDir     bool     `json:"is_dir"`
	Snippet   string   `json:"snippet,omitempty"`
	MatchType string   `json:"match_type"` // "name" | "content" | "tag"
	Tags      []string `json:"tags,omitempty"`
}

// Meta 元数据管理接口
type Meta interface {
	// Init 初始化数据库表
	Init() error

	// Close 关闭数据库连接
	Close() error

	// 笔记元数据操作
	GetNoteMeta(path string) (*NoteMeta, error)
	SaveNoteMeta(meta *NoteMeta) error
	DeleteNoteMeta(path string) error
	RenameNote(oldPath, newPath, newName string) error

	// 排序相关
	UpdateSortOrder(path string, sortOrder int) error
	GetChildrenSorted(dirPath string) ([]NoteMeta, error)

	// 标签操作
	AddTag(notePath, tagName string) error
	RemoveTag(notePath, tagName string) error
	GetTags(notePath string) ([]string, error)
	SearchByTag(tagName string) ([]NoteMeta, error)
	GetAllTags() ([]string, error)

	// 搜索
	SearchNotes(query string) ([]SearchResult, error)

	// 内容索引
	SaveNoteContent(path, content string) error
	DeleteNoteContent(path string) error
}
