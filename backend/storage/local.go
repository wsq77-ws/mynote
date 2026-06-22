package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage 本地文件系统存储
type LocalStorage struct {
	dataDir string
}

// NewLocalStorage 创建本地文件系统存储实例
func NewLocalStorage(dataDir string) (*LocalStorage, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	return &LocalStorage{dataDir: dataDir}, nil
}

// absPath 将相对路径转为绝对路径
func (s *LocalStorage) absPath(path string) string {
	return filepath.Join(s.dataDir, filepath.FromSlash(path))
}

// List 列出指定目录下的条目（非递归）
func (s *LocalStorage) List(dirPath string) ([]Entry, error) {
	absPath := s.absPath(dirPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, entry := range entries {
		name := entry.Name()
		// 跳过隐藏文件
		if strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath := filepath.ToSlash(filepath.Join(dirPath, name))
		result = append(result, Entry{
			Name:    name,
			Path:    relPath,
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}

	return result, nil
}

// Read 读取文件内容
func (s *LocalStorage) Read(path string) (string, time.Time, error) {
	absPath := s.absPath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return "", time.Time{}, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", time.Time{}, err
	}

	return string(content), info.ModTime(), nil
}

// Write 写入文件内容
func (s *LocalStorage) Write(path string, content string) error {
	absPath := s.absPath(path)
	// 确保父目录存在
	os.MkdirAll(filepath.Dir(absPath), 0755)
	return os.WriteFile(absPath, []byte(content), 0644)
}

// Mkdir 创建目录
func (s *LocalStorage) Mkdir(path string) error {
	absPath := s.absPath(path)
	return os.MkdirAll(absPath, 0755)
}

// Delete 删除文件或目录
func (s *LocalStorage) Delete(path string) error {
	absPath := s.absPath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(absPath)
	}
	return os.Remove(absPath)
}

// GetModTime 获取文件修改时间
func (s *LocalStorage) GetModTime(path string) (time.Time, error) {
	absPath := s.absPath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Exists 检查路径是否存在
func (s *LocalStorage) Exists(path string) (bool, error) {
	absPath := s.absPath(path)
	_, err := os.Stat(absPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Type 返回存储类型标识
func (s *LocalStorage) Type() string {
	return "local"
}
