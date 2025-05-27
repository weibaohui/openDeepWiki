package chatdoc

import (
	"context"
	"strings"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"k8s.io/klog/v2"
)

// 通用 Agent

type GenericAgent struct {
	Config   chatdoc.RoleConfig
	NextRole string
	TaskType string
	IsFinal  bool
}

func (a *GenericAgent) SetConfig(cfg chatdoc.RoleConfig) { a.Config = cfg }

func (a *GenericAgent) HandleTask(ctx context.Context, s *chatDocService, task chatdoc.Task) (chatdoc.Task, error) {
	klog.Infof("%s 处理任务: %s", a.Config.Name, utils.ToJSON(task))
	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{需求描述}}", task.Content)
	ctx = context.WithValue(ctx, constants.SystemPrompt, sysPrompt)

	reader, err := s.parent.agentChat(ctx, task.Content, "")
	if err != nil {
		klog.Errorf("%s 处理任务 chat 失败: %v", a.Config.Name, err)
		return chatdoc.Task{}, err
	}
	all, err := s.parent.readAndWrite(ctx, reader)
	if err != nil {
		klog.Errorf("%s 处理任务 readAndWrite 失败: %v", a.Config.Name, err)
		return chatdoc.Task{}, err
	}

	return chatdoc.Task{
		Role:     a.NextRole,
		Type:     a.TaskType,
		Content:  all,
		Metadata: map[string]string{},
		IsFinal:  a.IsFinal,
	}, nil
}
