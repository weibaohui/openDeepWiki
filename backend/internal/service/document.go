package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

type DocumentService struct {
	cfg        *config.Config
	docRepo    repository.DocumentRepository
	repoRepo   repository.RepoRepository
	ratingRepo repository.DocumentRatingRepository
}

// NewDocumentService 创建文档服务
func NewDocumentService(cfg *config.Config, docRepo repository.DocumentRepository, repoRepo repository.RepoRepository, ratingRepo repository.DocumentRatingRepository) *DocumentService {
	return &DocumentService{
		cfg:        cfg,
		docRepo:    docRepo,
		repoRepo:   repoRepo,
		ratingRepo: ratingRepo,
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

	if err := s.docRepo.CreateVersioned(doc); err != nil {
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

func (s *DocumentService) GetVersions(docID uint) ([]model.Document, error) {
	doc, err := s.docRepo.Get(docID)
	if err != nil {
		return nil, err
	}
	return s.docRepo.GetByTaskID(doc.TaskID)
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

// SubmitRating 提交文档评分并返回统计信息
func (s *DocumentService) SubmitRating(documentID uint, score int) (*model.DocumentRatingStats, error) {
	if s.ratingRepo == nil {
		return nil, fmt.Errorf("rating repository not configured")
	}
	if score < 1 || score > 5 {
		return nil, fmt.Errorf("score must be between 1 and 5")
	}

	klog.V(6).Infof("SubmitRating: document_id=%d score=%d", documentID, score)

	latest, err := s.ratingRepo.GetLatestByDocumentID(documentID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.Score == score && time.Since(latest.CreatedAt) <= 10*time.Second {
		klog.V(6).Infof("SubmitRating: duplicate ignored document_id=%d score=%d", documentID, score)
		return s.GetRatingStats(documentID)
	}

	now := time.Now()
	rating := &model.DocumentRating{
		DocumentID: documentID,
		Score:      score,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.ratingRepo.Create(rating); err != nil {
		return nil, err
	}

	klog.V(6).Infof("SubmitRating: created rating_id=%d document_id=%d", rating.ID, documentID)
	return s.GetRatingStats(documentID)
}

// GetRatingStats 获取文档评分统计信息
func (s *DocumentService) GetRatingStats(documentID uint) (*model.DocumentRatingStats, error) {
	if s.ratingRepo == nil {
		return nil, fmt.Errorf("rating repository not configured")
	}
	stats, err := s.ratingRepo.GetStatsByDocumentID(documentID)
	if err != nil {
		return nil, err
	}
	stats.AverageScore = math.Round(stats.AverageScore*10) / 10
	klog.V(6).Infof("GetRatingStats: document_id=%d average=%.1f count=%d", documentID, stats.AverageScore, stats.RatingCount)
	return stats, nil
}
