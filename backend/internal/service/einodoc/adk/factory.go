package adk

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/opendeepwiki/backend/internal/pkg/adkagents"
	"k8s.io/klog/v2"
)

// AgentName 定义各个子 Agent 的名称常量
const (
	// AgentRepoInitializer 仓库初始化 Agent - 负责克隆仓库和基础分析
	AgentRepoInitializer = "RepoInitializer"
	// AgentArchitect 架构师 Agent - 负责生成文档大纲
	AgentArchitect = "Architect"
	// AgentExplorer 探索者 Agent - 负责深度代码分析
	AgentExplorer = "Explorer"
	// AgentWriter 作者 Agent - 负责生成文档内容
	AgentWriter = "Writer"
	// AgentEditor 编辑 Agent - 负责组装最终文档
	AgentEditor = "Editor"
)

// AgentFactory 负责创建各种子 Agent
// 使用 adkagents.Manager 管理基础 Agent 的加载和创建
type AgentFactory struct {
	manager  *adkagents.Manager
	basePath string
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(chatModel model.ToolCallingChatModel, basePath string) (*AgentFactory, error) {
	// 创建 providers
	mp := &modelProvider{chatModel: chatModel}
	tp := &toolProvider{basePath: basePath}

	// 创建 Manager
	config := &adkagents.Config{
		Dir:            "../agents",
		ReloadInterval: 5 * time.Second,
		ModelProvider:  mp,
		ToolProvider:   tp,
	}

	manager, err := adkagents.GetOrCreateInstance(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create adkagents manager: %w", err)
	}

	return &AgentFactory{
		manager:  manager,
		basePath: basePath,
	}, nil
}

// GetAgent 获取指定名称的基础 Agent
// 这是获取基础 Agent 的推荐方式
func (f *AgentFactory) GetAgent(name string) (adk.Agent, error) {
	return f.manager.GetAgent(name)
}

// CreateSequentialAgent 创建顺序执行的 SequentialAgent
// 将所有子 Agent 按顺序组合
// 注意：此方法保持既有逻辑，不由 adkagents.Manager 直接管理
func (f *AgentFactory) CreateSequentialAgent() (adk.ResumableAgent, error) {
	ctx := context.Background()

	architect, err := f.manager.GetAgent(AgentArchitect)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentArchitect, err)
	}

	explorer, err := f.manager.GetAgent(AgentExplorer)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentExplorer, err)
	}

	writer, err := f.manager.GetAgent(AgentWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentWriter, err)
	}

	editor, err := f.manager.GetAgent(AgentEditor)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentEditor, err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create skill middleware: %w", err)
	}

	// 创建 SequentialAgent
	config := &adk.SequentialAgentConfig{

		Name:        "RepoDocSequentialAgent",
		Description: "仓库文档生成顺序执行 Agent - 按顺序执行初始化、分析、探索、撰写、编辑",
		SubAgents: []adk.Agent{
			architect,
			explorer,
			writer,
			editor,
		},
	}

	sequentialAgent, err := adk.NewSequentialAgent(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sequential agent: %w", err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 SequentialAgent 成功")
	return sequentialAgent, nil
}

// Stop 停止 AgentFactory，释放资源
func (f *AgentFactory) Stop() {
	if f.manager != nil {
		f.manager.Stop()
	}
}
