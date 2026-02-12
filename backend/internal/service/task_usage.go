package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// TaskUsageService 任务用量服务接口
type TaskUsageService interface {
	RecordUsage(ctx context.Context, taskID uint, apiKeyName string, usage *schema.TokenUsage) error
}

type taskUsageService struct {
	repo repository.TaskUsageRepository
}

// NewTaskUsageService 创建任务用量服务
func NewTaskUsageService(repo repository.TaskUsageRepository) TaskUsageService {
	return &taskUsageService{repo: repo}
}

// RecordUsage 记录任务的 token 使用量
func (s *taskUsageService) RecordUsage(ctx context.Context, taskID uint, apiKeyName string, usage *schema.TokenUsage) error {
	if usage == nil {
		klog.V(6).Infof("任务用量记录跳过：usage 为空")
		return nil
	}
	if taskID == 0 {
		klog.V(6).Infof("任务用量记录失败：taskID 为空")
		return fmt.Errorf("taskID 为空")
	}

	// 将 SDK 的 usage 结构映射为数据库模型字段
	record := &model.TaskUsage{
		TaskID:           taskID,
		APIKeyName:       apiKeyName,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CachedTokens:     usage.PromptTokenDetails.CachedTokens,
		ReasoningTokens:  usage.CompletionTokensDetails.ReasoningTokens,
	}

	if err := s.repo.Create(ctx, record); err != nil {
		klog.V(6).Infof("任务用量记录失败：taskID=%d, 模型=%s, err=%v", taskID, apiKeyName, err)
		return err
	}
	klog.V(6).Infof("任务用量记录成功：taskID=%d, 模型=%s", taskID, apiKeyName)
	return nil
}
