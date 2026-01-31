package einodoc

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
)

// LLMConfig LLM 配置
type LLMConfig struct {
	APIKey    string // API Key
	BaseURL   string // API 基础 URL
	Model     string // 模型名称
	MaxTokens int    // 最大生成 token 数
}

// RepoDocService 仓库文档解析服务接口
// 对外提供统一的仓库文档生成能力
type RepoDocService interface {
	// ParseRepo 解析仓库，生成文档
	// ctx: 上下文，可用于超时控制
	// repoURL: 仓库 Git URL
	// 返回: 解析结果或错误
	ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error)
}

// EinoRepoDocService 高级服务实现，支持更多配置选项
// 提供额外的工具获取等方法，便于扩展
type EinoRepoDocService struct {
	basePath  string                     // 仓库存储的基础路径
	llmCfg    *LLMConfig                 // LLM 配置
	chatModel model.ToolCallingChatModel // Eino ChatModel 实例
	chain     *RepoDocChain              // Eino Chain 实例
	callbacks *EinoCallbacks             // Eino 回调处理器
}

// NewEinoRepoDocService 创建高级服务实例
// basePath: 仓库存储的基础路径
// llmCfg: LLM 配置
// 返回: EinoRepoDocService 实例或错误
func NewEinoRepoDocService(basePath string, llmCfg *LLMConfig) (*EinoRepoDocService, error) {
	callbacks := NewEinoCallbacks(true, klog.Level(8))
	return NewEinoRepoDocServiceWithCallbacks(basePath, llmCfg, callbacks)
}

// NewEinoRepoDocServiceWithCallbacks 创建带回调的高级服务实例
// basePath: 仓库存储的基础路径
// llmCfg: LLM 配置
// callbacks: Eino 回调处理器，用于观察执行过程（可为 nil）
// 返回: EinoRepoDocService 实例或错误
func NewEinoRepoDocServiceWithCallbacks(basePath string, llmCfg *LLMConfig, callbacks *EinoCallbacks) (*EinoRepoDocService, error) {
	klog.V(6).Infof("[NewEinoRepoDocService] 开始创建高级服务: basePath=%s, model=%s", basePath, llmCfg.Model)

	klog.V(6).Infof("[NewEinoRepoDocService] 创建 ChatModel")
	chatModel, err := NewLLMChatModel(llmCfg.APIKey, llmCfg.BaseURL, llmCfg.Model, llmCfg.MaxTokens)
	if err != nil {
		klog.Errorf("[NewEinoRepoDocService] 创建 ChatModel 失败: %v", err)
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	klog.V(6).Infof("[NewEinoRepoDocService] 创建 RepoDocChain")
	chain, err := NewRepoDocChainWithCallbacks(basePath, chatModel, callbacks)
	if err != nil {
		klog.Errorf("[NewEinoRepoDocService] 创建 RepoDocChain 失败: %v", err)
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	klog.V(6).Infof("[NewEinoRepoDocService] 高级服务创建成功")
	return &EinoRepoDocService{
		basePath:  basePath,
		llmCfg:    llmCfg,
		chatModel: chatModel,
		chain:     chain,
		callbacks: callbacks,
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
func (s *EinoRepoDocService) GetChatModel() model.ToolCallingChatModel {
	klog.V(6).Infof("[EinoRepoDocService.GetChatModel] 获取 ChatModel")
	return s.chatModel
}

// GetTools 获取工具列表（用于扩展）
// 返回: Eino Tools 列表
func (s *EinoRepoDocService) GetTools() []tool.BaseTool {
	klog.V(6).Infof("[EinoRepoDocService.GetTools] 获取工具列表")
	ts := tools.CreateTools(s.basePath)
	klog.V(6).Infof("[EinoRepoDocService.GetTools] 工具列表: count=%d", len(ts))
	return ts
}

// GetCallbacks 获取回调处理器（用于扩展）
// 返回: EinoCallbacks 实例或 nil
func (s *EinoRepoDocService) GetCallbacks() *EinoCallbacks {
	klog.V(6).Infof("[EinoRepoDocService.GetCallbacks] 获取回调处理器")
	return s.callbacks
}

// SetCallbacks 设置回调处理器（用于动态启用/禁用）
// callbacks: EinoCallbacks 实例（可为 nil）
func (s *EinoRepoDocService) SetCallbacks(callbacks *EinoCallbacks) {
	klog.V(6).Infof("[EinoRepoDocService.SetCallbacks] 设置回调处理器: enabled=%v", callbacks != nil && callbacks.IsEnabled())
	s.callbacks = callbacks
	if s.chain != nil {
		s.chain.SetCallbacks(callbacks)
	}
}
