package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
	"k8s.io/klog/v2"
)

// CreateDocWriteTask 创建文档和任务，并建立双向关联
// 1. 创建文档
// 2. 创建任务
// 3. 更新文档关联的任务ID
func (s *TaskService) CreateDocWriteTask(ctx context.Context, repoID uint, title string, sortOrder int, writerNames ...domain.WriterName) (*model.Task, error) {
	docTitle := strings.TrimSpace(title)
	if len([]rune(docTitle)) > 20 {
		docTitle = string([]rune(docTitle)[:20])
	}
	doc, err := s.docService.Create(CreateDocumentRequest{
		RepositoryID: repoID,
		Title:        docTitle, //文章标题，限制长度
		Filename:     docTitle + ".md",
		Content:      title, //文档内容，初始为空，后续会被填充
		SortOrder:    sortOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("[CreateDocWriteTask] 创建文档失败: %w", err)
	}

	task := &model.Task{
		RepositoryID: repoID,
		DocID:        doc.ID,
		Title:        title, //任务标题，不限制长度，prompt会提取文档标题作为提示词一部分
		WriterName:   domain.DefaultWriter,
		TaskType:     domain.DocWrite,
		Status:       string(statemachine.TaskStatusPending),
		SortOrder:    sortOrder,
	}

	//指定Writer
	if len(writerNames) > 0 {
		task.WriterName = writerNames[0]
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("[CreateDocWriteTask] 创建任务失败: %w", err)
	}

	if err := s.docService.UpdateTaskID(doc.ID, task.ID); err != nil {
		// 记录日志但不返回错误，因为任务和文档已创建
		klog.Errorf("[CreateDocWriteTask] 更新文档关联的任务ID失败: docID=%d, taskID=%d, error=%v", doc.ID, task.ID, err)
	}

	return task, nil
}

// CreateTocWriteTask 创建目录任务，无需创建文档
func (s *TaskService) CreateTocWriteTask(ctx context.Context, repoID uint, title string, sortOrder int) (*model.Task, error) {
	// 创建目录任务，无需创建文档
	task := &model.Task{
		RepositoryID: repoID,
		Title:        title,
		WriterName:   domain.TocWriter,
		TaskType:     domain.TocWrite,
		Status:       string(statemachine.TaskStatusPending),
		SortOrder:    sortOrder,
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("[CreateTocWriteTask] 创建任务失败: %w", err)
	}

	return task, nil
}

// CreateTitleRewriteTask 创建标题重写任务
func (s *TaskService) CreateTitleRewriteTask(ctx context.Context, repoID uint, title string, runAfter uint, docId uint, sortOrder int) (*model.Task, error) {
	// 创建标题重写任务
	task := &model.Task{
		RepositoryID: repoID,
		Title:        title,
		DocID:        docId,
		WriterName:   domain.TitleRewriter,
		TaskType:     domain.TitleRewrite,
		RunAfter:     runAfter,
		Status:       string(statemachine.TaskStatusPending),
		SortOrder:    sortOrder,
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("[CreateTitleRewriteTask] 创建任务失败: %w", err)
	}

	return task, nil
}

// CreateUserRequestTask 创建用户请求任务
// 1. 创建一个分析任务，分析任务的结果会被用于创建文档
// 2. 创建一个titleRewrite 任务，将标题进行重写
func (s *TaskService) CreateUserRequestTask(ctx context.Context, repoID uint, content string, sortOrder int) (*model.Task, error) {
	// 创建用户请求任务
	// 首先创建一个分析任务，分析任务的结果会被用于创建文档
	// 创建一个titleRewrite 任务，将标题进行重写

	task1, err := s.CreateDocWriteTask(ctx, repoID, content, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("[CreateUserRequestTask] 创建任务失败: %w", err)
	}

	// 创建一个titleRewrite 任务，将标题进行重写
	task2, err := s.CreateTitleRewriteTask(ctx, repoID, content, task1.ID, task1.DocID, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("[CreateUserRequestTask] 创建任务失败: %w", err)
	}

	klog.V(6).Infof("[CreateUserRequestTask] 任务入队成功: taskID=%d, titleRewriteTaskID=%d", task1.ID, task2.ID)
	return task1, nil

}
 