package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/pkg/agents"
	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/opendeepwiki/backend/internal/pkg/llm/tools"
	"k8s.io/klog/v2"
)

// Executor Agent执行器
type Executor struct {
	cfg       *config.Config
	manager   *agents.Manager
	llmClient *llm.Client
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
		DefaultAgent: "architect-agent",
		Routes: map[string]string{
			"repo-analysis": "repo-analysis-agent",
			"code-diagnose": "code-diagnosis-agent",
			"docs-gen":      "documentation-generator-agent",
		},
	}

	manager, err := agents.NewManager(managerConfig)
	if err != nil {
		klog.Errorf("Failed to create agent manager: %v", err)
		// 继续运行，使用降级功能
	}

	return &Executor{
		cfg:       cfg,
		manager:   manager,
		llmClient: client,
	}
}

// Execute 执行AI分析流程
func (e *Executor) Execute(ctx context.Context, repoPath string, outputPath string) error {
	klog.V(6).Infof("开始AI分析: repoPath=%s, outputPath=%s", repoPath, outputPath)

	// 使用 Agent 执行仓库解读任务
	analysis, err := e.executeRepoAnalysisWithAgent(ctx, repoPath)
	if err != nil {
		klog.Errorf("代码分析失败: %v", err)
		return fmt.Errorf("代码分析失败: %w", err)
	}

	klog.V(6).Infof("代码分析完成: projectType=%s, language=%s", analysis.ProjectType, analysis.Language)

	// 2. 文档生成阶段 - 类似 WriterAgent
	docContent, err := e.generateDoc(ctx, analysis)
	if err != nil {
		klog.Errorf("文档生成失败: %v", err)
		return fmt.Errorf("文档生成失败: %w", err)
	}
	klog.V(6).Infof("文档生成完成: contentLength=%d", len(docContent))

	// 3. 写入文件
	if err := e.writeToFile(outputPath, docContent); err != nil {
		klog.Errorf("写入文档失败: %v", err)
		return fmt.Errorf("写入文档失败: %w", err)
	}
	klog.V(6).Infof("AI分析完成，文档已保存: %s", outputPath)

	return nil
}

// executeRepoAnalysisWithAgent 使用Agent执行仓库分析任务
func (e *Executor) executeRepoAnalysisWithAgent(ctx context.Context, repoPath string) (*AnalysisResult, error) {
	if e.manager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	// 准备路由上下文，选择仓库分析专用Agent
	routerCtx := agents.RouterContext{
		EntryPoint: "repo-analysis", // 用户访问仓库分析
		TaskType:   "repository-analysis",
		Metadata: map[string]string{
			"repo_path": repoPath,
		},
	}

	// 选择合适的Agent
	agent, err := e.manager.SelectAgent(routerCtx)
	if err != nil {
		// 尝试使用默认代理
		routerCtx = agents.RouterContext{}
		agent, err = e.manager.SelectAgent(routerCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to select agent: %w", err)
		}
	}
	//TODO Agent 如何发起llm对话？

	klog.V(6).Infof("Selected agent: %s", agent.Name)
	klog.V(6).Infof("System Prompt: %s", agent.SystemPrompt)
	klog.V(6).Infof("Allowed Skills: %v", agent.SkillPolicy.Allow)
	klog.V(6).Infof("Risk Level: %s", agent.RuntimePolicy.RiskLevel)

	// 现在使用这个 Agent 执行仓库解读任务
	// 这里可以使用 Agent 的信息构建特定的分析逻辑
	result := &AnalysisResult{
		EntryPoints: []string{},
		KeyFiles:    []FileAnalysis{},
	}

	// 1. 获取目录结构
	treeResult, err := tools.ListDir(json.RawMessage(`{"dir":".","recursive":true}`), repoPath)
	if err != nil {
		klog.Warningf("获取目录结构失败: %v", err)
	} else {
		result.TreeStructure = treeResult
	}

	// 2. 检测项目类型和技术栈
	projectInfo, err := e.detectProjectType(repoPath)
	if err != nil {
		klog.Warningf("检测项目类型失败: %v", err)
	} else {
		result.ProjectType = projectInfo.Type
		result.Language = projectInfo.Language
		result.Framework = projectInfo.Framework
	}

	// 3. 使用LLM分析项目结构（可能集成Agent的能力）
	analysis, err := e.analyzeWithLLM(ctx, repoPath, result.TreeStructure)
	if err != nil {
		klog.Warningf("LLM分析失败: %v", err)
	} else {
		result.Summary = analysis.Summary
		result.KeyFiles = analysis.KeyFiles
		result.EntryPoints = analysis.EntryPoints
	}

	return result, nil
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	Type      string
	Language  string
	Framework string
}

