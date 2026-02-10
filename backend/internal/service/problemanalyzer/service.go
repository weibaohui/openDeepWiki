package problemanalyzer

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

const (
	agentSolver  = "problem_solver"
	agentMdCheck = "markdown_checker"
)

var (
	ErrInvalidLocalPath     = errors.New("invalid local path")
	ErrAgentExecutionFailed = errors.New("agent execution failed")
)

type Service struct {
	factory  *adkagents.AgentFactory
	hintRepo repository.HintRepository
}

// New 创建问题分析服务实例。
func New(cfg *config.Config, hintRepo repository.HintRepository) (*Service, error) {
	klog.V(6).Infof("[problemanalyzer.New] 创建问题分析服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[problemanalyzer.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &Service{
		factory:  factory,
		hintRepo: hintRepo,
	}, nil
}

// Generate 生成问题分析报告。
func (s *Service) Generate(ctx context.Context, localPath string, problem string, repoID uint, taskID uint) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: local path is empty", ErrInvalidLocalPath)
	}
	if problem == "" {
		return "", fmt.Errorf("%w: problem statement is empty", ErrInvalidLocalPath)
	}

	title := "问题分析报告" // 默认标题，或者从problem截取

	klog.V(6).Infof("[problemanalyzer.Generate] 开始分析问题: 仓库路径=%s, 问题=%s, 任务ID=%d", localPath, problem, taskID)
	markdown, err := s.genDocument(ctx, localPath, title, problem, repoID, taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[problemanalyzer.Generate] 分析完成: 内容长度=%d", len(markdown))
	return markdown, nil
}

// genDocument 负责调用Agent并返回最终文档内容。
func (s *Service) genDocument(ctx context.Context, localPath string, title string, problem string, repoID uint, taskID uint) (string, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)
	adk.AddSessionValue(ctx, "problem_statement", problem)
	adk.AddSessionValue(ctx, "task_id", taskID)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"problem_solver_sequential_agent",
		"problem solver sequential agent - analyze codebase to answer specific problem",
		agentSolver,
		agentMdCheck,
	)
	if err != nil {
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并回答以下问题。

仓库地址: %s
问题描述: %s
`, localPath, problem)

	klog.V(6).Infof("[problemanalyzer.genDocument] 调用Agent: %s", initialMessage)

	lastContent, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})
	if err != nil {
		klog.Errorf("[problemanalyzer.genDocument] Agent执行失败: %v", err)
		return "", fmt.Errorf("agent execution error: %w", err)
	}
	if lastContent == "" {
		return "", fmt.Errorf("no agent output")
	}

	return lastContent, nil
}
