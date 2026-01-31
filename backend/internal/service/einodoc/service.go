package einodoc

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
)

// RepoDocService 仓库文档解析服务接口
// 对外提供统一的仓库文档生成能力
type RepoDocService interface {
	// ParseRepo 解析仓库，生成文档
	// ctx: 上下文，可用于超时控制
	// repoURL: 仓库 Git URL
	// 返回: 解析结果或错误
	ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error)
}

// repoDocService 服务实现
// 使用 Eino Chain 实现文档解析流程
type repoDocService struct {
	basePath  string        // 仓库存储的基础路径
	llmClient *llm.Client   // LLM 客户端
	chain     *RepoDocChain // Eino Chain 实例
}

// NewRepoDocService 创建新的服务实例
// basePath: 仓库存储的基础路径
// llmClient: LLM 客户端实例
// 返回: RepoDocService 接口实例或错误
func NewRepoDocService(basePath string, llmClient *llm.Client) (RepoDocService, error) {
	klog.V(6).Infof("[NewRepoDocService] 开始创建 RepoDocService: basePath=%s", basePath)

	// 创建 ChatModel
	klog.V(6).Infof("[NewRepoDocService] 创建 ChatModel")
	chatModel := NewLLMChatModel(llmClient)

	// 创建 Chain
	klog.V(6).Infof("[NewRepoDocService] 创建 RepoDocChain")
	chain, err := NewRepoDocChain(basePath, chatModel)
	if err != nil {
		klog.Errorf("[NewRepoDocService] 创建 RepoDocChain 失败: %v", err)
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	klog.V(6).Infof("[NewRepoDocService] RepoDocService 创建成功")
	return &repoDocService{
		basePath:  basePath,
		llmClient: llmClient,
		chain:     chain,
	}, nil
}

// ParseRepo 解析仓库
// 调用 Eino Chain 执行完整的文档解析流程
// ctx: 上下文
// repoURL: 仓库 Git URL
// 返回: 解析结果或错误
func (s *repoDocService) ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error) {
	klog.V(6).Infof("[repoDocService.ParseRepo] 开始解析仓库: repoURL=%s", repoURL)

	input := WorkflowInput{
		RepoURL: repoURL,
	}
	klog.V(6).Infof("[repoDocService.ParseRepo] 构建 WorkflowInput: %+v", input)

	result, err := s.chain.Run(ctx, input)
	if err != nil {
		klog.Errorf("[repoDocService.ParseRepo] 解析仓库失败: repoURL=%s, error=%v", repoURL, err)
		return nil, err
	}

	klog.V(6).Infof("[repoDocService.ParseRepo] 解析仓库成功: repoURL=%s, documentLength=%d, sections=%d",
		repoURL, len(result.Document), result.SectionsCount)
	return result, nil
}

// EinoRepoDocService 高级服务实现，支持更多配置选项
// 提供额外的工具获取等方法，便于扩展
type EinoRepoDocService struct {
	basePath   string       // 仓库存储的基础路径
	llmClient  *llm.Client  // LLM 客户端
	chatModel  model.ChatModel // Eino ChatModel 实例
	chain      *RepoDocChain    // Eino Chain 实例
}

// NewEinoRepoDocService 创建高级服务实例
// basePath: 仓库存储的基础路径
// llmClient: LLM 客户端实例
// 返回: EinoRepoDocService 实例或错误
func NewEinoRepoDocService(basePath string, llmClient *llm.Client) (*EinoRepoDocService, error) {
	klog.V(6).Infof("[NewEinoRepoDocService] 开始创建高级服务: basePath=%s", basePath)

	klog.V(6).Infof("[NewEinoRepoDocService] 创建 ChatModel")
	chatModel := NewLLMChatModel(llmClient)

	klog.V(6).Infof("[NewEinoRepoDocService] 创建 RepoDocChain")
	chain, err := NewRepoDocChain(basePath, chatModel)
	if err != nil {
		klog.Errorf("[NewEinoRepoDocService] 创建 RepoDocChain 失败: %v", err)
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	klog.V(6).Infof("[NewEinoRepoDocService] 高级服务创建成功")
	return &EinoRepoDocService{
		basePath:  basePath,
		llmClient: llmClient,
		chatModel: chatModel,
		chain:     chain,
	}, nil
}

// ParseRepo 实现 RepoDocService 接口
// ctx: 上下文
// repoURL: 仓库 Git URL
// 返回: 解析结果或错误
func (s *EinoRepoDocService) ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error) {
	klog.V(6).Infof("[EinoRepoDocService.ParseRepo] 开始解析仓库: repoURL=%s", repoURL)
	result, err := s.chain.Run(ctx, WorkflowInput{RepoURL: repoURL})
	if err != nil {
		klog.Errorf("[EinoRepoDocService.ParseRepo] 解析失败: %v", err)
		return nil, err
	}
	klog.V(6).Infof("[EinoRepoDocService.ParseRepo] 解析成功: sections=%d", result.SectionsCount)
	return result, nil
}

// GetChatModel 获取 ChatModel（用于扩展）
// 返回: Eino ChatModel 实例
func (s *EinoRepoDocService) GetChatModel() model.ChatModel {
	klog.V(6).Infof("[EinoRepoDocService.GetChatModel] 获取 ChatModel")
	return s.chatModel
}

// GetTools 获取工具列表（用于扩展）
// 返回: Eino Tools 列表
func (s *EinoRepoDocService) GetTools() []tool.BaseTool {
	klog.V(6).Infof("[EinoRepoDocService.GetTools] 获取工具列表")
	tools := tools.CreateTools(s.basePath)
	klog.V(6).Infof("[EinoRepoDocService.GetTools] 工具列表: count=%d", len(tools))
	return tools
}
