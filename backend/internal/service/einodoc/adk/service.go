package adk

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"k8s.io/klog/v2"
)

// LLMConfig LLM 配置
type LLMConfig struct {
	APIKey    string // API Key
	BaseURL   string // API 基础 URL
	Model     string // 模型名称
	MaxTokens int    // 最大生成 token 数
}

// ADKRepoDocService ADK 模式的仓库文档解析服务
// 使用 Eino ADK 原生的 SequentialAgent 和 Runner
type ADKRepoDocService struct {
	basePath  string                     // 仓库存储的基础路径
	llmCfg    *LLMConfig                 // LLM 配置
	chatModel model.ToolCallingChatModel // Eino ChatModel 实例
	workflow  *RepoDocWorkflow           // Workflow 实例
}

// NewADKRepoDocService 创建 ADK 服务实例
// basePath: 仓库存储的基础路径
// llmCfg: LLM 配置
// 返回: ADKRepoDocService 实例或错误
func NewADKRepoDocService(basePath string, llmCfg *LLMConfig) (*ADKRepoDocService, error) {
	klog.V(6).Infof("[NewADKRepoDocService] 开始创建 ADK 服务: basePath=%s, model=%s", basePath, llmCfg.Model)

	// 创建 ChatModel
	chatModel, err := einodoc.NewLLMChatModel(llmCfg.APIKey, llmCfg.BaseURL, llmCfg.Model, llmCfg.MaxTokens)
	if err != nil {
		klog.Errorf("[NewADKRepoDocService] 创建 ChatModel 失败: %v", err)
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	// 创建 Workflow
	workflow, err := NewRepoDocWorkflow(basePath, chatModel)
	if err != nil {
		klog.Errorf("[NewADKRepoDocService] 创建 Workflow 失败: %v", err)
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	klog.V(6).Infof("[NewADKRepoDocService] ADK 服务创建成功")

	return &ADKRepoDocService{
		basePath:  basePath,
		llmCfg:    llmCfg,
		chatModel: chatModel,
		workflow:  workflow,
	}, nil
}

// ParseRepo 解析仓库，生成文档
// ctx: 上下文，可用于超时控制
// repoURL: 仓库 Git URL
// 返回: 解析结果或错误
func (s *ADKRepoDocService) ParseRepo(ctx context.Context, localPath string) (*einodoc.RepoDocResult, error) {
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

// GetWorkflowInfo 获取 Workflow 信息
// 返回 Workflow 的结构信息，用于调试和展示
func (s *ADKRepoDocService) GetWorkflowInfo() *WorkflowInfo {
	return s.workflow.GetWorkflowInfo()
}

// GetChatModel 获取 ChatModel（用于扩展）
// 返回: Eino ChatModel 实例
func (s *ADKRepoDocService) GetChatModel() model.ToolCallingChatModel {
	klog.V(6).Infof("[ADKRepoDocService.GetChatModel] 获取 ChatModel")
	return s.chatModel
}

// ==================== 便捷构造函数 ====================

// NewADKServiceFromConfig 从配置创建 ADK 服务
func NewADKServiceFromConfig(basePath string, apiKey, baseURL, modelName string, maxTokens int) (*ADKRepoDocService, error) {
	llmCfg := &LLMConfig{
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Model:     modelName,
		MaxTokens: maxTokens,
	}
	return NewADKRepoDocService(basePath, llmCfg)
}

// ==================== 服务状态 ====================

// ServiceStatus 服务状态
type ServiceStatus struct {
	Ready     bool   `json:"ready"`
	BasePath  string `json:"base_path"`
	ModelName string `json:"model_name"`
}

// GetStatus 获取服务状态
func (s *ADKRepoDocService) GetStatus() *ServiceStatus {
	return &ServiceStatus{
		Ready:     s.chatModel != nil && s.workflow != nil,
		BasePath:  s.basePath,
		ModelName: s.llmCfg.Model,
	}
}

// ToJSON 将服务状态转换为 JSON
func (s *ServiceStatus) ToJSON() string {
	return ToJSON(s)
}
