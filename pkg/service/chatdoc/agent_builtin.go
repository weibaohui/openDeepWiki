package chatdoc

import (
	"context"
	"strings"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"github.com/weibaohui/openDeepWiki/pkg/service/ai"
	"k8s.io/klog/v2"
)

// Leader Agent 示例
// 实际可根据 prompt/AI 结果动态生成

type LeaderAgent struct {
	Config chatdoc.RoleConfig
}

func (a *LeaderAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *LeaderAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("LeaderAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)
	resp, err := ai.CallLLM(ctx, task.Content)
	if err != nil {
		klog.Errorf("LeaderAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "Writer",
		Type:    "write",
		Content: resp,
		Metadata: map[string]string{
			"section": "第一部分",
		},
	}, nil
}

type WriterAgent struct {
	Config chatdoc.RoleConfig
}

func (a *WriterAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *WriterAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("WriterAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)
	resp, err := ai.CallLLM(ctx, task.Content)
	if err != nil {
		klog.Errorf("WriterAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "Reviewer",
		Type:    "review",
		Content: resp,
		Metadata: map[string]string{
			"content": resp,
		},
	}, nil
}

type ReviewerAgent struct {
	Config chatdoc.RoleConfig
}

func (a *ReviewerAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *ReviewerAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("ReviewerAgent 处理任务: %s", utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx := context.WithValue(context.Background(), constants.SystemPrompt, sysPrompt)
	resp, err := ai.CallLLM(ctx, task.Content)
	if err != nil {
		klog.Errorf("ReviewerAgent 处理任务失败: %v", err)
		return chatdoc.Task{}, err
	}
	return chatdoc.Task{
		Role:    "Leader",
		Type:    "feedback",
		Content: resp,
		Metadata: map[string]string{
			"review_comments": resp,
		},
		IsFinal: true,
	}, nil
}

func init() {
	RegisterAgent("Leader", &LeaderAgent{})
	RegisterAgent("Writer", &WriterAgent{})
	RegisterAgent("Reviewer", &ReviewerAgent{})
}
