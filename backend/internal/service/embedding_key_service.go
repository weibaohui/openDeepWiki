package service

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

type EmbeddingKeyService struct {
	repo repository.EmbeddingKeyRepository
}

func NewEmbeddingKeyService(repo repository.EmbeddingKeyRepository) *EmbeddingKeyService {
	return &EmbeddingKeyService{repo: repo}
}

// Create 创建嵌入模型配置
func (s *EmbeddingKeyService) Create(ctx context.Context, key *model.EmbeddingKey) error {
	// 设置默认值
	if key.Priority == 0 {
		key.Priority = 0
	}
	if key.Status == "" {
		key.Status = "enabled"
	}
	if key.Timeout == 0 {
		key.Timeout = 30
	}
	if key.Dimension == 0 {
		key.Dimension = 1536
	}

	return s.repo.Create(ctx, key)
}

// GetByID 根据ID获取配置
func (s *EmbeddingKeyService) GetByID(ctx context.Context, id uint) (*model.EmbeddingKey, error) {
	return s.repo.GetByID(ctx, id)
}

// List 列出所有配置
func (s *EmbeddingKeyService) List(ctx context.Context) ([]model.EmbeddingKey, error) {
	return s.repo.List(ctx)
}

// GetAvailable 获取可用的配置
func (s *EmbeddingKeyService) GetAvailable(ctx context.Context) ([]model.EmbeddingKey, error) {
	return s.repo.GetAvailable(ctx)
}

// Update 更新配置
func (s *EmbeddingKeyService) Update(ctx context.Context, key *model.EmbeddingKey) error {
	return s.repo.Update(ctx, key)
}

// Delete 删除配置
func (s *EmbeddingKeyService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// Enable 启用配置
func (s *EmbeddingKeyService) Enable(ctx context.Context, id uint) error {
	return s.repo.SetStatus(ctx, id, "enabled")
}

// Disable 禁用配置
func (s *EmbeddingKeyService) Disable(ctx context.Context, id uint) error {
	return s.repo.SetStatus(ctx, id, "disabled")
}

// TestConnection 测试连接
func (s *EmbeddingKeyService) TestConnection(ctx context.Context, id uint) error {
	// 简单实现：获取配置并验证基本字段
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if key.APIKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if key.BaseURL == "" {
		return fmt.Errorf("Base URL 不能为空")
	}
	if key.Model == "" {
		return fmt.Errorf("Model 不能为空")
	}

	// TODO: 可以添加实际的 API 调用测试
	return nil
}

// GetUsageStats 获取使用统计
func (s *EmbeddingKeyService) GetUsageStats(ctx context.Context) (map[string]interface{}, error) {
	keys, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var totalRequests, totalErrors int
	var activeKeys int

	for _, key := range keys {
		if key.IsAvailable() {
			activeKeys++
		}
		totalRequests += key.RequestCount
		totalErrors += key.ErrorCount
	}

	return map[string]interface{}{
		"total_keys":      len(keys),
		"active_keys":     activeKeys,
		"total_requests":  totalRequests,
		"total_errors":    totalErrors,
		"last_updated":    time.Now(),
	}, nil
}