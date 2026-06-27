package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"mynote-backend/api"
	"mynote-backend/meta"
	"mynote-backend/service"
	"mynote-backend/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// 获取可执行文件目录
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)

	// 加载配置文件
	// 查找顺序: 环境变量 MYNOTE_CONFIG > 当前目录 config.yaml > 可执行文件目录 config.yaml
	configPath := os.Getenv("MYNOTE_CONFIG")
	if configPath == "" {
		configPath = "config.yaml"
		if _, err := os.Stat(configPath); err != nil {
			configPath = filepath.Join(execDir, "config.yaml")
		}
	}

	var cfg *storage.Config
	if _, err := os.Stat(configPath); err == nil {
		cfg, err = storage.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		log.Printf("已加载配置文件: %s", configPath)
	} else {
		log.Println("未找到配置文件，使用默认配置")
		cfg = storage.DefaultConfig()
	}

	// 环境变量覆盖: MYNOTE_DATA_DIR 优先于配置文件
	if envDir := os.Getenv("MYNOTE_DATA_DIR"); envDir != "" {
		cfg.Storage.Local.DataDir = envDir
	}

	// 创建存储后端
	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("创建存储后端失败: %v", err)
	}
	log.Printf("存储后端: %s", store.Type())

	// 创建元数据管理
	dbPath := filepath.Join(cfg.Storage.Local.DataDir, "mynote.db")
	if envDb := os.Getenv("MYNOTE_DB_PATH"); envDb != "" {
		dbPath = envDb
	}
	metaStore, err := meta.NewSQLiteMeta(dbPath)
	if err != nil {
		log.Fatalf("创建元数据管理失败: %v", err)
	}
	log.Printf("数据库路径: %s", dbPath)

	// 创建服务
	noteSvc := service.NewNoteService(store, metaStore)
	handler := api.NewHandler(noteSvc)

	// 创建路由
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
		distDir = filepath.Join(execDir, "..", "frontend", "dist")
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
	}

	// 启动服务
	port := os.Getenv("MYNOTE_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("服务启动在 http://localhost:%s", port)
	r.Run(":" + port)
}
