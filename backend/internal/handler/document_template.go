package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opendeepwiki/backend/internal/service"
)

// DocumentTemplateHandler 文档模板 Handler
type DocumentTemplateHandler struct {
	templateService service.TemplateService
	chapterService  service.ChapterService
	docService      service.DocTemplateService
}

// NewDocumentTemplateHandler 创建 Handler
func NewDocumentTemplateHandler(
	templateService service.TemplateService,
	chapterService service.ChapterService,
	docService service.DocTemplateService,
) *DocumentTemplateHandler {
	return &DocumentTemplateHandler{
		templateService: templateService,
		chapterService:  chapterService,
		docService:      docService,
	}
}

// ListTemplates 获取模板列表
func (h *DocumentTemplateHandler) ListTemplates(c *gin.Context) {
	templates, err := h.templateService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// GetTemplate 获取模板详情
func (h *DocumentTemplateHandler) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	template, err := h.templateService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == service.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// CreateTemplate 创建模板
func (h *DocumentTemplateHandler) CreateTemplate(c *gin.Context) {
	var req service.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := h.templateService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrTemplateKeyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "template key already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// UpdateTemplate 更新模板
func (h *DocumentTemplateHandler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req service.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := h.templateService.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		if err == service.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// DeleteTemplate 删除模板
func (h *DocumentTemplateHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.templateService.Delete(c.Request.Context(), uint(id)); err != nil {
		if err == service.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		if err == service.ErrSystemTemplate {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system template"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// CloneTemplate 克隆模板
func (h *DocumentTemplateHandler) CloneTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := h.templateService.Clone(c.Request.Context(), uint(id), req.Key)
	if err != nil {
		if err == service.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		if err == service.ErrTemplateKeyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "template key already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// CreateChapter 创建章节
func (h *DocumentTemplateHandler) CreateChapter(c *gin.Context) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	var req service.CreateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TemplateID = uint(templateID)

	chapter, err := h.chapterService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": chapter})
}

// UpdateChapter 更新章节
func (h *DocumentTemplateHandler) UpdateChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req service.UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chapter, err := h.chapterService.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		if err == service.ErrChapterNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": chapter})
}

// DeleteChapter 删除章节
func (h *DocumentTemplateHandler) DeleteChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.chapterService.Delete(c.Request.Context(), uint(id)); err != nil {
		if err == service.ErrChapterNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// CreateDocument 创建文档
func (h *DocumentTemplateHandler) CreateDocument(c *gin.Context) {
	chapterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chapter id"})
		return
	}

	var req service.CreateDocTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ChapterID = uint(chapterID)

	doc, err := h.docService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrChapterNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": doc})
}

// UpdateDocument 更新文档
func (h *DocumentTemplateHandler) UpdateDocument(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req service.UpdateDocTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.docService.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		if err == service.ErrDocTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": doc})
}

// DeleteDocument 删除文档
func (h *DocumentTemplateHandler) DeleteDocument(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.docService.Delete(c.Request.Context(), uint(id)); err != nil {
		if err == service.ErrDocTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
