package adk

import (
	"context"
	"fmt"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"k8s.io/klog/v2"
)

// RepoDocService ADK 模式的仓库文档解析服务
// 使用 Eino ADK 原生的 SequentialAgent 和 Runner
type RepoDocService struct {
	cfg      *config.Config
	basePath string           // 仓库存储的基础路径
	workflow *RepoDocWorkflow // Workflow 实例
}

// NewRepoDocService 创建 ADK 服务实例
// basePath: 仓库存储的基础路径
// llmCfg: LLM 配置
// 返回: ADKRepoDocService 实例或错误
func NewRepoDocService(cfg *config.Config) (*RepoDocService, error) {
	klog.V(6).Infof("[NewADKRepoDocService] 开始创建 ADK 服务 ")
	// 创建 Workflow
	workflow, err := NewRepoDocWorkflow(cfg)
	if err != nil {
		klog.Errorf("[NewADKRepoDocService] 创建 Workflow 失败: %v", err)
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	klog.V(6).Infof("[NewADKRepoDocService] ADK 服务创建成功")

	return &RepoDocService{
		cfg:      cfg,
		workflow: workflow,
	}, nil
}

// ParseRepo 解析仓库，生成文档
// ctx: 上下文，可用于超时控制
// repoURL: 仓库 Git URL
// 返回: 解析结果或错误
func (s *RepoDocService) ParseRepo(ctx context.Context, localPath string) (*einodoc.RepoDocResult, error) {
	klog.V(6).Infof("[ADKRepoDocService.ParseRepo] 开始解析仓库: localPath=%s", localPath)

	// 执行 Workflow
	result, err := s.workflow.Run(ctx, localPath)
	if err != nil {
		klog.Errorf("[ADKRepoDocService.ParseRepo] Workflow 执行失败: %v", err)
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	klog.V(6).Infof("[ADKRepoDocService.ParseRepo] 解析成功: sections=%d, document_length=%d",
		result.SectionsCount, len(result.Document))

	return result, nil
}
