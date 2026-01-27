package services

import (
	"archive/zip"
	"bytes"
	"fmt"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/models"
)

type DocumentService struct {
	cfg *config.Config
}

func NewDocumentService(cfg *config.Config) *DocumentService {
	return &DocumentService{cfg: cfg}
}

type CreateDocumentRequest struct {
	RepositoryID uint   `json:"repository_id"`
	TaskID       uint   `json:"task_id"`
	Title        string `json:"title"`
	Filename     string `json:"filename"`
	Content      string `json:"content"`
	SortOrder    int    `json:"sort_order"`
}

func (s *DocumentService) Create(req CreateDocumentRequest) (*models.Document, error) {
	doc := &models.Document{
		RepositoryID: req.RepositoryID,
		TaskID:       req.TaskID,
		Title:        req.Title,
		Filename:     req.Filename,
		Content:      req.Content,
		SortOrder:    req.SortOrder,
	}

	if err := models.DB.Create(doc).Error; err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) GetByRepository(repoID uint) ([]models.Document, error) {
	var docs []models.Document
	err := models.DB.Where("repository_id = ?", repoID).Order("sort_order").Find(&docs).Error
	return docs, err
}

func (s *DocumentService) Get(id uint) (*models.Document, error) {
	var doc models.Document
	err := models.DB.First(&doc, id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *DocumentService) Update(id uint, content string) (*models.Document, error) {
	var doc models.Document
	if err := models.DB.First(&doc, id).Error; err != nil {
		return nil, err
	}

	doc.Content = content
	if err := models.DB.Save(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *DocumentService) Delete(id uint) error {
	return models.DB.Delete(&models.Document{}, id).Error
}

func (s *DocumentService) ExportAll(repoID uint) ([]byte, string, error) {
	var repo models.Repository
	if err := models.DB.First(&repo, repoID).Error; err != nil {
		return nil, "", err
	}

	docs, err := s.GetByRepository(repoID)
	if err != nil {
		return nil, "", err
	}

	if len(docs) == 0 {
		return nil, "", fmt.Errorf("no documents to export")
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	indexContent := s.generateIndex(repo.Name, docs)
	indexFile, err := zipWriter.Create("index.md")
	if err != nil {
		return nil, "", err
	}
	indexFile.Write([]byte(indexContent))

	for _, doc := range docs {
		file, err := zipWriter.Create(doc.Filename)
		if err != nil {
			return nil, "", err
		}
		file.Write([]byte(doc.Content))
	}

	if err := zipWriter.Close(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("%s-docs.zip", repo.Name)
	return buf.Bytes(), filename, nil
}

func (s *DocumentService) generateIndex(repoName string, docs []models.Document) string {
	content := fmt.Sprintf("# %s 文档目录\n\n", repoName)
	content += "## 目录\n\n"

	for i, doc := range docs {
		content += fmt.Sprintf("%d. [%s](./%s)\n", i+1, doc.Title, doc.Filename)
	}

	content += "\n---\n\n*由 openDeepWiki 自动生成*\n"
	return content
}

func (s *DocumentService) GetIndex(repoID uint) (string, error) {
	var repo models.Repository
	if err := models.DB.First(&repo, repoID).Error; err != nil {
		return "", err
	}

	docs, err := s.GetByRepository(repoID)
	if err != nil {
		return "", err
	}

	return s.generateIndex(repo.Name, docs), nil
}
