package einodoc

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// RepoDocService 仓库文档解析服务接口
type RepoDocService interface {
	// ParseRepo 解析仓库，生成文档
	ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error)
}

// repoDocService 服务实现
type repoDocService struct {
	basePath  string
	llmClient *llm.Client
	chain     *RepoDocChain
}

// NewRepoDocService 创建新的服务实例
func NewRepoDocService(basePath string, llmClient *llm.Client) (RepoDocService, error) {
	// 创建 ChatModel
	chatModel := NewLLMChatModel(llmClient)

	// 创建 Chain
	chain, err := NewRepoDocChain(basePath, chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	return &repoDocService{
		basePath:  basePath,
		llmClient: llmClient,
		chain:     chain,
	}, nil
}

// ParseRepo 解析仓库
func (s *repoDocService) ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error) {
	klog.V(6).Infof("开始解析仓库: repoURL=%s", repoURL)

	input := WorkflowInput{
		RepoURL: repoURL,
	}

	result, err := s.chain.Run(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("parse repo failed: %w", err)
	}

	klog.V(6).Infof("仓库解析完成: repoURL=%s, sections=%d", repoURL, result.SectionsCount)
	return result, nil
}

// EinoRepoDocService 高级服务实现，支持更多配置选项
type EinoRepoDocService struct {
	basePath   string
	llmClient  *llm.Client
	chatModel  model.ChatModel
	chain      *RepoDocChain
}

// NewEinoRepoDocService 创建高级服务实例
func NewEinoRepoDocService(basePath string, llmClient *llm.Client) (*EinoRepoDocService, error) {
	chatModel := NewLLMChatModel(llmClient)

	chain, err := NewRepoDocChain(basePath, chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	return &EinoRepoDocService{
		basePath:  basePath,
		llmClient: llmClient,
		chatModel: chatModel,
		chain:     chain,
	}, nil
}

// ParseRepo 实现 RepoDocService 接口
func (s *EinoRepoDocService) ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error) {
	return s.chain.Run(ctx, WorkflowInput{RepoURL: repoURL})
}

// GetChatModel 获取 ChatModel（用于扩展）
func (s *EinoRepoDocService) GetChatModel() model.ChatModel {
	return s.chatModel
}

// GetTools 获取工具列表（用于扩展）
func (s *EinoRepoDocService) GetTools() []tool.BaseTool {
	return CreateTools(s.basePath)
}
