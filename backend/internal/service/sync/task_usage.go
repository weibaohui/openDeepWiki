package syncservice

import (
	"context"
	"fmt"
	"time"

	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

// TaskUsageManager 管理任务用量
type TaskUsageManager struct {
	taskUsageRepo repository.TaskUsageRepository
}

// NewTaskUsageManager 创建新的任务用量管理器
func NewTaskUsageManager(taskUsageRepo repository.TaskUsageRepository) *TaskUsageManager {
	return &TaskUsageManager{
		taskUsageRepo: taskUsageRepo,
	}
}

// GetByTaskID 获取任务的用量记录列表
func (m *TaskUsageManager) GetByTaskID(ctx context.Context, taskID uint) ([]model.TaskUsage, error) {
	return m.taskUsageRepo.GetByTaskIDList(ctx, taskID)
}

// Create 创建或覆盖任务用量记录
func (m *TaskUsageManager) Create(ctx context.Context, req syncdto.TaskUsageCreateRequest) (*model.TaskUsage, error) {
	if len(req.TaskUsages) > 0 {
		return m.createBatch(ctx, req)
	}
	return m.createSingle(ctx, req)
}

func (m *TaskUsageManager) createBatch(ctx context.Context, req syncdto.TaskUsageCreateRequest) (*model.TaskUsage, error) {
	taskID := req.TaskID
	usages := make([]model.TaskUsage, 0, len(req.TaskUsages))
	for _, item := range req.TaskUsages {
		if taskID == 0 {
			taskID = item.TaskID
		}
		if item.TaskID != taskID {
			return nil, fmt.Errorf("task_id 不一致: %d != %d", item.TaskID, taskID)
		}
		createdAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
		if err != nil {
			createdAt = time.Now()
		}
		usages = append(usages, model.TaskUsage{
			ID:               0,
			TaskID:           taskID,
			APIKeyName:       item.APIKeyName,
			PromptTokens:     item.PromptTokens,
			CompletionTokens: item.CompletionTokens,
			TotalTokens:      item.TotalTokens,
			CachedTokens:     item.CachedTokens,
			ReasoningTokens:  item.ReasoningTokens,
			CreatedAt:        createdAt,
		})
	}
	if err := m.taskUsageRepo.UpsertMany(ctx, usages); err != nil {
		return nil, err
	}
	return &model.TaskUsage{TaskID: taskID}, nil
}

func (m *TaskUsageManager) createSingle(ctx context.Context, req syncdto.TaskUsageCreateRequest) (*model.TaskUsage, error) {
	if req.TaskID == 0 {
		return nil, fmt.Errorf("task_id 不能为空")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, req.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}
	usage := &model.TaskUsage{
		TaskID:           req.TaskID,
		APIKeyName:       req.APIKeyName,
		PromptTokens:     req.PromptTokens,
		CompletionTokens: req.CompletionTokens,
		TotalTokens:      req.TotalTokens,
		CachedTokens:     req.CachedTokens,
		ReasoningTokens:  req.ReasoningTokens,
		CreatedAt:        createdAt,
	}
	if err := m.taskUsageRepo.Upsert(ctx, usage); err != nil {
		return nil, err
	}
	return usage, nil
}

// ToTaskUsageData 将 model.TaskUsage 转换为 TaskUsageData
func ToTaskUsageData(usages []model.TaskUsage) []TaskUsageData {
	result := make([]TaskUsageData, 0, len(usages))
	for _, u := range usages {
		result = append(result, TaskUsageData{
			TaskID:           u.TaskID,
			APIKeyName:       u.APIKeyName,
			PromptTokens:     u.PromptTokens,
			CompletionTokens: u.CompletionTokens,
			TotalTokens:      u.TotalTokens,
			CachedTokens:     u.CachedTokens,
			ReasoningTokens:  u.ReasoningTokens,
			CreatedAt:        u.CreatedAt,
		})
	}
	return result
}
