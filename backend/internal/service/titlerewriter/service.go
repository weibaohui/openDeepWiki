package titlerewriter

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

type Service struct {
	factory *adkagents.AgentFactory
	docRepo repository.DocumentRepository
}

// New 创建标题重写服务
func New(cfg *config.Config, docRepo repository.DocumentRepository) (*Service, error) {
	klog.V(6).Infof("[TitleRewriter] 创建服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[TitleRewriter] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &Service{
		factory: factory,
		docRepo: docRepo,
	}, nil
}

// RewriteTitle 根据 docID 分析并重写文档标题
// 返回值: (oldTitle, newTitle, updated, error)
func (s *Service) RewriteTitle(ctx context.Context, docID uint) (string, string, bool, error) {
	klog.V(6).Infof("[TitleRewriter] 开始处理文档 ID: %d", docID)

	// 1. 获取文档
	doc, err := s.docRepo.Get(docID)
	if err != nil {
		klog.Errorf("[TitleRewriter] 获取文档失败 docID=%d: %v", docID, err)
		return "", "", false, fmt.Errorf("get document failed: %w", err)
	}

	oldTitle := doc.Title
	content := doc.Content
	

	klog.V(6).Infof("[TitleRewriter] 当前标题: %s", oldTitle)

	// 2. 调用 Agent
	agent, err := s.factory.Manager.CreateAgent("title_rewriter")
	if err != nil {
		klog.Errorf("[TitleRewriter] 创建 Agent 'title_rewriter' 失败: %v", err)
		return oldTitle, "", false, fmt.Errorf("create agent failed: %w", err)
	}

	prompt := fmt.Sprintf("当前标题: %s\n\n文档内容:\n%s", oldTitle, content)

	newTitle, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	})
	if err != nil {
		klog.Errorf("[TitleRewriter] Agent 执行失败: %v", err)
		return oldTitle, "", false, fmt.Errorf("agent execution failed: %w", err)
	}

	newTitle = strings.TrimSpace(newTitle)
	// 去除可能的引号
	newTitle = strings.Trim(newTitle, "\"'`")
	// 去除可能的前缀（比如 "新标题："）
	newTitle = strings.TrimPrefix(newTitle, "新标题：")
	newTitle = strings.TrimPrefix(newTitle, "New Title:")
	newTitle = strings.TrimSpace(newTitle)

	if newTitle == "" {
		klog.Warningf("[TitleRewriter] Agent 返回了空标题")
		return oldTitle, "", false, fmt.Errorf("agent returned empty title")
	}

	// 3. 比较并更新
	if newTitle != oldTitle {
		klog.V(6).Infof("[TitleRewriter] 标题需要更新: '%s' -> '%s'", oldTitle, newTitle)
		doc.Title = newTitle
		if err := s.docRepo.Save(doc); err != nil {
			klog.Errorf("[TitleRewriter] 保存文档失败: %v", err)
			return oldTitle, newTitle, false, fmt.Errorf("save document failed: %w", err)
		}
		return oldTitle, newTitle, true, nil
	}

	klog.V(6).Infof("[TitleRewriter] 标题无需更新")
	return oldTitle, newTitle, false, nil
}
