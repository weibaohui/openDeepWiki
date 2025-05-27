package chatdoc

import (
	"context"
	"strings"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

// DocumentationLeader Agent
// 负责文档整体流程、任务分解与协调

type DocumentationLeaderAgent struct {
	Config chatdoc.RoleConfig
}

func (a *DocumentationLeaderAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *DocumentationLeaderAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("DocumentationLeaderAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)

	client, err := service.AIService().DefaultClient()
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	resp, err := client.GetCompletionNoHistory(ctx, task.Content)
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "CodeAnalyster",
		Type:    "analyze",
		Content: resp,
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

func (a *CodeAnalysterAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("CodeAnalysterAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)
	client, err := service.AIService().DefaultClient()
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	resp, err := client.GetCompletionNoHistory(ctx, task.Content)
	if err != nil {
		klog.Errorf("CodeAnalysterAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "TechnicalWriter",
		Type:    "write",
		Content: resp,
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

func (a *TechnicalWriterAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("TechnicalWriterAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)
	client, err := service.AIService().DefaultClient()
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	resp, err := client.GetCompletionNoHistory(ctx, task.Content)
	if err != nil {
		klog.Errorf("TechnicalWriterAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "UserExperienceReviewer",
		Type:    "review",
		Content: resp,
		Metadata: map[string]string{
			"content": resp,
		},
	}, nil
}

// UserExperienceReviewer Agent
// 负责用户体验评审

type UserExperienceReviewerAgent struct {
	Config chatdoc.RoleConfig
}

func (a *UserExperienceReviewerAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *UserExperienceReviewerAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("UserExperienceReviewerAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)
	client, err := service.AIService().DefaultClient()
	if err != nil {
		klog.Errorf("DocumentationLeaderAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	resp, err := client.GetCompletionNoHistory(ctx, task.Content)
	if err != nil {
		klog.Errorf("UserExperienceReviewerAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "DocumentationLeader",
		Type:    "feedback",
		Content: resp,
		Metadata: map[string]string{
			"review_comments": resp,
		},
		IsFinal: true,
	}, nil
}

func init() {
	RegisterAgent("DocumentationLeader", &DocumentationLeaderAgent{})
	RegisterAgent("CodeAnalyster", &CodeAnalysterAgent{})
	RegisterAgent("TechnicalWriter", &TechnicalWriterAgent{})
	RegisterAgent("UserExperienceReviewer", &UserExperienceReviewerAgent{})
}
