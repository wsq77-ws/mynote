package meta

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteMeta SQLite 元数据管理实现
type SQLiteMeta struct {
	db *sql.DB
}

// NewSQLiteMeta 创建 SQLite 元数据管理实例
// dbPath 为数据库文件路径，如 "./data/mynote.db"
func NewSQLiteMeta(dbPath string) (*SQLiteMeta, error) {
	// 使用 modernc.org/sqlite 驱动（纯 Go，无需 CGO）
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库失败: %w", err)
	}

	// 启用 WAL 模式以提升并发性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 WAL 模式失败: %w", err)
	}

	m := &SQLiteMeta{db: db}
	if err := m.Init(); err != nil {
		db.Close()
		return nil, err
	}

	return m, nil
}

// Init 初始化数据库表
func (m *SQLiteMeta) Init() error {
	const sql = `CREATE TABLE IF NOT EXISTS notes (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		path        TEXT NOT NULL UNIQUE,
		name        TEXT NOT NULL,
		is_dir      INTEGER NOT NULL DEFAULT 0,
		author      TEXT NOT NULL DEFAULT 'default',
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_notes_path ON notes(path);
	CREATE INDEX IF NOT EXISTS idx_notes_name ON notes(name);
	CREATE INDEX IF NOT EXISTS idx_notes_is_dir ON notes(is_dir);`

	_, err := m.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}
	return nil
}

// Close 关闭数据库连接
func (m *SQLiteMeta) Close() error {
	return m.db.Close()
}

// UpsertNote 创建或更新笔记元数据
func (m *SQLiteMeta) UpsertNote(meta *NoteMeta) error {
	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	if meta.UpdatedAt.IsZero() {
		meta.UpdatedAt = now
	}
	if meta.Author == "" {
		meta.Author = "default"
	}

	const sql = `INSERT INTO notes (path, name, is_dir, author, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name = excluded.name,
			is_dir = excluded.is_dir,
			author = excluded.author,
			updated_at = excluded.updated_at`

	isDir := 0
	if meta.IsDir {
		isDir = 1
	}

	result, err := m.db.Exec(sql,
		meta.Path, meta.Name, isDir, meta.Author,
		meta.CreatedAt, meta.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("写入元数据失败: %w", err)
	}

	// 获取自增 ID
	if id, err := result.LastInsertId(); err == nil {
		meta.ID = id
	}
	return nil
}

// GetNoteByPath 根据路径获取笔记元数据
func (m *SQLiteMeta) GetNoteByPath(path string) (*NoteMeta, error) {
	const sql = `SELECT id, path, name, is_dir, author, created_at, updated_at
		FROM notes WHERE path = ?`

	row := m.db.QueryRow(sql, path)
	meta, err := scanNoteMeta(row)
	if err != nil {
		return nil, fmt.Errorf("查询元数据失败: %w", err)
	}
	return meta, nil
}

// GetAllNotes 获取所有笔记和目录的元数据
func (m *SQLiteMeta) GetAllNotes() ([]NoteMeta, error) {
	const sql = `SELECT id, path, name, is_dir, author, created_at, updated_at
		FROM notes ORDER BY is_dir DESC, name ASC`

	rows, err := m.db.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("查询所有元数据失败: %w", err)
	}
	defer rows.Close()

	return scanNoteMetas(rows)
}

// GetChildren 获取指定目录下的子条目元数据
func (m *SQLiteMeta) GetChildren(dirPath string) ([]NoteMeta, error) {
	// 匹配 dirPath 下的直接子条目（路径前缀为 dirPath/，且不再有更深层的 /）
	prefix := dirPath + "/"
	const sql = `SELECT id, path, name, is_dir, author, created_at, updated_at
		FROM notes
		WHERE path LIKE ? || '%'
		AND path != ?
		AND path NOT LIKE ? || '%/%'
		ORDER BY is_dir DESC, name ASC`

	rows, err := m.db.Query(sql, prefix, dirPath, prefix)
	if err != nil {
		return nil, fmt.Errorf("查询子条目失败: %w", err)
	}
	defer rows.Close()

	return scanNoteMetas(rows)
}

