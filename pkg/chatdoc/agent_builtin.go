package chatdoc

import (
	"context"
	"strings"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"k8s.io/klog/v2"
)

// DocumentationLeader Agent
// 负责文档整体流程、任务分解与协调

type DocumentationLeaderAgent struct {
	Config chatdoc.RoleConfig
}

func (a *DocumentationLeaderAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *DocumentationLeaderAgent) HandleTask(ctx context.Context, s *chatDocService, task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("DocumentationLeaderAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx = context.WithValue(ctx, constants.SystemPrompt, sysPrompt)

	reader, err := s.parent.agentChat(ctx, task.Content, "")
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务 chat 失败: %v", err)
		return chatdoc.Task{}, err

	}
	all, err := s.parent.readAndWrite(ctx, reader)
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务 readAndWrite 失败: %v", err)
		return chatdoc.Task{}, err
	}

	return chatdoc.Task{
		Role:    "CodeAnalyster",
		Type:    "analyze",
		Content: all,
		Metadata: map[string]string{
			"section": "任务分解",
		},
	}, nil
}

// CodeAnalyster Agent
// 负责代码分析与技术说明

type CodeAnalysterAgent struct {
	Config chatdoc.RoleConfig
}

func (a *CodeAnalysterAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *CodeAnalysterAgent) HandleTask(ctx context.Context, s *chatDocService, task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("CodeAnalysterAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx = context.WithValue(ctx, constants.SystemPrompt, sysPrompt)

	reader, err := s.parent.agentChat(ctx, task.Content, "")
	if err != nil {
		klog.Errorf("CodeAnalysterAgent 处理任务 chat 失败: %v", err)
		return chatdoc.Task{}, err

	}
	all, err := s.parent.readAndWrite(ctx, reader)
	if err != nil {
		klog.Errorf("CodeAnalysterAgent 处理任务 readAndWrite 失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "TechnicalWriter",
		Type:    "write",
		Content: all,
		Metadata: map[string]string{
			"section": "技术说明",
		},
	}, nil
}

// TechnicalWriter Agent
// 负责文档撰写

type TechnicalWriterAgent struct {
	Config chatdoc.RoleConfig
}

func (a *TechnicalWriterAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *TechnicalWriterAgent) HandleTask(ctx context.Context, s *chatDocService, task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("TechnicalWriterAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx = context.WithValue(ctx, constants.SystemPrompt, sysPrompt)

	reader, err := s.parent.agentChat(ctx, task.Content, "")
	if err != nil {
		klog.Errorf("TechnicalWriterAgent 处理任务 chat 失败: %v", err)
		return chatdoc.Task{}, err

	}
	all, err := s.parent.readAndWrite(ctx, reader)
	if err != nil {
		klog.Errorf("TechnicalWriterAgent 处理任务 readAndWrite 失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:     "UserExperienceReviewer",
		Type:     "review",
		Content:  all,
		Metadata: map[string]string{},
	}, nil
}

// UserExperienceReviewer Agent
// 负责用户体验评审

type UserExperienceReviewerAgent struct {
	Config chatdoc.RoleConfig
}

func (a *UserExperienceReviewerAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *UserExperienceReviewerAgent) HandleTask(ctx context.Context, s *chatDocService, task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("UserExperienceReviewerAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx = context.WithValue(ctx, constants.SystemPrompt, sysPrompt)

	reader, err := s.parent.agentChat(ctx, task.Content, "")
	if err != nil {
		klog.Errorf("UserExperienceReviewerAgent 处理任务 chat 失败: %v", err)
		return chatdoc.Task{}, err

	}
	all, err := s.parent.readAndWrite(ctx, reader)
	if err != nil {
		klog.Errorf("UserExperienceReviewerAgent 处理任务 readAndWrite 失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:     "DocumentationLeader",
		Type:     "feedback",
		Content:  all,
		Metadata: map[string]string{},
		IsFinal:  true,
	}, nil
}

func init() {
	RegisterAgent("DocumentationLeader", &DocumentationLeaderAgent{})
	RegisterAgent("CodeAnalyster", &CodeAnalysterAgent{})
	RegisterAgent("TechnicalWriter", &TechnicalWriterAgent{})
	RegisterAgent("UserExperienceReviewer", &UserExperienceReviewerAgent{})
}
