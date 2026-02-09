package dirmaker

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// Agent 名称常量
const (
	agentTocEditor  = "toc_editor"  // 目录制定者
	agentTocChecker = "toc_checker" // 目录校验 Agent
)

// 错误定义
var (
	ErrInvalidLocalPath     = errors.New("无效的本地路径")
	ErrAgentExecutionFailed = errors.New("Agent 执行失败")
	ErrYAMLParseFailed      = errors.New("YAML 解析失败")
	ErrTaskCreationFailed   = errors.New("任务创建失败")
	ErrNoAgentOutput        = errors.New("Agent 未产生任何输出内容")
)

// Service 目录分析任务生成服务。
// 基于 Eino ADK 实现，用于分析代码目录并动态生成分析任务。
type Service struct {
	factory *adkagents.AgentFactory
	docRepo repository.DocumentRepository
}

// New 创建目录分析服务实例。
func New(cfg *config.Config, docRepo repository.DocumentRepository) (*Service, error) {
	klog.V(6).Infof("[dirmaker.New] 开始创建目录分析服务")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[dirmaker.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("创建 AgentFactory 失败: %w", err)
	}

	return &Service{
		factory: factory,
		docRepo: docRepo,
	}, nil
}

// CreateDirs 分析仓库目录并创建目录。
func (s *Service) CreateDirs(ctx context.Context, repo *model.Repository) (*model.DirMakerGenerationResult, error) {
	if repo == nil {
		return nil, fmt.Errorf("%w: repo 为空", ErrInvalidLocalPath)
	}
	if repo.LocalPath == "" {
		return nil, ErrInvalidLocalPath
	}
	if _, err := os.Stat(repo.LocalPath); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLocalPath, err)
	}

	result, err := s.genDirList(ctx, repo.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[dirmaker.CreateTasks] 分析完成，生成目录数: %d", len(result.Dirs))
	klog.V(6).Infof("[dirmaker.CreateTasks] 分析摘要: %s", result.AnalysisSummary)
	return result, nil
}

// generateTaskPlan 执行任务生成链路，返回解析后的任务列表结果。
func (s *Service) genDirList(ctx context.Context, localPath string) (*model.DirMakerGenerationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"toc_generator_sequential_agent",
		"目录制定者顺序执行 Agent - 先生成目录，再校验修正",
		agentTocEditor,
		agentTocChecker,
	)

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成需要的技术分析任务列表。

仓库地址: %s

请按以下步骤执行：
1. 分析仓库目录结构，识别项目类型和技术栈
2. 根据项目特征生成初步的任务列表
3. 校验并修正任务列表，确保完整性和合理性

请确保最终输出为严格符合 YAML 规范的目录结构（包含 dirs 与 analysis_summary），无多余注释或解释性文字。`, localPath)
	lastContent, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Agent 执行出错: %w", err)
	}

	if lastContent == "" {
		return nil, ErrNoAgentOutput
	}
	klog.V(6).Infof("[dirmaker.generateDirList] 最后 Agent 输出 原文: \n%s\n", lastContent)

	result, err := parseDirList(lastContent)
	if err != nil {
		klog.Errorf("[dirmaker.generateDirList] 解析目录生成结果失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[dirmaker.generateDirList] 执行成功，生成目录数: %d", len(result.Dirs))
	return result, nil
}

// parseDirList 从 Agent 输出解析目录生成结果。
func parseDirList(content string) (*model.DirMakerGenerationResult, error) {
	klog.V(6).Infof("[dm.parseList] 开始解析 Agent 输出内容，长度: %d", len(content))

	// 尝试从内容中提取 YAML
	yamlStr := utils.ExtractYAML(content)
	if yamlStr == "" {
		klog.Warningf("[dm.parseList] 提取 YAML 失败")
		return nil, fmt.Errorf("%w: 提取 YAML 失败", ErrYAMLParseFailed)
	}

	var result model.DirMakerGenerationResult
	if err := yaml.Unmarshal([]byte(yamlStr), &result); err != nil {
		klog.Errorf("[dm.parseList] YAML 解析失败: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrYAMLParseFailed, err)
	}
	//TODO 入库，后续分析作为背景。
	klog.V(6).Infof("AI分析概要\n%s\n", result.AnalysisSummary)

	klog.V(6).Infof("[dm.parseList] 解析完成，目录数: %d", len(result.Dirs))
	return &result, nil
}
