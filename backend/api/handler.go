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
		api.GET("/tree", h.GetTree)
		api.GET("/note", h.GetNote)
		api.POST("/note", h.CreateNote)
		api.PUT("/note", h.UpdateNote)
		api.DELETE("/note", h.DeleteNote)
		api.GET("/search", h.Search)
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
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "搜索关键词不能为空",
		})
		return
	}

	results, err := h.svc.Search(keyword)
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