// DeleteNote 删除笔记元数据
// 如果是目录，递归删除该路径前缀下的所有条目
func (m *SQLiteMeta) DeleteNote(path string) error {
	// 先删除该路径本身
	_, err := m.db.Exec("DELETE FROM notes WHERE path = ?", path)
	if err != nil {
		return fmt.Errorf("删除元数据失败: %w", err)
	}

	// 递归删除子路径（目录情况）
	prefix := path + "/"
	_, err = m.db.Exec("DELETE FROM notes WHERE path LIKE ?", prefix+"%")
	if err != nil {
		return fmt.Errorf("删除子条目元数据失败: %w", err)
	}

	return nil
}

// Search 搜索笔记（按名称或路径模糊匹配）
func (m *SQLiteMeta) Search(keyword string) ([]NoteMeta, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, nil
	}

	// SQLite 的 LIKE 不区分大小写（默认对 ASCII 字母）
	pattern := "%" + keyword + "%"
	const sql = `SELECT id, path, name, is_dir, author, created_at, updated_at
		FROM notes
		WHERE name LIKE ? OR path LIKE ?
		ORDER BY is_dir DESC, updated_at DESC`

	rows, err := m.db.Query(sql, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}
	defer rows.Close()

	return scanNoteMetas(rows)
}

// SyncFromStorage 根据存储层的实际条目同步元数据
// 新增的条目会被插入，存储中已删除的条目会从元数据中删除
func (m *SQLiteMeta) SyncFromStorage(entries []SyncEntry) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 获取现有所有路径
	existingPaths := make(map[string]bool)
	rows, err := tx.Query("SELECT path FROM notes")
	if err != nil {
		return fmt.Errorf("查询现有路径失败: %w", err)
	}
	for rows.Next() {
		var p string
		rows.Scan(&p)
		existingPaths[p] = true
	}
	rows.Close()

	// 插入或更新当前条目
	currentPaths := make(map[string]bool)
	for _, entry := range entries {
		currentPaths[entry.Path] = true

		// 提取名称（不含路径前缀和 .md 后缀）
		name := entry.Name
		if !entry.IsDir && strings.HasSuffix(name, ".md") {
			name = strings.TrimSuffix(name, ".md")
		}

		isDir := 0
		if entry.IsDir {
			isDir = 1
		}

		_, err := tx.Exec(
			`INSERT INTO notes (path, name, is_dir, author, created_at, updated_at)
			 VALUES (?, ?, ?, 'default', ?, ?)
			 ON CONFLICT(path) DO UPDATE SET
				name = excluded.name,
				is_dir = excluded.is_dir,
				updated_at = excluded.updated_at`,
			entry.Path, name, isDir, entry.ModTime, entry.ModTime,
		)
		if err != nil {
			return fmt.Errorf("同步条目 %s 失败: %w", entry.Path, err)
		}
	}

	// 删除存储中已不存在的条目
	for path := range existingPaths {
		if !currentPaths[path] {
			_, err := tx.Exec("DELETE FROM notes WHERE path = ?", path)
			if err != nil {
				return fmt.Errorf("删除过期条目 %s 失败: %w", path, err)
			}
		}
	}

	return tx.Commit()
}

// scanNoteMeta 扫描单行元数据
func scanNoteMeta(row *sql.Row) (*NoteMeta, error) {
	var meta NoteMeta
	var isDir int
	err := row.Scan(&meta.ID, &meta.Path, &meta.Name, &isDir, &meta.Author, &meta.CreatedAt, &meta.UpdatedAt)
	if err != nil {
		return nil, err
	}
	meta.IsDir = isDir != 0
	return &meta, nil
}

// scanNoteMetas 扫描多行元数据
func scanNoteMetas(rows *sql.Rows) ([]NoteMeta, error) {
	var result []NoteMeta
	for rows.Next() {
		var meta NoteMeta
		var isDir int
		if err := rows.Scan(&meta.ID, &meta.Path, &meta.Name, &isDir, &meta.Author, &meta.CreatedAt, &meta.UpdatedAt); err != nil {
			return nil, err
		}
		meta.IsDir = isDir != 0
		result = append(result, meta)
	}
	return result, nil
}
