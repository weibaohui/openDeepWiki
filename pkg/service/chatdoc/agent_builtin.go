package chatdoc

import (
	"fmt"

	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"k8s.io/klog/v2"
)

// Leader Agent 示例
// 实际可根据 prompt/AI 结果动态生成

type LeaderAgent struct {
	Config chatdoc.RoleConfig
}

func (a *LeaderAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *LeaderAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("LeaderAgent 处理任务: %s", task.Content)
	// 可用 a.Config.Description 等
	return chatdoc.Task{
		Role:    "Writer",
		Type:    "write",
		Content: fmt.Sprintf("请根据需求撰写文档: %s", task.Content),
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
	klog.Infof("WriterAgent 处理任务: %s", task.Content)
	// 简单模拟写作
	return chatdoc.Task{
		Role:    "Reviewer",
		Type:    "review",
		Content: fmt.Sprintf("文档内容: %s (已完成)", task.Content),
		Metadata: map[string]string{
			"content": task.Content,
		},
	}, nil
}

type ReviewerAgent struct {
	Config chatdoc.RoleConfig
}

func (a *ReviewerAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *ReviewerAgent) HandleTask(task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("ReviewerAgent 处理任务: %s", task.Content)
	// 简单模拟审核
	return chatdoc.Task{
		Role:    "Leader",
		Type:    "feedback",
		Content: "审核通过，流程结束。",
		Metadata: map[string]string{
			"review_comments": "很好",
		},
		IsFinal: true,
	}, nil
}

func init() {
	RegisterAgent("Leader", &LeaderAgent{})
	RegisterAgent("Writer", &WriterAgent{})
	RegisterAgent("Reviewer", &ReviewerAgent{})
}
