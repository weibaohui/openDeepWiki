package chatdoc

import (
	"context"
	"fmt"
	"os"

	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type ChatDocService struct {
	Roles         []chatdoc.Role
	Collaboration []chatdoc.Collaboration
}

func NewChatDocService() *ChatDocService {
	roles := loadRoles()
	collab := loadCollaboration()
	return &ChatDocService{
		Roles:         roles,
		Collaboration: collab,
	}
}

func loadRoles() []chatdoc.Role {
	f, err := os.ReadFile("data/chatdoc_roles.yaml")
	if err != nil {
		return nil
	}
	var data struct {
		Roles []chatdoc.Role `yaml:"roles"`
	}
	_ = yaml.Unmarshal(f, &data)
	return data.Roles
}

func loadCollaboration() []chatdoc.Collaboration {
	f, err := os.ReadFile("data/chatdoc_collaboration.yaml")
	if err != nil {
		return nil
	}
	var data struct {
		Collaboration []chatdoc.Collaboration `yaml:"collaboration"`
	}
	_ = yaml.Unmarshal(f, &data)
	return data.Collaboration
}

// 示例：发起协作会话
// StartSession 支持传入初始任务描述
func (svc *ChatDocService) StartSession(ctx context.Context, initialTask string) *chatdoc.ChatDocSession {
	return &chatdoc.ChatDocSession{
		ID:           "session-id",
		CurrentStage: "init",
		History:      []string{},
		Roles:        svc.Roles,
		InitialTask:  initialTask,
	}
}

// ExecuteTask: 以 initialTask 作为 prompt，调用 AI 聊天服务，返回 AI 回复并追加到历史
func (svc *ChatDocService) ExecuteTask(ctx context.Context, session *chatdoc.ChatDocSession) (string, error) {
	if session.InitialTask == "" {
		return "", nil
	}
	ctxInst := ctx
	stream, err := service.ChatService().GetChatStream(ctxInst, session.InitialTask)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var result string
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		if len(resp.Choices) > 0 {
			result += resp.Choices[0].Delta.Content
		}
	}
	if result != "" {
		session.History = append(session.History, result)
	}
	return result, nil
}

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
		klog.Infof("注册角色: %s, 类型: %s, 描述: %s", r.Name, r.Type, r.Description)
	}
	initTask := chatdoc.Task{
		Content:  initialContent,
		Metadata: map[string]string{},
	}
	return ExecuteWorkflow(initTask, wf)
}
