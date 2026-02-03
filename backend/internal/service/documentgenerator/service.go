package documentgenerator

import (
	"context"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"k8s.io/klog/v2"
)

// Service 文档生成服务
// 基于 Eino ADK 实现，用于分析代码并生成技术文档
type Service struct {
	cfg      *config.Config
	workflow *DocumentGeneratorWorkflow
}

// New 创建文档生成服务实例
func New(cfg *config.Config) (*Service, error) {
	klog.V(6).Infof("[documentgenerator.New] 开始创建文档生成服务")

	workflow, err := NewDocumentGeneratorWorkflow(cfg)
	if err != nil {
		klog.Errorf("[documentgenerator.New] 创建 workflow 失败: %v", err)
		return nil, fmt.Errorf("创建 workflow 失败: %w", err)
	}

	return &Service{
		cfg:      cfg,
		workflow: workflow,
	}, nil
}

// Generate 分析仓库代码并生成文档
// ctx: 上下文
// localPath: 仓库本地路径
// title: 文档标题/任务类型
// 返回: 生成的文档内容（Markdown 格式）
func (s *Service) Generate(ctx context.Context, localPath string, title string) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("本地路径为空")
	}
	if title == "" {
		return "", fmt.Errorf("文档标题为空")
	}

	klog.V(6).Infof("[documentgenerator.Generate] 开始生成文档: localPath=%s, title=%s", localPath, title)

	result, err := s.workflow.Run(ctx, localPath, title)
	if err != nil {
		klog.Errorf("[documentgenerator.Generate] 生成失败: %v", err)
		return "", err
	}

	klog.V(6).Infof("[documentgenerator.Generate] 生成成功，文档内容长度: %d", len(result.Content))
	klog.V(6).Infof("[documentgenerator.Generate] 分析摘要: %s", result.AnalysisSummary)

	return result.Content, nil
}
