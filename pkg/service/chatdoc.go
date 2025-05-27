package service

import (
	"fmt"
	"os"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// chatDocService 服务结构体
// 用于多角色协作流程相关方法
// 你可以通过实例化 chatDocService{} 来调用相关方法
type chatDocService struct{}

func NewChatDocService() *chatDocService {
	return &chatDocService{}
}

// 加载新版角色配置
func (s *chatDocService) LoadRoleConfigs(path string) ([]chatdoc.RoleConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var roles []chatdoc.RoleConfig
	err = yaml.Unmarshal(f, &roles)
	return roles, err
}

// 加载新版工作流配置
func (s *chatDocService) LoadWorkflowConfig(path string) (*chatdoc.WorkflowConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wf chatdoc.WorkflowConfig
	err = yaml.Unmarshal(f, &wf)
	return &wf, err
}

// 主流程调度骨架
func (s *chatDocService) ExecuteWorkflow(initialTask chatdoc.Task, wf *chatdoc.WorkflowConfig) error {
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
		klog.Infof(" 下一个任务: %s", utils.ToJSON(nextTask))
		// TODO: 根据 workflow steps 匹配流转、条件、元数据
		if nextTask.IsFinal {
			return nil
		}
		currentTask = nextTask
	}
}

// 对外API：启动动态多角色协作流程
func (s *chatDocService) StartWorkflow(initialContent string) error {
	roles, err := s.LoadRoleConfigs("data/chatdoc_roles.yaml")
	if err != nil {
		klog.Errorf("加载角色配置失败: %v", err)
		return err
	}
	wf, err := s.LoadWorkflowConfig("data/chatdoc_workflow.yaml")
	if err != nil {
		klog.Errorf("加载工作流配置失败: %v", err)
		return err
	}
	// 注册所有角色并注入 config
	for _, r := range roles {
		agent, ok := RegisteredAgents[r.Name]
		if !ok {
			return fmt.Errorf("未注册的智能体处理器: %s", r.Name)
		}
		RegisterAgentWithConfig(r.Name, agent, r)
		klog.Infof("注册智能体: %s", utils.ToJSON(r))
	}
	initTask := chatdoc.Task{
		Content:  initialContent,
		Metadata: map[string]string{},
	}
	return s.ExecuteWorkflow(initTask, wf)
}
