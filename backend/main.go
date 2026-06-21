package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"mynote-backend/api"
	"mynote-backend/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 获取数据目录
	execPath, _ := os.Executable()
	dataDir := filepath.Join(filepath.Dir(execPath), "data")

	// 支持通过环境变量覆盖数据目录
	if envDir := os.Getenv("MYNOTE_DATA_DIR"); envDir != "" {
		dataDir = envDir
	}

	// 确保数据目录存在
	os.MkdirAll(dataDir, 0755)
	log.Printf("数据目录: %s", dataDir)

	// 创建服务
	noteSvc := service.NewNoteService(dataDir)
	handler := api.NewHandler(noteSvc)

	// 创建路由 - ReleaseMode 下关闭 Gin 日志彩色输出
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = "debug"
	}
	gin.SetMode(mode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 注册API路由
	handler.RegisterRoutes(r)

	// 生产模式：Serve 前端静态文件
	distDir := os.Getenv("MYNOTE_DIST_DIR")
	if distDir == "" {
		distDir = filepath.Join(filepath.Dir(execPath), "..", "frontend", "dist")
	}
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		log.Printf("静态文件目录: %s", distDir)
		r.Use(func(c *gin.Context) {
			// 非 /api 开头的请求，尝试返回前端页面
			if len(c.Request.URL.Path) < 4 || c.Request.URL.Path[:4] != "/api" {
				http.FileServer(http.Dir(distDir)).ServeHTTP(c.Writer, c.Request)
				c.Abort()
			}
		})
		// 404 fallback 到 index.html（支持 Vue Router 的 History 模式）
		r.NoRoute(func(c *gin.Context) {
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API not found"})
				return
			}
			http.ServeFile(c.Writer, c.Request, filepath.Join(distDir, "index.html"))
		})
		log.Println("生产模式：前端静态文件由后端提供服务")
	} else {
		log.Println("开发模式：前端由 Vite 开发服务器提供服务 (http://localhost:3000)")
		// 开发模式下不需要 fallback
	}

	// 启动服务
	port := os.Getenv("MYNOTE_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("服务启动在 http://localhost:%s", port)
	r.Run(":" + port)
}
