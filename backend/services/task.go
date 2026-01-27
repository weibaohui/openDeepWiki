package services

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/models"
	"github.com/opendeepwiki/backend/pkg/llm"
	"github.com/opendeepwiki/backend/services/analyzer"
)

type TaskService struct {
	cfg *config.Config
}

func NewTaskService(cfg *config.Config) *TaskService {
	return &TaskService{cfg: cfg}
}

func (s *TaskService) GetByRepository(repoID uint) ([]models.Task, error) {
	var tasks []models.Task
	err := models.DB.Where("repository_id = ?", repoID).Order("sort_order").Find(&tasks).Error
	return tasks, err
}

func (s *TaskService) Get(id uint) (*models.Task, error) {
	var task models.Task
	err := models.DB.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskService) Run(taskID uint) error {
	klog.V(6).Infof("开始执行任务: taskID=%d", taskID)

	var task models.Task
	if err := models.DB.First(&task, taskID).Error; err != nil {
		klog.V(6).Infof("获取任务失败: taskID=%d, error=%v", taskID, err)
		return err
	}
	klog.V(6).Infof("任务信息: taskID=%d, type=%s, title=%s", task.ID, task.Type, task.Title)

	var repo models.Repository
	if err := models.DB.First(&repo, task.RepositoryID).Error; err != nil {
		klog.V(6).Infof("获取仓库失败: repoID=%d, error=%v", task.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("仓库信息: repoID=%d, name=%s, localPath=%s", repo.ID, repo.Name, repo.LocalPath)

	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	task.ErrorMsg = ""
	models.DB.Save(&task)
	klog.V(6).Infof("任务状态更新为 running: taskID=%d", taskID)

	klog.V(6).Infof("开始静态分析: repoPath=%s", repo.LocalPath)
	projectInfo, err := analyzer.Analyze(repo.LocalPath)
	if err != nil {
		klog.V(6).Infof("静态分析失败: error=%v", err)
		s.failTask(&task, fmt.Sprintf("静态分析失败: %v", err))
		return err
	}
	klog.V(6).Infof("静态分析完成: projectType=%s, totalFiles=%d, totalLines=%d",
		projectInfo.Type, projectInfo.TotalFiles, projectInfo.TotalLines)

	klog.V(6).Infof("初始化 LLM 客户端: apiURL=%s, model=%s, maxTokens=%d",
		s.cfg.LLM.APIURL, s.cfg.LLM.Model, s.cfg.LLM.MaxTokens)
	llmClient := llm.NewClient(
		s.cfg.LLM.APIURL,
		s.cfg.LLM.APIKey,
		s.cfg.LLM.Model,
		s.cfg.LLM.MaxTokens,
	)

	llmAnalyzer := analyzer.NewLLMAnalyzer(llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	klog.V(6).Infof("开始 LLM 分析: taskType=%s", task.Type)
	content, err := llmAnalyzer.Analyze(ctx, analyzer.AnalyzeRequest{
		TaskType:    task.Type,
		ProjectInfo: projectInfo,
	})

	if err != nil {
		klog.V(6).Infof("LLM 分析失败: error=%v", err)
		s.failTask(&task, fmt.Sprintf("LLM 分析失败: %v", err))
		return err
	}
	klog.V(6).Infof("LLM 分析完成: contentLength=%d", len(content))

	docService := NewDocumentService(s.cfg)
	taskDef := getTaskDefinition(task.Type)

	klog.V(6).Infof("保存文档: title=%s, filename=%s", taskDef.Title, taskDef.Filename)
	_, err = docService.Create(CreateDocumentRequest{
		RepositoryID: task.RepositoryID,
		TaskID:       task.ID,
		Title:        taskDef.Title,
		Filename:     taskDef.Filename,
		Content:      content,
		SortOrder:    taskDef.SortOrder,
	})

	if err != nil {
		klog.V(6).Infof("保存文档失败: error=%v", err)
		s.failTask(&task, fmt.Sprintf("保存文档失败: %v", err))
		return err
	}
	klog.V(6).Infof("文档保存成功")

	completedAt := time.Now()
	task.Status = "completed"
	task.CompletedAt = &completedAt
	models.DB.Save(&task)

	duration := completedAt.Sub(now)
	klog.V(6).Infof("任务执行完成: taskID=%d, duration=%v", taskID, duration)

	return nil
}

func (s *TaskService) failTask(task *models.Task, errMsg string) {
	klog.V(6).Infof("任务失败: taskID=%d, error=%s", task.ID, errMsg)
	task.Status = "failed"
	task.ErrorMsg = errMsg
	models.DB.Save(task)
}

func getTaskDefinition(taskType string) struct {
	Type      string
	Title     string
	Filename  string
	SortOrder int
} {
	for _, t := range models.TaskTypes {
		if t.Type == taskType {
			return t
		}
	}
	return models.TaskTypes[0]
}

func (s *TaskService) Reset(taskID uint) error {
	klog.V(6).Infof("重置任务: taskID=%d", taskID)
	var task models.Task
	if err := models.DB.First(&task, taskID).Error; err != nil {
		return err
	}

	models.DB.Where("task_id = ?", taskID).Delete(&models.Document{})

	task.Status = "pending"
	task.ErrorMsg = ""
	task.StartedAt = nil
	task.CompletedAt = nil
	return models.DB.Save(&task).Error
}

// ForceReset 强制重置任务，无论当前状态
func (s *TaskService) ForceReset(taskID uint) error {
	klog.V(6).Infof("强制重置任务: taskID=%d", taskID)
	var task models.Task
	if err := models.DB.First(&task, taskID).Error; err != nil {
		return err
	}

	klog.V(6).Infof("任务当前状态: taskID=%d, status=%s, startedAt=%v",
		taskID, task.Status, task.StartedAt)

	// 删除关联的文档
	models.DB.Where("task_id = ?", taskID).Delete(&models.Document{})

	// 重置任务状态
	task.Status = "pending"
	task.ErrorMsg = ""
	task.StartedAt = nil
	task.CompletedAt = nil

	klog.V(6).Infof("任务已强制重置: taskID=%d", taskID)
	return models.DB.Save(&task).Error
}

// CleanupStuckTasks 清理卡住的任务（运行超过指定时间的任务）
func (s *TaskService) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	klog.V(6).Infof("开始清理卡住的任务: timeout=%v", timeout)

	cutoff := time.Now().Add(-timeout)

	result := models.DB.Model(&models.Task{}).
		Where("status = ? AND started_at < ?", "running", cutoff).
		Updates(map[string]interface{}{
			"status":    "failed",
			"error_msg": fmt.Sprintf("任务超时（超过 %v），已自动标记为失败", timeout),
		})

	if result.Error != nil {
		klog.V(6).Infof("清理卡住任务失败: error=%v", result.Error)
		return 0, result.Error
	}

	klog.V(6).Infof("清理卡住任务完成: affected=%d", result.RowsAffected)
	return result.RowsAffected, nil
}

// GetStuckTasks 获取卡住的任务列表
func (s *TaskService) GetStuckTasks(timeout time.Duration) ([]models.Task, error) {
	cutoff := time.Now().Add(-timeout)

	var tasks []models.Task
	err := models.DB.Where("status = ? AND started_at < ?", "running", cutoff).Find(&tasks).Error
	return tasks, err
}
