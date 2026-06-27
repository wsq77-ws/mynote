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

// NoteService 笔记服务
type NoteService struct {
	storage storage.Storage
	meta    meta.Meta
}

// NewNoteService 创建笔记服务
func NewNoteService(s storage.Storage, m meta.Meta) *NoteService {
	// 确保默认目录存在
	s.Mkdir(DefaultDir)
	return &NoteService{storage: s, meta: m}
}

// GetTree 获取目录树
func (s *NoteService) GetTree(dirPath string) ([]*models.TreeNode, error) {
	entries, err := s.storage.List(dirPath)
	if err != nil {
		return nil, err
	}

	// 尝试从元数据获取排序信息
	sortedMetas, _ := s.meta.GetChildrenSorted(dirPath)
	sortOrderMap := make(map[string]int)
	for _, m := range sortedMetas {
		sortOrderMap[m.Path] = m.SortOrder
	}

	var nodes []*models.TreeNode
	for _, entry := range entries {
		name := entry.Name
		// 跳过隐藏文件和目录
		if strings.HasPrefix(name, ".") {
			continue
		}

		node := &models.TreeNode{
			Name: name,
			Path: entry.Path,
		}

		// 保存/更新元数据
		isDir := entry.IsDir
		metaName := name
		if !isDir && strings.HasSuffix(name, ".md") {
			metaName = strings.TrimSuffix(name, ".md")
		}
		s.meta.SaveNoteMeta(&meta.NoteMeta{
			Path:      entry.Path,
			Name:      metaName,
			IsDir:     isDir,
			SortOrder: sortOrderMap[entry.Path],
		})

		if entry.IsDir {
			node.Type = models.TypeDirectory
			// 递归获取子节点
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
			continue // 跳过非md文件
		}

		nodes = append(nodes, node)
	}

	// 排序：优先按 sort_order，然后目录在前，文件在后，按名称排序
	sort.Slice(nodes, func(i, j int) bool {
		orderI := sortOrderMap[nodes[i].Path]
		orderJ := sortOrderMap[nodes[j].Path]
		if orderI != orderJ {
			return orderI < orderJ
		}
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type == models.TypeDirectory
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

// GetNote 获取笔记内容
func (s *NoteService) GetNote(path string) (*models.Note, error) {
	// 确保是 .md 文件
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	content, modTime, err := s.storage.Read(path)
	if err != nil {
		return nil, err
	}

	name := path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(name, ".md")

	return &models.Note{
		Path:      path,
		Name:      name,
		Content:   content,
		UpdatedAt: modTime,
	}, nil
}

// CreateNote 创建笔记或目录
// 当 req.Path 为空时，笔记/目录将创建在默认目录 default 下
func (s *NoteService) CreateNote(req models.CreateNoteRequest) error {
	// 路径为空时使用默认目录
	dirPath := req.Path
	if dirPath == "" {
		dirPath = DefaultDir
	}

	if req.IsDir {
		fullPath := dirPath + "/" + req.Name
		if err := s.storage.Mkdir(fullPath); err != nil {
			return err
		}
		// 保存元数据
		return s.meta.SaveNoteMeta(&meta.NoteMeta{
			Path:  fullPath,
			Name:  req.Name,
			IsDir: true,
		})
	}

	// 创建笔记
	fileName := req.Name
	if !strings.HasSuffix(fileName, ".md") {
		fileName += ".md"
	}

	fullPath := dirPath + "/" + fileName
	if err := s.storage.Write(fullPath, req.Content); err != nil {
		return err
	}
	// 保存元数据和内容索引
	s.meta.SaveNoteMeta(&meta.NoteMeta{
		Path:  fullPath,
		Name:  req.Name,
		IsDir: false,
	})
	return s.meta.SaveNoteContent(fullPath, req.Content)
}

// UpdateNote 更新笔记内容
func (s *NoteService) UpdateNote(path string, req models.UpdateNoteRequest) error {
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	if err := s.storage.Write(path, req.Content); err != nil {
		return err
	}
	// 更新内容索引
	return s.meta.SaveNoteContent(path, req.Content)
}

// DeleteNode 删除笔记或目录
func (s *NoteService) DeleteNode(path string) error {
	// 删除元数据
	s.meta.DeleteNoteMeta(path)
	return s.storage.Delete(path)
}

// GetNoteModTime 获取笔记修改时间
func (s *NoteService) GetNoteModTime(path string) (time.Time, error) {
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	return s.storage.GetModTime(path)
}

// Rename 重命名笔记或目录
func (s *NoteService) Rename(oldPath, newName string) error {
	// 计算新路径
	parentDir := ""
	if idx := strings.LastIndex(oldPath, "/"); idx >= 0 {
		parentDir = oldPath[:idx]
	}

	isDir := false
	if !strings.HasSuffix(oldPath, ".md") {
		// 检查是否是目录
		exists, _ := s.storage.Exists(oldPath)
		if exists {
			isDir = true
		}
	}

	newPath := newName
	if parentDir != "" {
		newPath = parentDir + "/" + newName
	}
	if !isDir && !strings.HasSuffix(newPath, ".md") {
		newPath += ".md"
	}

	// 如果源路径没有 .md 后缀但实际是文件，需要添加
	if !isDir && !strings.HasSuffix(oldPath, ".md") {
		oldPath += ".md"
	}

	// 执行重命名
	if err := s.storage.Rename(oldPath, newPath); err != nil {
		return err
	}

	// 更新元数据
	return s.meta.RenameNote(oldPath, newPath, newName)
}

// Search 搜索笔记（名称和内容）
func (s *NoteService) Search(query string) ([]meta.SearchResult, error) {
	return s.meta.SearchNotes(query)
}

// AddTag 添加标签
func (s *NoteService) AddTag(notePath, tagName string) error {
	// 确保 .md 后缀
	if !strings.HasSuffix(notePath, ".md") {
		notePath += ".md"
	}
	return s.meta.AddTag(notePath, tagName)
}

// RemoveTag 删除标签
func (s *NoteService) RemoveTag(notePath, tagName string) error {
	if !strings.HasSuffix(notePath, ".md") {
		notePath += ".md"
	}
	return s.meta.RemoveTag(notePath, tagName)
}

// GetTags 获取笔记的所有标签
func (s *NoteService) GetTags(notePath string) ([]string, error) {
	if !strings.HasSuffix(notePath, ".md") {
		notePath += ".md"
	}
	return s.meta.GetTags(notePath)
}

// SearchByTag 按标签搜索
func (s *NoteService) SearchByTag(tagName string) ([]meta.NoteMeta, error) {
	return s.meta.SearchByTag(tagName)
}

// GetAllTags 获取所有标签
func (s *NoteService) GetAllTags() ([]string, error) {
	return s.meta.GetAllTags()
}

// UpdateSortOrder 更新排序
func (s *NoteService) UpdateSortOrder(path string, sortOrder int) error {
	return s.meta.UpdateSortOrder(path, sortOrder)
}

// UpdateTags 批量更新笔记标签
func (s *NoteService) UpdateTags(notePath string, newTags []string) error {
	if !strings.HasSuffix(notePath, ".md") {
		notePath += ".md"
	}
	// 获取当前标签
	oldTags, err := s.meta.GetTags(notePath)
	if err != nil {
		return err
	}

	// 计算需要添加和删除的标签
	oldSet := make(map[string]bool)
	for _, t := range oldTags {
		oldSet[t] = true
	}
	newSet := make(map[string]bool)
	for _, t := range newTags {
		newSet[t] = true
	}

	// 添加新标签
	for _, t := range newTags {
		if !oldSet[t] {
			if err := s.meta.AddTag(notePath, t); err != nil {
				return err
			}
		}
	}

	// 删除旧标签
	for _, t := range oldTags {
		if !newSet[t] {
			if err := s.meta.RemoveTag(notePath, t); err != nil {
				return err
			}
		}
	}

	return nil
}
