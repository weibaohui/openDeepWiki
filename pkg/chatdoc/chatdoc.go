package chatdoc

import (
	"context"
	"fmt"
	"os"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type chatDocService struct {
	parent *docService
}

func (s *docService) ChatDocService() *chatDocService {
	return &chatDocService{
		parent: s,
	}
}

// LoadRoleConfigs 加载新版角色配置
func (s *chatDocService) LoadRoleConfigs(path string) ([]chatdoc.RoleConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var roles []chatdoc.RoleConfig
	err = yaml.Unmarshal(f, &roles)
	return roles, err
}

// LoadWorkflowConfig 加载新版工作流配置
func (s *chatDocService) LoadWorkflowConfig(path string) (*chatdoc.WorkflowConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wf chatdoc.WorkflowConfig
	err = yaml.Unmarshal(f, &wf)
	return &wf, err
}

// ExecuteWorkflow 主流程调度骨架
func (s *chatDocService) ExecuteWorkflow(ctx context.Context, initialTask chatdoc.Task, wf *chatdoc.WorkflowConfig) error {
	currentTask := initialTask
	currentTask.Role = wf.StartRole
	for {
		agent, ok := RegisteredAgents[currentTask.Role]
		if !ok {
			return fmt.Errorf("未知角色: %s", currentTask.Role)
		}
		nextTask, err := agent.HandleTask(ctx, s, currentTask)
		if err != nil {
			return err
		}
		klog.Infof(" 下一个任务: %s", utils.ToJSON(nextTask))
		// TODO: 根据 workflow steps 匹配流转、条件、元数据
		if nextTask.IsFinal {
			return nil
		}
		currentTask = nextTask
	}
}

// StartWorkflow 对外API：启动动态多角色协作流程
func (s *chatDocService) StartWorkflow(ctx context.Context, initialContent string) error {
	roles, err := s.LoadRoleConfigs("config/chatdoc_roles.yaml")
	if err != nil {
		klog.Errorf("加载角色配置失败: %v", err)
		return err
	}
	wf, err := s.LoadWorkflowConfig("config/chatdoc_workflow.yaml")
	if err != nil {
		klog.Errorf("加载工作流配置失败: %v", err)
		return err
	}

	// 注册Agent
	for _, r := range roles {
		agent := &GenericAgent{}
		RegisterAgentWithConfig(r.Name, agent, r)
		klog.Infof("注册智能体: %s", utils.ToJSON(r))
	}
 
	initTask := chatdoc.Task{
		Content:  initialContent,
		Metadata: map[string]string{},
	}
	return s.ExecuteWorkflow(ctx, initTask, wf)
}