// detectProjectType 检测项目类型
func (e *Executor) detectProjectType(repoPath string) (*ProjectInfo, error) {
	info := &ProjectInfo{
		Type:      "unknown",
		Language:  "unknown",
		Framework: "unknown",
	}

	// 检查各种配置文件
	files, err := os.ReadDir(repoPath)
	if err != nil {
		return info, err
	}

	for _, f := range files {
		name := f.Name()
		switch name {
		case "go.mod":
			info.Language = "Go"
			info.Type = "go-module"
			// 读取go.mod获取框架信息
			content, _ := os.ReadFile(filepath.Join(repoPath, name))
			contentStr := string(content)
			if strings.Contains(contentStr, "gin") {
				info.Framework = "Gin"
			} else if strings.Contains(contentStr, "echo") {
				info.Framework = "Echo"
			} else if strings.Contains(contentStr, "fiber") {
				info.Framework = "Fiber"
			}
		case "package.json":
			info.Language = "JavaScript/TypeScript"
			info.Type = "node-project"
			content, _ := os.ReadFile(filepath.Join(repoPath, name))
			contentStr := string(content)
			if strings.Contains(contentStr, "react") {
				info.Framework = "React"
			} else if strings.Contains(contentStr, "vue") {
				info.Framework = "Vue"
			} else if strings.Contains(contentStr, "express") {
				info.Framework = "Express"
			}
		case "requirements.txt", "setup.py":
			info.Language = "Python"
			info.Type = "python-project"
		case "pom.xml", "build.gradle":
			info.Language = "Java"
			info.Type = "java-project"
		case "Cargo.toml":
			info.Language = "Rust"
			info.Type = "rust-project"
		}
	}

	return info, nil
}

// LLMAnalysis LLM分析结果
type LLMAnalysis struct {
	Summary     string         `json:"summary"`
	KeyFiles    []FileAnalysis `json:"key_files"`
	EntryPoints []string       `json:"entry_points"`
}

// analyzeWithLLM 使用LLM分析项目
func (e *Executor) analyzeWithLLM(ctx context.Context, repoPath string, treeStructure string) (*LLMAnalysis, error) {
	// 构建提示词
	prompt := fmt.Sprintf(`你是一个代码分析专家。请分析以下项目结构，识别关键文件和入口点。

项目目录结构:
%s

请用JSON格式输出分析结果:
{
  "summary": "项目整体描述(100字以内)",
  "entry_points": ["入口文件1", "入口文件2"],
  "key_files": [
    {
      "path": "文件路径",
      "type": "类型(main/config/handler/model等)",
      "description": "文件功能描述",
      "key_symbols": ["关键函数/类名"]
    }
  ]
}

注意:
1. 只输出JSON，不要其他内容
2. entry_points最多3个
3. key_files最多8个`, treeStructure)

	// 调用LLM
	messages := []llm.ChatMessage{
		{Role: "system", Content: "你是一个代码分析专家，专门分析项目结构并识别关键文件。"},
		{Role: "user", Content: prompt},
	}

	content, err := e.llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 解析JSON响应
	// 提取JSON部分
	if idx := strings.Index(content, "{"); idx != -1 {
		content = content[idx:]
	}
	if idx := strings.LastIndex(content, "}"); idx != -1 {
		content = content[:idx+1]
	}

	var analysis LLMAnalysis
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		klog.Warningf("解析LLM响应失败: %v, content=%s", err, content)
		// 返回简化结果
		return &LLMAnalysis{
			Summary:     "项目分析完成",
			KeyFiles:    []FileAnalysis{},
			EntryPoints: []string{},
		}, nil
	}

	return &analysis, nil
}

