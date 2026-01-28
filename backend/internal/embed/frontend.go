package embed

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var embeddedFiles embed.FS

// FileSystem 前端文件系统
type FileSystem struct {
	fs fs.FS
}

// NewFileSystem 创建前端文件系统
func NewFileSystem() *FileSystem {
	return &FileSystem{
		fs: embeddedFiles,
	}
}

// GetFileSystem 获取文件系统
func (f *FileSystem) GetFileSystem() fs.FS {
	return f.fs
}

// GetFrontendFS 获取前端文件系统（用于嵌入）
// 返回整个dist目录的文件系统
func GetFrontendFS() fs.FS {
	return embeddedFiles
}

// GetAssetsFS 获取assets目录的文件系统
func GetAssetsFS() fs.FS {
	assetsFS, err := fs.Sub(embeddedFiles, "ui/dist/assets")
	if err != nil {
		// 如果获取失败，返回根文件系统
		return embeddedFiles
	}
	return assetsFS
}

// SetupRouter 设置前端静态文件路由
func SetupRouter(r *gin.Engine) {
	// 创建文件系统
	frontendFS := GetFrontendFS()
	assetsFS := GetAssetsFS()

	// 设置assets路由
	r.GET("/assets/*filepath", func(c *gin.Context) {
		// 移除前缀 /assets
		filepath := strings.TrimPrefix(c.Request.URL.Path, "/assets")
		c.FileFromFS(filepath, http.FS(assetsFS))
	})

	// 设置favicon
	r.GET("/favicon.ico", func(c *gin.Context) {
		favicon, err := fs.ReadFile(assetsFS, "vite.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/svg+xml", favicon)
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
