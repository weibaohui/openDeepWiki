package writers

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

type userRequestWriter struct {
	factory  *adkagents.AgentFactory
	hintRepo repository.HintRepository
}

// New 创建问题分析服务实例。
func NewUserRequestWriter(cfg *config.Config, hintRepo repository.HintRepository) (*userRequestWriter, error) {
	klog.V(6).Infof("[writers.NewUserRequestWriter] 创建用户请求分析服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[writers.NewUserRequestWriter] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &userRequestWriter{
		factory:  factory,
		hintRepo: hintRepo,
	}, nil
}

func (s *userRequestWriter) Name() domain.WriterName {
	return domain.UserRequestWriter
}

// Generate 生成问题分析报告。
func (s *userRequestWriter) Generate(ctx context.Context, localPath string, userRequest string, taskID uint) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: local path is empty", domain.ErrInvalidLocalPath)
	}
	if userRequest == "" {
		return "", fmt.Errorf("%w: user request is empty", domain.ErrInvalidLocalPath)
	}

	klog.V(6).Infof("[%s] 开始分析问题: 仓库路径=%s, 用户请求=%s, 任务ID=%d", s.Name(), localPath, userRequest, taskID)
	markdown, err := s.genDocument(ctx, localPath, userRequest, taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[%s] 分析完成: 内容长度=%d", s.Name(), len(markdown))
	return markdown, nil
}

// genDocument 负责调用Agent并返回最终文档内容。
func (s *userRequestWriter) genDocument(ctx context.Context, localPath string, userRequest string, taskID uint) (string, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "task_id", taskID)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"problem_solver_sequential_agent",
		"problem solver sequential agent - analyze codebase to answer specific problem",
		domain.AgentProblemSolver,
		domain.AgentMdCheck,
	)
	if err != nil {
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并回答以下问题。

仓库地址: %s
问题描述: %s
`, localPath, userRequest)

	klog.V(6).Infof("[%s] 调用Agent: %s", s.Name(), initialMessage)

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
