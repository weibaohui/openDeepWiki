package documentgenerator

import (
	"context"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"k8s.io/klog/v2"
)

// DocumentGeneratorService 文档生成服务
type DocumentGeneratorService struct {
	cfg      *config.Config
	workflow *DocumentGeneratorWorkflow
}

// NewDocumentGeneratorService 创建文档生成服务
func NewDocumentGeneratorService(cfg *config.Config) (*DocumentGeneratorService, error) {
	workflow, err := NewDocumentGeneratorWorkflow(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}
	return &DocumentGeneratorService{
		cfg:      cfg,
		workflow: workflow,
	}, nil
}

// Generate 生成文档
// localPath: 仓库本地路径
// taskType: 任务/文档类型
// 返回: 生成的文档内容
func (s *DocumentGeneratorService) Generate(ctx context.Context, localPath string, title string) (string, error) {
	klog.V(6).Infof("[DocumentGeneratorService.Generate] 开始生成文档: localPath=%s, title=%s", localPath, title)

	result, err := s.workflow.Run(ctx, localPath, title)
	if err != nil {
		klog.Errorf("[DocumentGeneratorService.Generate] 生成失败: %v", err)
		return "", err
	}

	klog.V(6).Infof("[DocumentGeneratorService.Generate] 生成成功: summary=%s", result.AnalysisSummary)
	return result.Content, nil
}
