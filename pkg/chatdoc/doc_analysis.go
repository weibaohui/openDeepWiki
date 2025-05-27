package service

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"k8s.io/klog/v2"
)

type docAnalysisService struct {
	parent *docService
}

func (s *docService) AnalysisService() *docAnalysisService {
	return &docAnalysisService{
		parent: s,
	}
}

// Create 创建新的文档解读实例
func (s *docAnalysisService) Create(ctx context.Context) (*models.DocAnalysis, error) {
	analysis := &models.DocAnalysis{
		RepoID:    s.parent.repo.ID,
		Status:    "pending",
		StartTime: time.Now(),
	}

	if err := dao.DB().Save(analysis).Error; err != nil {
		return nil, err
	}
	klog.V(6).Infof("Created new DocAnalysis instance: %v", analysis)

	return analysis, nil
}

// UpdateStatus 更新文档解读实例状态
func (s *docAnalysisService) UpdateStatus(ctx context.Context, analysis *models.DocAnalysis, status string, result string, err error) error {
	updates := map[string]interface{}{
		"status": status,
		"result": result,
	}

	if status == "completed" || status == "failed" {
		updates["end_time"] = time.Now()
	}

	if err != nil {
		updates["error_msg"] = err.Error()
	}

	return dao.DB().Model(analysis).Updates(updates).Error
}

// UpdateReadmePath 更新README文件路径
func (s *docAnalysisService) UpdateReadmePath(ctx context.Context, analysis *models.DocAnalysis, path string) error {
	return dao.DB().Model(analysis).Update("readme_path", path).Error
}

// GetLatest 获取最新的文档解读实例
func (s *docAnalysisService) GetLatest(ctx context.Context) (*models.DocAnalysis, error) {
	var analysis models.DocAnalysis
	err := dao.DB().Where("repo_id = ?", s.parent.repo.ID).Order("created_at desc").First(&analysis).Error
	return &analysis, err
}

// GetByAnalysisID 根据ID获取文档解读实例
func (s *docAnalysisService) GetByAnalysisID(analysisID string) (*models.DocAnalysis, error) {
	idUint, err := utils.StringToUintID(analysisID)
	if err != nil {
		return nil, fmt.Errorf("invalid analysis ID: %w", err)
	}

	var analysis models.DocAnalysis
	err = dao.DB().Where("id = ?", idUint).First(&analysis).Error
	if err != nil {
		return nil, err
	}

	return &analysis, nil
}
