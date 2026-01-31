package adk

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
)

// ExampleNewADKRepoDocService 演示如何创建 ADK 服务
func ExampleNewADKRepoDocService() {
	// 配置 LLM
	llmCfg := &LLMConfig{
		APIKey:    os.Getenv("OPENAI_API_KEY"),
		BaseURL:   os.Getenv("OPENAI_BASE_URL"),
		Model:     "gpt-4o",
		MaxTokens: 4000,
	}

	// 创建服务
	service, err := NewADKRepoDocService("/tmp/repos", llmCfg)
	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
	}

	fmt.Printf("ADK 服务创建成功，使用模型: %s\n", llmCfg.Model)

	// Output: ADK 服务创建成功，使用模型: gpt-4o
	_ = service
}

// ExampleADKRepoDocService_ParseRepo 演示如何使用 ADK 服务解析仓库
func ExampleADKRepoDocService_ParseRepo() {
	// 配置 LLM
	llmCfg := &LLMConfig{
		APIKey:    os.Getenv("OPENAI_API_KEY"),
		BaseURL:   os.Getenv("OPENAI_BASE_URL"),
		Model:     "gpt-4o",
		MaxTokens: 4000,
	}

	// 创建服务
	service, err := NewADKRepoDocService("/tmp/repos", llmCfg)
	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
	}

	// 解析仓库
	ctx := context.Background()
	repoURL := "https://github.com/example/project.git"

	fmt.Printf("开始解析仓库: %s\n", repoURL)

	// 注意：实际执行需要有效的 API Key 和网络连接
	// result, err := service.ParseRepo(ctx, repoURL)
	// if err != nil {
	//     log.Fatalf("解析失败: %v", err)
	// }
	// fmt.Printf("文档生成成功，共 %d 个小节\n", result.SectionsCount)

	_ = ctx
	_ = service
}

// ExampleADKRepoDocService_ParseRepoWithProgress 演示带进度反馈的解析
func ExampleADKRepoDocService_ParseRepoWithProgress() {
	// 配置 LLM
	llmCfg := &LLMConfig{
		APIKey:    os.Getenv("OPENAI_API_KEY"),
		BaseURL:   os.Getenv("OPENAI_BASE_URL"),
		Model:     "gpt-4o",
		MaxTokens: 4000,
	}

	// 创建服务
	service, err := NewADKRepoDocService("/tmp/repos", llmCfg)
	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
	}

	// 解析仓库并获取进度
	ctx := context.Background()
	repoURL := "https://github.com/example/project.git"

	progressCh, err := service.ParseRepoWithProgress(ctx, repoURL)
	if err != nil {
		log.Fatalf("启动解析失败: %v", err)
	}

	// 处理进度事件
	for event := range progressCh {
		if event.Error != nil {
			log.Printf("步骤 %d [%s] 出错: %v\n", event.Step, event.AgentName, event.Error)
			continue
		}

		switch event.Status {
		case WorkflowStatusCompleted:
			log.Printf("步骤 %d [%s] 完成\n", event.Step, event.AgentName)
		case WorkflowStatusFinished:
			log.Printf("全部完成！共生成 %d 个小节\n", event.Result.SectionsCount)
		case WorkflowStatusError:
			log.Printf("步骤 %d [%s] 出错\n", event.Step, event.AgentName)
		}
	}

	_ = ctx
	_ = service
}

// TestSequentialAgentCreation 测试 SequentialAgent 的创建
func TestSequentialAgentCreation(t *testing.T) {
	// 注意：这个测试需要有效的 ChatModel
	// 这里仅测试配置结构的正确性

	config := &adk.SequentialAgentConfig{
		Name:        "TestSequentialAgent",
		Description: "测试顺序 Agent",
		SubAgents:   []adk.Agent{}, // 空的 Agent 列表
	}

	if config.Name != "TestSequentialAgent" {
		t.Errorf("期望名称 TestSequentialAgent，实际 %s", config.Name)
	}

	if len(config.SubAgents) != 0 {
		t.Errorf("期望 SubAgents 长度为 0，实际 %d", len(config.SubAgents))
	}
}

