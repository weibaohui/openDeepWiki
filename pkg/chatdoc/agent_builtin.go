package chatdoc

import (
	"context"
	"fmt"
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

	// 从 task.Metadata 中获取用户需求
	userRequirement := task.Content
	if task.Metadata != nil {
		str := `
			仓库名称= %s 。
			仓库路径= %s 。
			文档路径= %s 。
		`
		userRequirement = fmt.Sprintf(str, task.Metadata["repoName"], task.Metadata["repoPath"], task.Metadata["docPath"])
	}

	sysPrompt := strings.ReplaceAll(a.Config.Prompt, "{{代码仓库信息}}", userRequirement)
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{input}}", strings.Join(task.Inputs, "\n"))
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{output}}", strings.Join(task.Outputs, ".md\n"))

	ctx = context.WithValue(ctx, constants.SystemPrompt, sysPrompt)

	reader, err := s.parent.agentChat(ctx, userRequirement)
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
