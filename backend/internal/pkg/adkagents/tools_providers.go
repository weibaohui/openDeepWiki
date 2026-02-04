package adkagents

import (
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents/tools"
)

// toolProvider 实现 adkagents.ToolProvider
type ToolProvider struct {
	BasePath string
	SkillDir string
}

// GetTool 获取指定名称的工具
func (p *ToolProvider) GetTool(name string) (tool.BaseTool, error) {
	switch name {
	case "list_dir":
		return tools.NewListDirTool(p.BasePath), nil
	case "read_file":
		return tools.NewReadFileTool(p.BasePath), nil
	case "search_files":
		return tools.NewSearchFilesTool(p.BasePath), nil
	case "list_skills":
		return tools.NewListSkillsTool(p.SkillDir), nil
	case "run_terminal_command":
		return tools.NewRunTerminalCommandTool(p.BasePath), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ListTools 列出所有可用工具名称
func (p *ToolProvider) ListTools() []string {
	return []string{"list_dir", "read_file", "search_files", "list_skills"}
}
