package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// ListSkillsTool 本地技能发现工具
type ListSkillsTool struct {
	skillDir string
}

// NewListSkillsTool 创建技能发现工具
// skillDir: 技能定义所在的目录
func NewListSkillsTool(skillDir string) *ListSkillsTool {
	klog.V(6).Infof("[ListSkillsTool] 创建工具实例: skillDir=%s", skillDir)
	return &ListSkillsTool{skillDir: skillDir}
}

// Info 返回工具信息
func (t *ListSkillsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_skills",
		Desc: "List all available local skills discovered in the system. Use this to find new capabilities.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			// 无需参数，列出所有
		}),
	}, nil
}

// InvokableRun 执行工具调用
func (t *ListSkillsTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	klog.V(6).Infof("[ListSkillsTool] 开始扫描技能目录: %s", t.skillDir)

	// 使用 Eino 的 LocalBackend 扫描技能
	sb, err := skill.NewLocalBackend(&skill.LocalBackendConfig{
		BaseDir: t.skillDir,
	})
	if err != nil {
		klog.Errorf("[ListSkillsTool] 创建 Skill Backend 失败: %v", err)
		return "", fmt.Errorf("failed to initialize skill backend: %w", err)
	}

	skills, err := sb.List(ctx)
	if err != nil {
		klog.Errorf("[ListSkillsTool] 获取技能列表失败: %v", err)
		return "", fmt.Errorf("failed to list skills: %w", err)
	}

	// 构造简化的返回结果
	type SkillSummary struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	summaries := make([]SkillSummary, 0, len(skills))
	for _, s := range skills {
		summaries = append(summaries, SkillSummary{
			Name:        s.Name,
			Description: s.Description,
		})
	}

	result, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	klog.V(6).Infof("[ListSkillsTool] 发现 %d 个技能", len(summaries))
	return string(result), nil
}
