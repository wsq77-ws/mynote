package api

import (
	"net/http"
	"strings"

	"mynote-backend/models"
	"mynote-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler API处理器
type Handler struct {
	svc *service.NoteService
}

// NewHandler 创建处理器
func NewHandler(svc *service.NoteService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 笔记相关
		api.GET("/tree", h.GetTree)
		api.GET("/note", h.GetNote)
		api.POST("/note", h.CreateNote)
		api.PUT("/note", h.UpdateNote)
		api.DELETE("/note", h.DeleteNote)

		// 搜索
		api.GET("/search", h.Search)

		// 标签
		api.GET("/tags", h.GetTags)
		api.POST("/tags", h.AddTag)
		api.PUT("/tags", h.UpdateTags)
		api.DELETE("/tags", h.RemoveTag)
		api.GET("/tags/search", h.SearchByTag)
		api.GET("/tags/all", h.GetAllTags)

		// 重命名
		api.PUT("/rename", h.Rename)

		// 排序
		api.POST("/sort", h.UpdateSortOrder)
	}
}

// GetTree 获取目录树
func (h *Handler) GetTree(c *gin.Context) {
	path := strings.Trim(c.Query("path"), "/")
	if path == "" {
		path = "."
	}

	tree, err := h.svc.GetTree(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "获取目录树失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    tree,
	})
}

// GetNote 获取笔记内容
func (h *Handler) GetNote(c *gin.Context) {
	path := strings.Trim(c.Query("path"), "/")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "path参数不能为空",
		})
		return
	}

	note, err := h.svc.GetNote(path)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Code:    404,
			Message: "笔记不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    note,
	})
}

// CreateNote 创建笔记或目录
func (h *Handler) CreateNote(c *gin.Context) {
	var req models.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.svc.CreateNote(req); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "创建失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "创建成功",
	})
}

// UpdateNote 更新笔记内容
func (h *Handler) UpdateNote(c *gin.Context) {
	path := strings.Trim(c.Query("path"), "/")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "path参数不能为空",
		})
		return
	}

	var req models.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.svc.UpdateNote(path, req); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "更新成功",
	})
}

// DeleteNote 删除笔记或目录
func (h *Handler) DeleteNote(c *gin.Context) {
	path := strings.Trim(c.Query("path"), "/")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "path参数不能为空",
		})
		return
	}

	if err := h.svc.DeleteNode(path); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "删除成功",
	})
}

// Search 搜索笔记
func (h *Handler) Search(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "query参数不能为空",
		})
		return
	}

	results, err := h.svc.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "搜索失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    results,
	})
}

// GetTags 获取笔记的标签
func (h *Handler) GetTags(c *gin.Context) {
	path := strings.Trim(c.Query("path"), "/")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "path参数不能为空",
		})
		return
	}

	tags, err := h.svc.GetTags(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "获取标签失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    tags,
	})
}

// AddTag 添加标签
func (h *Handler) AddTag(c *gin.Context) {
	var req models.TagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.svc.AddTag(req.Path, req.Tag); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "添加标签失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "添加标签成功",
	})
}

// UpdateTags 批量更新笔记标签
func (h *Handler) UpdateTags(c *gin.Context) {
	var req models.UpdateTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.svc.UpdateTags(req.Path, req.Tags); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "更新标签失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "更新标签成功",
	})
}

// RemoveTag 删除标签
func (h *Handler) RemoveTag(c *gin.Context) {
	var req models.TagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.svc.RemoveTag(req.Path, req.Tag); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "删除标签失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "删除标签成功",
	})
}

// SearchByTag 按标签搜索
func (h *Handler) SearchByTag(c *gin.Context) {
	tag := c.Query("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "tag参数不能为空",
		})
		return
	}

	results, err := h.svc.SearchByTag(tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "搜索失败: " + err.Error(),
		})
		return
	}

	// 转换为前端友好的格式
	var nodes []*models.TreeNode
	for _, m := range results {
		name := m.Name
		if !m.IsDir && !strings.HasSuffix(m.Path, ".md") {
			name = strings.TrimSuffix(name, ".md")
		}
		nodes = append(nodes, &models.TreeNode{
			Name: name,
			Path: m.Path,
			Type: models.TypeFile,
		})
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    nodes,
	})
}

// GetAllTags 获取所有标签
func (h *Handler) GetAllTags(c *gin.Context) {
	tags, err := h.svc.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "获取标签列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    tags,
	})
}

// Rename 重命名笔记或目录
func (h *Handler) Rename(c *gin.Context) {
	path := strings.Trim(c.Query("path"), "/")
	newName := c.Query("newName")

	if path == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "path参数不能为空",
		})
		return
	}

	if newName == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "newName参数不能为空",
		})
		return
	}

	if err := h.svc.Rename(path, newName); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "重命名失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "重命名成功",
	})
}

// UpdateSortOrder 更新排序
func (h *Handler) UpdateSortOrder(c *gin.Context) {
	var req models.SortOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.svc.UpdateSortOrder(req.Path, req.SortOrder); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "更新排序失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "更新排序成功",
	})
}
