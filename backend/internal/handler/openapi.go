package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

// OpenAPIHandler OpenAPI 文档处理器
// 提供 OpenAPI 规范文档的访问端点，使 AI 工具能够理解和使用 API
type OpenAPIHandler struct {
	// openAPIPath OpenAPI 文档文件路径
	openAPIPath string
}

// NewOpenAPIHandler 创建 OpenAPI 处理器
// openAPIPath: OpenAPI 文档的文件路径
func NewOpenAPIHandler(openAPIPath string) *OpenAPIHandler {
	klog.V(6).Infof("[openapi] 创建 OpenAPI 处理器, 路径=%s", openAPIPath)
	return &OpenAPIHandler{
		openAPIPath: openAPIPath,
	}
}

// ServeOpenAPI 提供 OpenAPI 规范文档
// 端点: /.well-known/openapi.yaml
// 方法: GET
// 响应: application/x-yaml
func (h *OpenAPIHandler) ServeOpenAPI(c *gin.Context) {
	klog.V(6).Infof("[openapi] 接收到 OpenAPI 文档请求")

	// 读取 OpenAPI 文档文件
	c.File(h.openAPIPath)

	// 设置正确的 Content-Type
	c.Header("Content-Type", "application/x-yaml")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Cache-Control", "public, max-age=3600")

	klog.V(6).Infof("[openapi] OpenAPI 文档返回成功")
}

// ServeOpenAPIJSON 提供 OpenAPI 规范文档（JSON 格式）
// 端点: /.well-known/openapi.json
// 方法: GET
// 响应: application/json
//
// 此方法为扩展方法，提供 JSON 格式的 OpenAPI 文档，
// 方便某些不支持 YAML 的工具使用
func (h *OpenAPIHandler) ServeOpenAPIJSON(c *gin.Context) {
	klog.V(6).Infof("[openapi] 接收到 OpenAPI JSON 文档请求")

	// 暂不实现，返回 404
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "OpenAPI JSON format is not supported yet",
	})
}

// GetOpenAPISpec 获取 OpenAPI 规范内容（用于其他内部使用）
// 返回 OpenAPI 文档的字节数组和可能的错误
func (h *OpenAPIHandler) GetOpenAPISpec() ([]byte, error) {
	// 使用 gin.Context 的 File 方法无法直接读取内容
	// 这里只是预留接口，如有需要可以实现文件读取逻辑
	return nil, nil
}