// TestWorkflowProgressEvent 测试进度事件
func TestWorkflowProgressEvent(t *testing.T) {
	event := &WorkflowProgressEvent{
		Step:      1,
		AgentName: "TestAgent",
		Status:    WorkflowStatusCompleted,
		Content:   "测试内容",
	}

	if event.Step != 1 {
		t.Errorf("期望 Step=1，实际 %d", event.Step)
	}

	if event.AgentName != "TestAgent" {
		t.Errorf("期望 AgentName=TestAgent，实际 %s", event.AgentName)
	}

	if event.Status != WorkflowStatusCompleted {
		t.Errorf("期望 Status=completed，实际 %s", event.Status)
	}
}

// TestStateManager 测试状态管理器
func TestStateManager(t *testing.T) {
	sm := NewStateManager("https://github.com/test/repo.git", "/tmp/test")

	// 测试设置和获取
	sm.SetRepoTree("- src\n  - main.go\n  - utils.go")
	if sm.GetRepoTree() == "" {
		t.Error("RepoTree 不应为空")
	}

	sm.SetRepoInfo("go", []string{"Go", "Gin"})
	repoType, techStack := sm.GetRepoInfo()
	if repoType != "go" {
		t.Errorf("期望 repoType=go，实际 %s", repoType)
	}
	if len(techStack) != 2 {
		t.Errorf("期望 techStack 长度为 2，实际 %d", len(techStack))
	}
}

// TestAgentRoles 测试 Agent 角色配置
func TestAgentRoles(t *testing.T) {
	// 验证所有预定义角色
	expectedRoles := []string{
		AgentRepoInitializer,
		AgentArchitect,
		AgentExplorer,
		AgentWriter,
		AgentEditor,
	}

	for _, roleName := range expectedRoles {
		role, ok := AgentRoles[roleName]
		if !ok {
			t.Errorf("缺少角色定义: %s", roleName)
			continue
		}

		if role.Name == "" {
			t.Errorf("角色 %s 的名称为空", roleName)
		}

		if role.Description == "" {
			t.Errorf("角色 %s 的描述为空", roleName)
		}

		if role.Instruction == "" {
			t.Errorf("角色 %s 的指令为空", roleName)
		}
	}
}

// TestWorkflowOutputStructure 测试 Workflow 输出结构
func TestWorkflowOutputStructure(t *testing.T) {
	state := einodoc.NewRepoDocState("https://github.com/test/repo.git", "/tmp/test")
	state.SetRepoInfo("go", []string{"Go", "Gin"})
	state.SetOutline([]einodoc.Chapter{
		{
			Title: "项目概述",
			Sections: []einodoc.Section{
				{Title: "简介", Hints: []string{"项目背景"}},
			},
		},
	})

	// 测试状态管理器构建结果
	sm := &StateManager{state: state}
	result := sm.BuildResult()

	if result.RepoURL != "https://github.com/test/repo.git" {
		t.Errorf("期望 RepoURL 匹配，实际 %s", result.RepoURL)
	}

	if result.RepoType != "go" {
		t.Errorf("期望 RepoType=go，实际 %s", result.RepoType)
	}

	if len(result.Outline) != 1 {
		t.Errorf("期望 Outline 长度为 1，实际 %d", len(result.Outline))
	}
}

// TestBuildWorkflowInput 测试构建 Workflow 输入
func TestBuildWorkflowInput(t *testing.T) {
	repoURL := "https://github.com/test/repo.git"
	input := BuildWorkflowInput(repoURL)

	if input == nil {
		t.Fatal("input 不应为 nil")
	}

	if len(input.Messages) != 1 {
		t.Errorf("期望 Messages 长度为 1，实际 %d", len(input.Messages))
	}

	if input.Messages[0].Role != schema.User {
		t.Errorf("期望 Role=User，实际 %s", input.Messages[0].Role)
	}
}

// TestExtractJSON 测试 JSON 提取函数
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `{"repo_type": "go", "tech_stack": ["Go"]}}`,
			expected: `{"repo_type": "go", "tech_stack": ["Go"]}`,
		},
		{
			input:    `Some text {"key": "value"} more text`,
			expected: `{"key": "value"}`,
		},
		{
			input:    `{"nested": {"key": "value"}}`,
			expected: `{"nested": {"key": "value"}}`,
		},
	}

	for _, tt := range tests {
		result := extractJSON(tt.input)
		if result != tt.expected {
			t.Errorf("extractJSON(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestTruncate 测试截断函数
func TestTruncate(t *testing.T) {
	tests := []struct {
		s       string
		maxLen  int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"test", 4, "test"},
	}

	for _, tt := range tests {
		result := truncate(tt.s, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.s, tt.maxLen, result, tt.expected)
		}
	}
}
