package meta

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteMeta SQLite 元数据管理
type SQLiteMeta struct {
	db   *sql.DB
	path string
}

// NewSQLiteMeta 创建 SQLite 元数据管理实例
func NewSQLiteMeta(dbPath string) (*SQLiteMeta, error) {
	// 确保数据库目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 单写连接
	db.SetMaxIdleConns(1)

	m := &SQLiteMeta{db: db, path: dbPath}
	if err := m.Init(); err != nil {
		db.Close()
		return nil, err
	}

	return m, nil
}

// Init 初始化数据库表
func (m *SQLiteMeta) Init() error {
	// 创建笔记元数据表（不包含 sort_order，先检查是否需要迁移）
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			is_dir INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_notes_path ON notes(path);
	`)
	if err != nil {
		return fmt.Errorf("创建 notes 表失败: %w", err)
	}

	// 检查并添加 sort_order 列（数据库迁移）
	var sortOrderCount int
	err = m.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('notes') WHERE name='sort_order'`).Scan(&sortOrderCount)
	if err != nil {
		// pragma_table_info 可能不支持，使用另一种方式检查
		_, err = m.db.Exec(`SELECT sort_order FROM notes LIMIT 1`)
		if err != nil {
			// 列不存在，添加列
			_, err = m.db.Exec(`ALTER TABLE notes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`)
			if err != nil {
				return fmt.Errorf("添加 sort_order 列失败: %w", err)
			}
			// 创建索引
			_, err = m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_notes_sort_order ON notes(sort_order)`)
			if err != nil {
				return fmt.Errorf("创建 sort_order 索引失败: %w", err)
			}
		}
	} else if sortOrderCount == 0 {
		_, err = m.db.Exec(`ALTER TABLE notes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return fmt.Errorf("添加 sort_order 列失败: %w", err)
		}
		_, err = m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_notes_sort_order ON notes(sort_order)`)
		if err != nil {
			return fmt.Errorf("创建 sort_order 索引失败: %w", err)
		}
	}

	// 创建标签表
	_, err = m.db.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			note_path TEXT NOT NULL,
			tag_name TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tags_note_path ON tags(note_path);
		CREATE INDEX IF NOT EXISTS idx_tags_tag_name ON tags(tag_name);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_unique ON tags(note_path, tag_name);
	`)
	if err != nil {
		return fmt.Errorf("创建 tags 表失败: %w", err)
	}

	// 创建内容搜索虚拟表（FTS5）
	_, err = m.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS notes_content USING fts5(
			path,
			content,
			content='',
			tokenize='unicode61'
		);
	`)
	if err != nil {
		// FTS5 可能不可用，忽略错误，后续使用 LIKE 查询
		// 创建普通表作为备份
		_, err = m.db.Exec(`
			CREATE TABLE IF NOT EXISTS notes_content (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				path TEXT NOT NULL UNIQUE,
				content TEXT
			);
			CREATE INDEX IF NOT EXISTS idx_notes_content_path ON notes_content(path);
		`)
		if err != nil {
			return fmt.Errorf("创建 notes_content 表失败: %w", err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (m *SQLiteMeta) Close() error {
	return m.db.Close()
}

// GetNoteMeta 获取笔记元数据
func (m *SQLiteMeta) GetNoteMeta(path string) (*NoteMeta, error) {
	var meta NoteMeta
	var isDir int
	err := m.db.QueryRow(`
		SELECT path, name, is_dir, sort_order, created_at, updated_at
		FROM notes WHERE path = ?
	`, path).Scan(&meta.Path, &meta.Name, &isDir, &meta.SortOrder, &meta.CreatedAt, &meta.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	meta.IsDir = isDir == 1
	return &meta, nil
}

// SaveNoteMeta 保存笔记元数据
func (m *SQLiteMeta) SaveNoteMeta(meta *NoteMeta) error {
	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.UpdatedAt = now

	isDir := 0
	if meta.IsDir {
		isDir = 1
	}

	_, err := m.db.Exec(`
		INSERT INTO notes (path, name, is_dir, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name = excluded.name,
			is_dir = excluded.is_dir,
			sort_order = excluded.sort_order,
			updated_at = excluded.updated_at
	`, meta.Path, meta.Name, isDir, meta.SortOrder, meta.CreatedAt, meta.UpdatedAt)
	return err
}

// DeleteNoteMeta 删除笔记元数据
func (m *SQLiteMeta) DeleteNoteMeta(path string) error {
	// 删除相关的标签
	_, err := m.db.Exec(`DELETE FROM tags WHERE note_path = ?`, path)
	if err != nil {
		return err
	}

	// 删除内容索引
	_, err = m.db.Exec(`DELETE FROM notes_content WHERE path = ?`, path)
	if err != nil {
		// 忽略错误，可能表不存在
	}

	// 删除元数据
	_, err = m.db.Exec(`DELETE FROM notes WHERE path = ?`, path)
	return err
}

// RenameNote 重命名笔记元数据
func (m *SQLiteMeta) RenameNote(oldPath, newPath, newName string) error {
	now := time.Now()

	// 更新笔记元数据
	_, err := m.db.Exec(`
		UPDATE notes SET path = ?, name = ?, updated_at = ? WHERE path = ?
	`, newPath, newName, now, oldPath)
	if err != nil {
		return err
	}

	// 更新标签中的路径
	_, err = m.db.Exec(`
		UPDATE tags SET note_path = ? WHERE note_path = ?
	`, newPath, oldPath)
	if err != nil {
		return err
	}

	// 更新内容索引中的路径
	_, err = m.db.Exec(`
		UPDATE notes_content SET path = ? WHERE path = ?
	`, newPath, oldPath)
	if err != nil {
		// 忽略错误
	}

	// 如果是目录，需要更新子项的路径
	rows, err := m.db.Query(`SELECT path FROM notes WHERE path LIKE ?`, oldPath+"/%")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var childPaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		childPaths = append(childPaths, path)
	}

	for _, childPath := range childPaths {
		newChildPath := strings.Replace(childPath, oldPath, newPath, 1)
		_, err := m.db.Exec(`UPDATE notes SET path = ?, updated_at = ? WHERE path = ?`, newChildPath, now, childPath)
		if err != nil {
			continue
		}
		_, err = m.db.Exec(`UPDATE tags SET note_path = ? WHERE note_path = ?`, newChildPath, childPath)
		if err != nil {
			continue
		}
		_, err = m.db.Exec(`UPDATE notes_content SET path = ? WHERE path = ?`, newChildPath, childPath)
		if err != nil {
			continue
		}
	}

	return nil
}

// UpdateSortOrder 更新排序
func (m *SQLiteMeta) UpdateSortOrder(path string, sortOrder int) error {
	_, err := m.db.Exec(`
		UPDATE notes SET sort_order = ?, updated_at = ? WHERE path = ?
	`, sortOrder, time.Now(), path)
	return err
}

// GetChildrenSorted 获取子项（按排序返回）
func (m *SQLiteMeta) GetChildrenSorted(dirPath string) ([]NoteMeta, error) {
	var query string
	var args []interface{}

	if dirPath == "" || dirPath == "." {
		query = `
			SELECT path, name, is_dir, sort_order, created_at, updated_at
			FROM notes WHERE path NOT LIKE ? AND path NOT LIKE ?
			ORDER BY sort_order, name
		`
		args = []interface{}{"%/%", "%/%"}
	} else {
		query = `
			SELECT path, name, is_dir, sort_order, created_at, updated_at
			FROM notes WHERE path LIKE ? AND path NOT LIKE ?
			ORDER BY sort_order, name
		`
		args = []interface{}{dirPath + "/%", dirPath + "/%/%"}
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metas []NoteMeta
	for rows.Next() {
		var meta NoteMeta
		var isDir int
		if err := rows.Scan(&meta.Path, &meta.Name, &meta.IsDir, &isDir, &meta.SortOrder, &meta.CreatedAt, &meta.UpdatedAt); err != nil {
			continue
		}
		meta.IsDir = isDir == 1
		metas = append(metas, meta)
	}
	return metas, nil
}

// AddTag 添加标签
func (m *SQLiteMeta) AddTag(notePath, tagName string) error {
	_, err := m.db.Exec(`
		INSERT OR IGNORE INTO tags (note_path, tag_name, created_at)
		VALUES (?, ?, ?)
	`, notePath, tagName, time.Now())
	return err
}

// RemoveTag 删除标签
func (m *SQLiteMeta) RemoveTag(notePath, tagName string) error {
	_, err := m.db.Exec(`
		DELETE FROM tags WHERE note_path = ? AND tag_name = ?
	`, notePath, tagName)
	return err
}

// GetTags 获取笔记的所有标签
func (m *SQLiteMeta) GetTags(notePath string) ([]string, error) {
	rows, err := m.db.Query(`
		SELECT tag_name FROM tags WHERE note_path = ? ORDER BY tag_name
	`, notePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// SearchByTag 按标签搜索笔记
func (m *SQLiteMeta) SearchByTag(tagName string) ([]NoteMeta, error) {
	rows, err := m.db.Query(`
		SELECT n.path, n.name, n.is_dir, n.sort_order, n.created_at, n.updated_at
		FROM notes n
		INNER JOIN tags t ON n.path = t.note_path
		WHERE t.tag_name = ?
		ORDER BY n.sort_order, n.name
	`, tagName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metas []NoteMeta
	for rows.Next() {
		var meta NoteMeta
		var isDir int
		if err := rows.Scan(&meta.Path, &meta.Name, &isDir, &meta.SortOrder, &meta.CreatedAt, &meta.UpdatedAt); err != nil {
			continue
		}
		meta.IsDir = isDir == 1
		metas = append(metas, meta)
	}
	return metas, nil
}

// GetAllTags 获取所有标签（去重）
func (m *SQLiteMeta) GetAllTags() ([]string, error) {
	rows, err := m.db.Query(`SELECT DISTINCT tag_name FROM tags ORDER BY tag_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// SaveNoteContent 保存笔记内容索引
// 使用删除+插入的方式，兼容 FTS5 虚拟表（不支持 UPSERT）
func (m *SQLiteMeta) SaveNoteContent(path, content string) error {
	// 先删除旧记录
	_, err := m.db.Exec(`DELETE FROM notes_content WHERE path = ?`, path)
	if err != nil {
		// 忽略删除错误（可能不存在）
	}

	// 插入新记录
	_, err = m.db.Exec(`INSERT INTO notes_content (path, content) VALUES (?, ?)`, path, content)
	return err
}

// DeleteNoteContent 删除笔记内容索引
func (m *SQLiteMeta) DeleteNoteContent(path string) error {
	_, err := m.db.Exec(`DELETE FROM notes_content WHERE path = ?`, path)
	return err
}

// SearchNotes 搜索笔记（标签、名称、内容）
// 注意：SQLite 连接池设置为 MaxOpenConns=1，因此在 rows 关闭前
// 不能调用其它查询（如 GetTags），否则会造成死锁。
// 本实现先收集所有命中结果并关闭 rows，再统一获取标签。
func (m *SQLiteMeta) SearchNotes(query string) ([]SearchResult, error) {
	var results []SearchResult
	seen := make(map[string]bool)
	like := "%" + query + "%"

	// 1. 搜索标签（优先级最高，明确按标签匹配）
	tagRows, err := m.db.Query(`
		SELECT DISTINCT n.path, n.name, n.is_dir, n.sort_order
		FROM notes n
		INNER JOIN tags t ON n.path = t.note_path
		WHERE t.tag_name LIKE ?
		ORDER BY n.sort_order, n.name
	`, like)
	if err != nil {
		log.Printf("[SearchNotes] 标签查询失败: %v", err)
	} else {
		for tagRows.Next() {
			var path, name string
			var isDir, sortOrder int
			if err = tagRows.Scan(&path, &name, &isDir, &sortOrder); err != nil {
				log.Printf("[SearchNotes] 标签行扫描失败: %v", err)
				continue
			}
			if seen[path] {
				// 已命中，提升匹配类型为 tag
				for i, r := range results {
					if r.Path == path && r.MatchType != "tag" {
						results[i].MatchType = "tag"
						break
					}
				}
				continue
			}
			seen[path] = true
			results = append(results, SearchResult{
				Path:      path,
				Name:      name,
				IsDir:     isDir == 1,
				MatchType: "tag",
			})
		}
		tagRows.Close()
	}

	// 2. 搜索名称
	nameRows, err := m.db.Query(`
		SELECT path, name, is_dir FROM notes
		WHERE name LIKE ? OR path LIKE ?
		ORDER BY name
	`, like, like)
	if err != nil {
		log.Printf("[SearchNotes] 名称查询失败: %v", err)
		return results, nil
	}
	for nameRows.Next() {
		var path, name string
		var isDir int
		if err = nameRows.Scan(&path, &name, &isDir); err != nil {
			log.Printf("[SearchNotes] 名称行扫描失败: %v", err)
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		results = append(results, SearchResult{
			Path:      path,
			Name:      name,
			IsDir:     isDir == 1,
			MatchType: "name",
		})
	}
	nameRows.Close()

	// 3. 搜索内容
	contentRows, err := m.db.Query(`
		SELECT path, content FROM notes_content
		WHERE content LIKE ?
	`, like)
	if err != nil {
		log.Printf("[SearchNotes] 内容查询失败: %v", err)
		return results, nil
	}
	for contentRows.Next() {
		var path, content string
		if err = contentRows.Scan(&path, &content); err != nil {
			log.Printf("[SearchNotes] 内容行扫描失败: %v", err)
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true

		snippet := extractSnippet(content, query, 100)
		name := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		}
		name = strings.TrimSuffix(name, ".md")

		results = append(results, SearchResult{
			Path:      path,
			Name:      name,
			IsDir:     false,
			Snippet:   snippet,
			MatchType: "content",
		})
	}
	contentRows.Close()

	// 4. 统一获取每条结果的标签（此时所有 rows 已关闭，可安全查询）
	// 使用缓存避免对同一路径重复查询
	tagCache := make(map[string][]string)
	for i := range results {
		p := results[i].Path
		if cached, ok := tagCache[p]; ok {
			results[i].Tags = cached
			continue
		}
		tags, err := m.GetTags(p)
		if err != nil {
			log.Printf("[SearchNotes] 获取标签失败 path=%s: %v", p, err)
			tags = nil // 获取失败不阻断整体搜索
		}
		tagCache[p] = tags
		results[i].Tags = tags
	}

	return results, nil
}

// extractSnippet 从内容中提取包含查询词的片段
func extractSnippet(content, query string, maxLen int) string {
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")

	// 查找查询词的位置
	idx := strings.Index(strings.ToLower(content), strings.ToLower(query))
	if idx == -1 {
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}

	// 计算片段的起始位置
	start := idx - maxLen/2 + len(query)/2
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}
	return snippet
}
