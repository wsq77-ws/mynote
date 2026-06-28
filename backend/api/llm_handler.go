package api

import (
	"net/http"
	"strings"

	"mynote-backend/models"
	"mynote-backend/service"

	"github.com/gin-gonic/gin"
)

// LLMHandler LLM HTTP 处理器
type LLMHandler struct {
	svc *service.LLMService
}

// NewLLMHandler 创建 LLM 处理器
func NewLLMHandler(svc *service.LLMService) *LLMHandler {
	return &LLMHandler{svc: svc}
}

// RegisterRoutes 注册 LLM 路由到 /api/llm/*
func (h *LLMHandler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/api/llm")
	{
		g.GET("/config", h.GetConfig)
		g.PUT("/config", h.UpdateConfig)
		g.POST("/complete", h.Complete)
		g.POST("/generate", h.Generate)
		g.POST("/summarize", h.Summarize)
	}
}

// GetConfig 获取 LLM 配置（api_key 脱敏）
func (h *LLMHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "获取配置失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    cfg,
	})
}

// UpdateConfig 更新 LLM 配置（部分更新）
func (h *LLMHandler) UpdateConfig(c *gin.Context) {
	var req models.UpdateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}
	if err := h.svc.UpdateConfig(req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "更新配置失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "更新成功",
	})
}

// Complete 自动补全（F1）
func (h *LLMHandler) Complete(c *gin.Context) {
	var req models.CompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "text 不能为空",
		})
		return
	}

	suggestion, err := h.svc.Complete(req.Text)
	if err != nil {
		// 未配置返回 400，其它调用失败返回 500
		if strings.Contains(err.Error(), "未配置") {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Code:    400,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "LLM 调用失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    models.CompleteResponse{Suggestion: suggestion},
	})
}

// Generate 生成笔记内容（F2）
func (h *LLMHandler) Generate(c *gin.Context) {
	var req models.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}
	// prompt 长度校验：≤ 2000 字符（超出返回 400）
	if len([]rune(req.Prompt)) > 2000 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "prompt 长度不能超过 2000 字符",
		})
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Code:    400,
			Message: "prompt 不能为空",
		})
		return
	}

	content, err := h.svc.Generate(req.Prompt)
	if err != nil {
		if strings.Contains(err.Error(), "未配置") {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Code:    400,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "LLM 调用失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    models.GenerateResponse{Content: content},
	})
}

// Summarize 总结所有笔记（F3）
func (h *LLMHandler) Summarize(c *gin.Context) {
	resp, err := h.svc.Summarize()
	if err != nil {
		if strings.Contains(err.Error(), "未配置") {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Code:    400,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Code:    500,
			Message: "LLM 调用失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    resp,
	})
}
