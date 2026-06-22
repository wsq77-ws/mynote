package service

import (
	"mynote-backend/meta"
	"mynote-backend/models"
	"mynote-backend/storage"
	"sort"
	"strings"
	"time"
)

// DefaultDir 默认目录名
const DefaultDir = "default"

// DefaultAuthor 默认作者名
const DefaultAuthor = "default"

// NoteService 笔记服务
type NoteService struct {
	storage storage.Storage
	meta    meta.Meta
}

// NewNoteService 创建笔记服务
func NewNoteService(s storage.Storage, m meta.Meta) *NoteService {
	// 确保默认目录存在
	s.Mkdir(DefaultDir)
	svc := &NoteService{storage: s, meta: m}

	// 启动时同步元数据
	svc.syncMetadata()

	return svc
}

// syncMetadata 从存储层同步所有元数据到 SQLite
func (s *NoteService) syncMetadata() {
	entries := s.collectAllEntries(".")
	_ = s.meta.SyncFromStorage(entries)
}

// collectAllEntries 递归收集存储层所有条目
func (s *NoteService) collectAllEntries(dirPath string) []meta.SyncEntry {
	storageEntries, err := s.storage.List(dirPath)
	if err != nil {
		return nil
	}

	var result []meta.SyncEntry
	for _, entry := range storageEntries {
		if strings.HasPrefix(entry.Name, ".") {
			continue
		}

		// 跳过非 .md 文件
		if !entry.IsDir && !strings.HasSuffix(entry.Name, ".md") {
			continue
		}

		result = append(result, meta.SyncEntry{
			Path:    entry.Path,
			Name:    entry.Name,
			IsDir:   entry.IsDir,
			ModTime: entry.ModTime,
		})

		if entry.IsDir {
			children := s.collectAllEntries(entry.Path)
			result = append(result, children...)
		}
	}
	return result
}

// GetTree 获取目录树
// 先从 SQLite 读取元数据，再从存储层加载目录结构
func (s *NoteService) GetTree(dirPath string) ([]*models.TreeNode, error) {
	if dirPath == "" {
		dirPath = "."
	}

	// 从存储层获取实际目录结构
	entries, err := s.storage.List(dirPath)
	if err != nil {
		return nil, err
	}

	var nodes []*models.TreeNode
	for _, entry := range entries {
		name := entry.Name
		if strings.HasPrefix(name, ".") {
			continue
		}

		node := &models.TreeNode{
			Name:      name,
			Path:      entry.Path,
			UpdatedAt: entry.ModTime,
		}

		// 从 SQLite 获取元数据（作者等）
		if m, err := s.meta.GetNoteByPath(entry.Path); err == nil {
			node.Author = m.Author
			node.UpdatedAt = m.UpdatedAt
		}

		if entry.IsDir {
			node.Type = models.TypeDirectory
			children, err := s.GetTree(entry.Path)
			if err == nil && len(children) > 0 {
				node.Children = children
			} else {
				node.Children = []*models.TreeNode{}
			}
		} else if strings.HasSuffix(name, ".md") {
			node.Type = models.TypeFile
			node.Name = strings.TrimSuffix(name, ".md")
		} else {
			continue
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type == models.TypeDirectory
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

// GetNote 获取笔记内容
// 先从 SQLite 读取元数据，再从存储层加载文件内容
func (s *NoteService) GetNote(path string) (*models.Note, error) {
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	// 从存储层读取内容
	content, modTime, err := s.storage.Read(path)
	if err != nil {
		return nil, err
	}

	name := path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(name, ".md")

	note := &models.Note{
		Path:      path,
		Name:      name,
		Content:   content,
		UpdatedAt: modTime,
	}

	// 从 SQLite 读取元数据（作者等）
	if m, err := s.meta.GetNoteByPath(path); err == nil {
		note.Author = m.Author
		note.UpdatedAt = m.UpdatedAt
	} else {
		note.Author = DefaultAuthor
	}

	return note, nil
}

// CreateNote 创建笔记或目录
// 同时写入存储层和 SQLite 元数据
func (s *NoteService) CreateNote(req models.CreateNoteRequest) error {
	dirPath := req.Path
	if dirPath == "" {
		dirPath = DefaultDir
	}

	author := req.Author
	if author == "" {
		author = DefaultAuthor
	}

	if req.IsDir {
		fullPath := dirPath + "/" + req.Name
		if err := s.storage.Mkdir(fullPath); err != nil {
			return err
		}
		// 写入元数据
		return s.meta.UpsertNote(&meta.NoteMeta{
			Path:      fullPath,
			Name:      req.Name,
			IsDir:     true,
			Author:    author,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	fileName := req.Name
	if !strings.HasSuffix(fileName, ".md") {
		fileName += ".md"
	}

	fullPath := dirPath + "/" + fileName
	if err := s.storage.Write(fullPath, req.Content); err != nil {
		return err
	}

	// 写入元数据
	return s.meta.UpsertNote(&meta.NoteMeta{
		Path:      fullPath,
		Name:      req.Name,
		IsDir:     false,
		Author:    author,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

// UpdateNote 更新笔记内容
// 同时更新存储层和 SQLite 元数据
func (s *NoteService) UpdateNote(path string, req models.UpdateNoteRequest) error {
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	if err := s.storage.Write(path, req.Content); err != nil {
		return err
	}

	// 更新元数据的时间戳
	if m, err := s.meta.GetNoteByPath(path); err == nil {
		m.UpdatedAt = time.Now()
		return s.meta.UpsertNote(m)
	}

	// 元数据不存在则创建
	name := path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(name, ".md")
	return s.meta.UpsertNote(&meta.NoteMeta{
		Path:      path,
		Name:      name,
		IsDir:     false,
		Author:    DefaultAuthor,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

// DeleteNode 删除笔记或目录
// 同时删除存储层文件和 SQLite 元数据
func (s *NoteService) DeleteNode(path string) error {
	if err := s.storage.Delete(path); err != nil {
		return err
	}
	return s.meta.DeleteNote(path)
}

// Search 搜索笔记
func (s *NoteService) Search(keyword string) ([]models.SearchResult, error) {
	metas, err := s.meta.Search(keyword)
	if err != nil {
		return nil, err
	}

	var results []models.SearchResult
	for _, m := range metas {
		results = append(results, models.SearchResult{
			Path:      m.Path,
			Name:      m.Name,
			Author:    m.Author,
			IsDir:     m.IsDir,
			UpdatedAt: m.UpdatedAt,
		})
	}
	return results, nil
}
