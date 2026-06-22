package models

import "time"

// NodeType 节点类型
type NodeType string

const (
	TypeFile      NodeType = "file"
	TypeDirectory NodeType = "directory"
)

// TreeNode 目录树节点
type TreeNode struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Type      NodeType    `json:"type"`
	Author    string      `json:"author"`
	UpdatedAt time.Time   `json:"updated_at"`
	Children  []*TreeNode `json:"children,omitempty"`
}

// Note 笔记信息
type Note struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateNoteRequest 创建笔记请求
// Path 为空时后端默认使用 "default" 目录
// Author 为空时默认使用 "default"
type CreateNoteRequest struct {
	Path    string `json:"path"`
	Name    string `json:"name" binding:"required"`
	Author  string `json:"author"`
	IsDir   bool   `json:"is_dir"`
	Content string `json:"content"`
}

// UpdateNoteRequest 更新笔记请求
type UpdateNoteRequest struct {
	Content string `json:"content" binding:"required"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Author    string    `json:"author"`
	IsDir     bool      `json:"is_dir"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIResponse 通用API响应
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
