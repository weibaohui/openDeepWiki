package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/repository"
)

type DocumentService struct {
	cfg      *config.Config
	docRepo  repository.DocumentRepository
	repoRepo repository.RepoRepository
}

func NewDocumentService(cfg *config.Config, docRepo repository.DocumentRepository, repoRepo repository.RepoRepository) *DocumentService {
	return &DocumentService{
		cfg:      cfg,
		docRepo:  docRepo,
		repoRepo: repoRepo,
	}
}

type CreateDocumentRequest struct {
	RepositoryID uint   `json:"repository_id"`
	TaskID       uint   `json:"task_id"`
	Title        string `json:"title"`
	Filename     string `json:"filename"`
	Content      string `json:"content"`
	SortOrder    int    `json:"sort_order"`
}

func (s *DocumentService) Create(req CreateDocumentRequest) (*model.Document, error) {
	doc := &model.Document{
		RepositoryID: req.RepositoryID,
		TaskID:       req.TaskID,
		Title:        req.Title,
		Filename:     req.Filename,
		Content:      req.Content,
		SortOrder:    req.SortOrder,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.docRepo.Create(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) GetByRepository(repoID uint) ([]model.Document, error) {
	return s.docRepo.GetByRepository(repoID)
}

func (s *DocumentService) Get(id uint) (*model.Document, error) {
	return s.docRepo.Get(id)
}

func (s *DocumentService) Update(id uint, content string) (*model.Document, error) {
	doc, err := s.docRepo.Get(id)
	if err != nil {
		return nil, err
	}

	doc.Content = content
	doc.UpdatedAt = time.Now()
	if err := s.docRepo.Save(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) Delete(id uint) error {
	return s.docRepo.Delete(id)
}

func (s *DocumentService) DeleteByTaskID(taskID uint) error {
	return s.docRepo.DeleteByTaskID(taskID)
}

func (s *DocumentService) ExportAll(repoID uint) ([]byte, string, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, "", err
	}

	docs, err := s.docRepo.GetByRepository(repoID)
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
		f, err := zipWriter.Create(doc.Filename)
		if err != nil {
			return nil, "", err
		}
		f.Write([]byte(doc.Content))
	}

	if err := zipWriter.Close(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("%s-docs.zip", repo.Name)
	return buf.Bytes(), filename, nil
}

func (s *DocumentService) generateIndex(repoName string, docs []model.Document) string {
	content := fmt.Sprintf("# %s - 项目文档\n\n", repoName)
	content += "## 目录\n\n"

	for _, doc := range docs {
		content += fmt.Sprintf("- [%s](%s)\n", doc.Title, doc.Filename)
	}

	return content
}

func (s *DocumentService) GetIndex(repoID uint) (string, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return "", err
	}

	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		return "", err
	}

	return s.generateIndex(repo.Name, docs), nil
}