// generateDoc 生成文档 - 实现WriterAgent功能
func (e *Executor) generateDoc(ctx context.Context, analysis *AnalysisResult) (string, error) {
	// 构建文档内容
	var doc strings.Builder

	// 文档头
	doc.WriteString("# AI 代码分析报告\n\n")
	doc.WriteString(fmt.Sprintf("> 生成时间: %s\n> 分析工具: openDeepWiki AI Agent\n\n",
		time.Now().Format("2006-01-30 15:04:05")))

	// 1. 项目概述
	doc.WriteString("## 1. 项目概述\n\n")
	doc.WriteString(fmt.Sprintf("**项目类型**: %s\n\n", analysis.ProjectType))
	doc.WriteString(fmt.Sprintf("**主要语言**: %s\n\n", analysis.Language))
	if analysis.Framework != "unknown" {
		doc.WriteString(fmt.Sprintf("**框架**: %s\n\n", analysis.Framework))
	}
	doc.WriteString(fmt.Sprintf("**项目简介**: %s\n\n", analysis.Summary))

	// 2. 入口点分析
	if len(analysis.EntryPoints) > 0 {
		doc.WriteString("## 2. 入口点\n\n")
		for _, ep := range analysis.EntryPoints {
			doc.WriteString(fmt.Sprintf("- `%s`\n", ep))
		}
		doc.WriteString("\n")
	}

	// 3. 关键文件分析
	if len(analysis.KeyFiles) > 0 {
		doc.WriteString("## 3. 关键文件分析\n\n")
		for _, file := range analysis.KeyFiles {
			doc.WriteString(fmt.Sprintf("### 3.%d %s\n\n", getFileIndex(&file), file.Path))
			doc.WriteString(fmt.Sprintf("**类型**: %s\n\n", file.Type))
			doc.WriteString(fmt.Sprintf("**功能**: %s\n\n", file.Description))
			if len(file.KeySymbols) > 0 {
				doc.WriteString("**关键符号**: ")
				doc.WriteString(strings.Join(file.KeySymbols, ", "))
				doc.WriteString("\n\n")
			}
		}
	}

	// 4. 目录结构
	if analysis.TreeStructure != "" {
		doc.WriteString("## 4. 目录结构\n\n")
		doc.WriteString("```\n")
		doc.WriteString(analysis.TreeStructure)
		doc.WriteString("\n```\n\n")
	}

	// 5. 总结
	doc.WriteString("## 5. 总结\n\n")
	doc.WriteString("本项目是一个使用AI Agent自动生成的代码分析报告。\n")
	doc.WriteString("报告基于静态代码分析和LLM理解生成，涵盖了项目的主要结构和关键文件。\n\n")
	doc.WriteString("**分析方式**: ExplorerAgent + WriterAgent 协作\n")
	doc.WriteString(fmt.Sprintf("**分析时间**: %s\n", time.Now().Format("2006-01-30 15:04:05")))

	return doc.String(), nil
}

// getFileIndex 获取文件索引（用于序号）
var fileIndex = 0

func getFileIndex(file *FileAnalysis) int {
	fileIndex++
	return fileIndex
}

// writeToFile 写入文件
func (e *Executor) writeToFile(outputPath string, content string) error {
	// 确保目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// ResetFileIndex 重置文件索引（测试用）
func ResetFileIndex() {
	fileIndex = 0
}
