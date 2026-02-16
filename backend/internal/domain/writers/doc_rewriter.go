package writers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

type docRewriter struct {
	factory  *adkagents.AgentFactory
	docRepo  repository.DocumentRepository
	taskRepo repository.TaskRepository
}

// NewDocRewriter 创建文档内容重写服务
func NewDocRewriter(cfg *config.Config, docRepo repository.DocumentRepository, taskRepo repository.TaskRepository) (*docRewriter, error) {
	klog.V(6).Infof("[DocRewriter] 创建服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[DocRewriter] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &docRewriter{
		factory:  factory,
		docRepo:  docRepo,
		taskRepo: taskRepo,
	}, nil
}

// Name 返回写入器名称
func (s *docRewriter) Name() domain.WriterName {
	return domain.DocRewriter
}

// Generate 生成文档重写后的内容
func (s *docRewriter) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	klog.V(6).Infof("[%s] 开始处理文档重写任务: taskID=%d", s.Name(), taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		klog.Errorf("[%s] 获取任务失败 taskID=%d: %v", s.Name(), taskID, err)
		return "", fmt.Errorf("get task failed: %w", err)
	}

	doc, err := s.docRepo.Get(task.DocID)
	if err != nil {
		klog.Errorf("[%s] 获取文档失败 docID=%d: %v", s.Name(), task.DocID, err)
		return "", fmt.Errorf("get document failed: %w", err)
	}

	guide := strings.TrimSpace(task.Outline)
	if guide == "" {
		klog.Errorf("[%s] 任务重写指引为空 taskID=%d", s.Name(), taskID)
		return "", fmt.Errorf("rewrite guide is empty")
	}

	agent, err := s.factory.Manager.CreateAgent(domain.AgentDocRewriter)
	if err != nil {
		klog.Errorf("[%s] 创建 Agent '%s' 失败: %v", s.Name(), domain.AgentDocRewriter, err)
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	prompt := fmt.Sprintf("文档标题: %s \t DocId=%d \n重写指引:\n%s", doc.Title, doc.ID, guide)

	newContent, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	})
	if err != nil {
		klog.Errorf("[%s] Agent 执行失败: %v", s.Name(), err)
		return "", fmt.Errorf("agent execution failed: %w", err)
	}
	return newContent, nil
}
