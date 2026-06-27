package storage

import "time"

// Entry 表示存储中的一个条目（文件或目录）
type Entry struct {
	Name    string    // 条目名称
	Path    string    // 相对路径（使用 / 分隔）
	IsDir   bool      // 是否为目录
	ModTime time.Time // 修改时间
}

// Storage 存储后端接口
// 不同的存储实现（本地文件系统、对象存储等）都需要实现此接口
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

	// Rename 重命名文件或目录
	Rename(oldPath, newPath string) error

	// Type 返回存储类型标识
	Type() string
}
