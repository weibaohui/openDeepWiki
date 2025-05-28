package chatdoc

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type chatDocService struct {
	parent   *docService
	workflow *chatdoc.WorkflowConfig
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

// executeStep 递归执行单个步骤及其所有子步骤
func (s *chatDocService) executeStep(
	ctx context.Context,
	step chatdoc.WorkflowStep,
	currentTask chatdoc.Task,
	outputs map[string]string,
	depth int,
) (chatdoc.Task, error) {
	prefix := strings.Repeat("  ", depth)
	klog.Infof("%s执行步骤: %s", prefix, step.Step)

	// 获取该步骤对应的 agent
	agent, ok := RegisteredAgents[step.Actor]
	if !ok {
		return chatdoc.Task{}, fmt.Errorf("步骤 '%s' 的角色未注册: %s", step.Step, step.Actor)
	}

	// 准备当前步骤的输入
	stepTask := currentTask
	stepTask.Role = step.Actor
	stepTask.Step = step.Step
	stepTask.Inputs = step.Input
	stepTask.Outputs = step.Output

	// ----------- 新增：确保 Metadata 累积并传递 -----------
	if stepTask.Metadata == nil {
		stepTask.Metadata = make(map[string]string)
	}
	// 记录当前步骤信息到 Metadata
	stepTask.Metadata["当前步骤"] = step.Step
	stepTask.Metadata["当前角色"] = step.Actor
	// ------------------------------------------------------

	// 从之前步骤的输出中收集所需的输入
	inputContent := stepTask.Content
	for _, requiredInput := range step.Input {
		if output, exists := outputs[requiredInput]; exists {
			inputContent += "\n\n关于 " + requiredInput + ":\n" + output
		}
	}
	stepTask.Content = inputContent

	// 执行子步骤
	if len(step.Substeps) > 0 {
		klog.Infof("%s处理子步骤，共 %d 个", prefix, len(step.Substeps))
		subStepOutputs := make(map[string]string)

		for subIndex, substep := range step.Substeps {
			klog.Infof("%s子步骤 %d/%d: %s", prefix, subIndex+1, len(step.Substeps), substep.Step)

			// 递归执行子步骤
			subTask := stepTask
			// ----------- 新增：传递累积的 Metadata -----------
			if subTask.Metadata == nil {
				subTask.Metadata = make(map[string]string)
			}
			for k, v := range stepTask.Metadata {
				subTask.Metadata[k] = v
			}
			// -----------------------------------------------
			subTask, err := s.executeStep(ctx, substep, subTask, outputs, depth+1)
			if err != nil {
				return chatdoc.Task{}, fmt.Errorf("子步骤 '%s' 执行失败: %v", substep.Step, err)
			}

			// 保存子步骤的输出
			for _, output := range substep.Output {
				subStepOutputs[output] = subTask.Content
			}
		}

		// 将子步骤的输出合并到当前步骤的输入中
		for output, content := range subStepOutputs {
			stepTask.Content += "\n\n子步骤输出 " + output + ":\n" + content
		}
	}

	// 执行当前步骤
	nextTask, err := agent.HandleTask(ctx, s, stepTask)
	if err != nil {
		return chatdoc.Task{}, fmt.Errorf("步骤 '%s' 执行失败: %v", step.Step, err)
	}

	// ----------- 新增：累积 Metadata 到 nextTask -----------
	if nextTask.Metadata == nil {
		nextTask.Metadata = make(map[string]string)
	}
	for k, v := range stepTask.Metadata {
		nextTask.Metadata[k] = v
	}
	// ------------------------------------------------------

	// 保存步骤的输出供后续使用
	for _, output := range step.Output {
		outputs[output] = nextTask.Content
	}

	klog.Infof("%s步骤 '%s' 执行完成", prefix, step.Step)
	return nextTask, nil
}

// ExecuteWorkflow 按工作流配置的步骤顺序执行
func (s *chatDocService) ExecuteWorkflow(ctx context.Context, initialTask chatdoc.Task, wf *chatdoc.WorkflowConfig) error {
	if len(wf.Steps) == 0 {
		return fmt.Errorf("工作流步骤为空")
	}

	currentTask := initialTask
	outputs := make(map[string]string) // 存储每个步骤的输出，用于后续步骤的输入

	// 遍历顶层步骤
	for stepIndex, step := range wf.Steps {
		klog.Infof("执行主流程步骤 %d/%d: %s", stepIndex+1, len(wf.Steps), step.Step)

		nextTask, err := s.executeStep(ctx, step, currentTask, outputs, 0)
		if err != nil {
			return fmt.Errorf("步骤 '%s' 执行失败: %v", step.Step, err)
		}

		if nextTask.IsFinal {
			klog.Infof("工作流执行完成，最后步骤: %s", step.Step)
			return nil
		}

		currentTask = nextTask
	}

	return nil
}

// StartWorkflow 对外API：启动动态多角色协作流程
func (s *chatDocService) StartWorkflow(ctx context.Context, info *RepoInfo) error {
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
	s.workflow = wf
	// 注册Agent
	for _, r := range roles {
		agent := &GenericAgent{}
		RegisterAgentWithConfig(r.Name, agent, r)
		klog.Infof("注册智能体: %s", utils.ToJSON(r))
	}

	initTask := chatdoc.Task{
		Content: "请分析代码仓库并编写技术文档",
		Metadata: map[string]string{
			"repoName":    info.RepoName,
			"repoPath":    info.RepoPath,
			"docPath":     info.DocPath,
			"description": info.Description,
		},
	}
	klog.V(6).Infof("工作流配置: %s", utils.ToJSON(initTask))
	return s.ExecuteWorkflow(ctx, initTask, wf)
}
