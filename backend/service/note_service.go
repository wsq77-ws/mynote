package service

import (
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
}

// NewNoteService 创建笔记服务
func NewNoteService(s storage.Storage) *NoteService {
	// 确保默认目录存在
	s.Mkdir(DefaultDir)
	return &NoteService{storage: s}
}

// GetTree 获取目录树
func (s *NoteService) GetTree(dirPath string) ([]*models.TreeNode, error) {
	entries, err := s.storage.List(dirPath)
	if err != nil {
		return nil, err
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

	// 排序：目录在前，文件在后，按名称排序
	sort.Slice(nodes, func(i, j int) bool {
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
		return s.storage.Mkdir(fullPath)
	}

	// 创建笔记
	fileName := req.Name
	if !strings.HasSuffix(fileName, ".md") {
		fileName += ".md"
	}

	fullPath := dirPath + "/" + fileName
	return s.storage.Write(fullPath, req.Content)
}

// UpdateNote 更新笔记内容
func (s *NoteService) UpdateNote(path string, req models.UpdateNoteRequest) error {
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	return s.storage.Write(path, req.Content)
}

// DeleteNode 删除笔记或目录
func (s *NoteService) DeleteNode(path string) error {
	return s.storage.Delete(path)
}

// GetNoteModTime 获取笔记修改时间
func (s *NoteService) GetNoteModTime(path string) (time.Time, error) {
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	return s.storage.GetModTime(path)
}
