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

type titleRewriter struct {
	factory  *adkagents.AgentFactory
	docRepo  repository.DocumentRepository
	taskRepo repository.TaskRepository
}

// NewTitleRewriter 创建标题重写服务
func NewTitleRewriter(cfg *config.Config, docRepo repository.DocumentRepository, taskRepo repository.TaskRepository) (*titleRewriter, error) {

	klog.V(6).Infof("[TitleRewriter] 创建服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[TitleRewriter] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &titleRewriter{
		factory:  factory,
		docRepo:  docRepo,
		taskRepo: taskRepo,
	}, nil
}

func (s *titleRewriter) Name() domain.WriterName {
	return domain.TitleRewriter
}

// Generate 生成标题
// title 传入标题没用，从数据库文档表读取
func (s *titleRewriter) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	klog.V(6).Infof("[%s] 开始处理文档 ID: %d", s.Name(), taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		klog.Errorf("[%s] 获取任务失败 taskID=%d: %v", s.Name(), taskID, err)
		return "", fmt.Errorf("get task failed: %w", err)
	}

	// 1. 获取文档
	doc, err := s.docRepo.Get(task.DocID)
	if err != nil {
		klog.Errorf("[%s] 获取文档失败 docID=%d: %v", s.Name(), task.DocID, err)
		return "", fmt.Errorf("get document failed: %w", err)
	}

	oldTitle := doc.Title
	content := doc.Content

	klog.V(6).Infof("[%s] 当前标题: %s", s.Name(), oldTitle)

	// 2. 调用 Agent
	agent, err := s.factory.Manager.CreateAgent(domain.AgentTitleRewriter)
	if err != nil {
		klog.Errorf("[%s] 创建 Agent '%s' 失败: %v", s.Name(), domain.AgentTitleRewriter, err)
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	prompt := fmt.Sprintf("当前标题: %s\n\n文档内容:\n%s", oldTitle, content)

	newTitle, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	})
	if err != nil {
		klog.Errorf("[%s] Agent 执行失败: %v", s.Name(), err)
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	newTitle = strings.TrimSpace(newTitle)
	// 去除可能的引号
	newTitle = strings.Trim(newTitle, "\"'`")
	// 去除可能的前缀（比如 "新标题："）
	newTitle = strings.TrimPrefix(newTitle, "新标题：")
	newTitle = strings.TrimPrefix(newTitle, "New Title:")
	newTitle = strings.TrimSpace(newTitle)

	if newTitle == "" {
		klog.Warningf("[%s] Agent 返回了空标题", s.Name())
		return "", fmt.Errorf("[%s] agent returned empty title", s.Name())
	}

	// 3. 比较并更新
	if newTitle != oldTitle {
		klog.V(6).Infof("[%s] 标题需要更新: '%s' -> '%s'", s.Name(), oldTitle, newTitle)
		doc.Title = newTitle
		if err := s.docRepo.Save(doc); err != nil {
			klog.Errorf("[%s] 保存文档失败: %v", s.Name(), err)
			return "", fmt.Errorf("save document failed: %w", err)
		}
		return newTitle, nil
	}

	klog.V(6).Infof("[%s] 标题无需更新", s.Name())
	return oldTitle, nil
}
