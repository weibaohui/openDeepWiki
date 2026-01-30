package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/pkg/agents"
	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"k8s.io/klog/v2"
)

// Executor Agent执行器
type Executor struct {
	cfg          *config.Config
	manager      *agents.Manager
	llmClient    *llm.Client
	toolExecutor *llm.SafeExecutor
	defaultTools []llm.Tool // 缓存默认 tools
}

// AnalysisResult 代码分析结果
type AnalysisResult struct {
	ProjectType   string         `json:"project_type"`
	Language      string         `json:"language"`
	Framework     string         `json:"framework"`
	EntryPoints   []string       `json:"entry_points"`
	KeyFiles      []FileAnalysis `json:"key_files"`
	TreeStructure string         `json:"tree_structure"`
	Summary       string         `json:"summary"`
}

// FileAnalysis 文件分析结果
type FileAnalysis struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	KeySymbols  []string `json:"key_symbols"`
}

// NewExecutor 创建Agent执行器
func NewExecutor(cfg *config.Config) *Executor {
	client := llm.NewClient(
		cfg.LLM.APIURL,
		cfg.LLM.APIKey,
		cfg.LLM.Model,
		cfg.LLM.MaxTokens,
	)

	// 创建 Agent Manager
	managerConfig := &agents.Config{
		Dir:          "../agents",
		AutoReload:   true,
		DefaultAgent: "orchestrator-agent",
		Routes: map[string]string{
			"repo-initializer": "repo-initializer",
			"architect-agent":  "architect-agent",
		},
	}

	manager, err := agents.NewManager(managerConfig)
	if err != nil {
		klog.Errorf("Failed to create agent manager: %v", err)
	}

	return &Executor{
		cfg:          cfg,
		manager:      manager,
		llmClient:    client,
		toolExecutor: llm.NewSafeExecutor(".", llm.DefaultExecutorConfig()),
		defaultTools: llm.DefaultTools(), // 缓存默认 tools
	}
}

// Execute 执行AI分析流程
func (e *Executor) Execute(ctx context.Context, repoPath string, outputPath string) error {
	klog.V(6).Infof("开始AI分析: repoPath=%s, outputPath=%s", repoPath, outputPath)

	// 执行对话
	result, err := e.ExecuteConversation(ctx, "orchestrator-agent", fmt.Sprintf("分析仓库 %s", repoPath), &ConversationOptions{})
	if err != nil {
		klog.Errorf("对话执行失败: %v", err)
		return fmt.Errorf("对话执行失败: %w", err)
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		klog.Errorf("JSON序列化失败: %v", err)
		return fmt.Errorf("JSON序列化失败: %w", err)
	}
	klog.V(6).Infof("AI分析结果: %s", string(jsonData))
	return nil
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	Type      string
	Language  string
	Framework string
}
