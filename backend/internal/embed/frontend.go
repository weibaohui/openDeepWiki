package embed

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/gzip"
)

//go:embed ui/dist/*
var embeddedFiles embed.FS

// GetFrontendFS 获取前端文件系统（用于嵌入）
func GetFrontendFS() fs.FS {
	return embeddedFiles
}

// SetupRouter 设置前端静态文件路由
func SetupRouter(r *gin.Engine) {
	// 添加 gzip 压缩中间件，使用最佳压缩级别
	r.Use(gzip.Gzip(gzip.BestCompression))

	// 获取嵌入的文件系统
	frontendFS := GetFrontendFS()

	// 获取assets目录的文件系统
	assetsFS, err := fs.Sub(frontendFS, "ui/dist/assets")
	if err == nil {
		r.GET("/assets/*filepath", gin.WrapH(http.StripPrefix("/assets", http.FileServer(http.FS(assetsFS)))))
	}

	// 设置favicon
	r.GET("/favicon.ico", func(c *gin.Context) {
		favicon, err := fs.ReadFile(frontendFS, "ui/dist/favicon.ico")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/x-icon", favicon)
	})

	// 设置根路由（SPA）
	r.NoRoute(func(c *gin.Context) {
		// 对于API请求，返回404
		if len(c.Request.URL.Path) > 4 && strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// 对于其他请求，返回index.html（SPA路由）
		indexHTML, err := fs.ReadFile(frontendFS, "ui/dist/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load index.html")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}
