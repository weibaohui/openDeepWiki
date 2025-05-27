package chatdoc

import (
	"fmt"
	"os"

	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"gopkg.in/yaml.v3"
)

// 加载新版角色配置
func LoadRoleConfigs(path string) ([]chatdoc.RoleConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var roles []chatdoc.RoleConfig
	err = yaml.Unmarshal(f, &roles)
	return roles, err
}

// 加载新版工作流配置
func LoadWorkflowConfig(path string) (*chatdoc.WorkflowConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wf chatdoc.WorkflowConfig
	err = yaml.Unmarshal(f, &wf)
	return &wf, err
}

// 主流程调度骨架
func ExecuteWorkflow(initialTask chatdoc.Task, wf *chatdoc.WorkflowConfig) error {
	currentTask := initialTask
	currentTask.Role = wf.StartRole
	for {
		agent, ok := RegisteredAgents[currentTask.Role]
		if !ok {
			return fmt.Errorf("未知角色: %s", currentTask.Role)
		}
		nextTask, err := agent.HandleTask(currentTask)
		if err != nil {
			return err
		}
		// TODO: 根据 workflow steps 匹配流转、条件、元数据
		if nextTask.IsFinal {
			return nil
		}
		currentTask = nextTask
	}
}

// 对外API：启动动态多角色协作流程
func StartWorkflow(initialContent string) error {
	roles, err := LoadRoleConfigs("data/chatdoc_roles.yaml")
	if err != nil {
		return err
	}
	wf, err := LoadWorkflowConfig("data/chatdoc_workflow.yaml")
	if err != nil {
		return err
	}
	// 注册所有角色并注入 config
	for _, r := range roles {
		agent, ok := RegisteredAgents[r.Name]
		if !ok {
			return fmt.Errorf("未注册的智能体处理器: %s", r.Name)
		}
		RegisterAgentWithConfig(r.Name, agent, r)
	}
	initTask := chatdoc.Task{
		Content:  initialContent,
		Metadata: map[string]string{},
	}
	return ExecuteWorkflow(initTask, wf)
}
