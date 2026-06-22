package service

import (
	"mynote-backend/models"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// NoteService 笔记服务
type NoteService struct {
	DataDir string
}

// DefaultDir 默认目录名
const DefaultDir = "default"

// NewNoteService 创建笔记服务
func NewNoteService(dataDir string) *NoteService {
	// 确保数据目录存在
	os.MkdirAll(dataDir, 0755)
	// 确保默认目录存在
	os.MkdirAll(filepath.Join(dataDir, DefaultDir), 0755)
	return &NoteService{DataDir: dataDir}
}

// GetTree 获取目录树
func (s *NoteService) GetTree(dirPath string) ([]*models.TreeNode, error) {
	absPath := filepath.Join(s.DataDir, dirPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var nodes []*models.TreeNode
	for _, entry := range entries {
		name := entry.Name()
		// 跳过隐藏文件和目录
		if strings.HasPrefix(name, ".") {
			continue
		}

		relPath := filepath.Join(dirPath, name)
		node := &models.TreeNode{
			Name: name,
			Path: filepath.ToSlash(relPath),
		}

		if entry.IsDir() {
			node.Type = models.TypeDirectory
			// 递归获取子节点
			children, err := s.GetTree(relPath)
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
	absPath := filepath.Join(s.DataDir, path)
	// 确保是 .md 文件
	if !strings.HasSuffix(absPath, ".md") {
		absPath += ".md"
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSuffix(info.Name(), ".md")
	relPath := filepath.ToSlash(path)
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	return &models.Note{
		Path:      relPath,
		Name:      name,
		Content:   string(content),
		UpdatedAt: info.ModTime(),
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
		absPath := filepath.Join(s.DataDir, dirPath, req.Name)
		return os.MkdirAll(absPath, 0755)
	}

	// 创建笔记
	absDirPath := filepath.Join(s.DataDir, dirPath)
	os.MkdirAll(absDirPath, 0755)

	fileName := req.Name
	if !strings.HasSuffix(fileName, ".md") {
		fileName += ".md"
	}

	absPath := filepath.Join(absDirPath, fileName)
	return os.WriteFile(absPath, []byte(req.Content), 0644)
}

// UpdateNote 更新笔记内容
func (s *NoteService) UpdateNote(path string, req models.UpdateNoteRequest) error {
	absPath := filepath.Join(s.DataDir, path)
	if !strings.HasSuffix(absPath, ".md") {
		absPath += ".md"
	}
	return os.WriteFile(absPath, []byte(req.Content), 0644)
}

// DeleteNode 删除笔记或目录
func (s *NoteService) DeleteNode(path string) error {
	absPath := filepath.Join(s.DataDir, path)
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(absPath)
	}
	return os.Remove(absPath)
}

// GetNoteModTime 获取笔记修改时间
func (s *NoteService) GetNoteModTime(path string) (time.Time, error) {
	absPath := filepath.Join(s.DataDir, path)
	if !strings.HasSuffix(absPath, ".md") {
		absPath += ".md"
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
