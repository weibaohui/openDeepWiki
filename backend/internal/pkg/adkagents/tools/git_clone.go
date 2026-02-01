package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/pkg/llm/tools"
)

// GitCloneTool Git 克隆工具
// 实现 Eino 的 tool.BaseTool 接口，用于克隆远程 Git 仓库到本地
type GitCloneTool struct {
	basePath string // 基础路径，所有克隆操作都在此路径下进行
}

// NewGitCloneTool 创建 Git 克隆工具
// basePath: 仓库存储的基础路径
func NewGitCloneTool(basePath string) *GitCloneTool {
	klog.V(6).Infof("[GitCloneTool] 创建工具实例: basePath=%s", basePath)
	return &GitCloneTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口，描述工具的名称、用途和参数
func (t *GitCloneTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[GitCloneTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "git_clone",
		Desc: "Clone a Git repository to local directory",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"repo_url": {
				Type: schema.String,
				Desc: "Repository URL to clone",
			},
			"target_dir": {
				Type: schema.String,
				Desc: "Target directory name (optional, auto-generated if not provided)",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 实现 tool.BaseTool 接口，执行实际的 Git 克隆操作
// arguments: JSON 格式的参数字符串
// 返回: 克隆结果信息或错误
// 注意: 工具调用的输入输出日志由 EinoCallbacks 处理，此处仅记录业务相关日志
func (t *GitCloneTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	var args struct {
		RepoURL   string `json:"repo_url"`
		TargetDir string `json:"target_dir"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[GitCloneTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.TargetDir == "" {
		args.TargetDir = GenerateRepoDirName(args.RepoURL)
		klog.V(6).Infof("[GitCloneTool] 自动生成目标目录名: %s", args.TargetDir)
	}

	// 检查目标目录是否已存在
	targetPath := filepath.Join(t.basePath, args.TargetDir)
	if _, err := os.Stat(targetPath); err == nil {
		klog.V(6).Infof("[GitCloneTool] 目标目录已存在: %s", targetPath)
		//删除目标目录
		if err := os.RemoveAll(targetPath); err != nil {
			klog.Errorf("[GitCloneTool] 删除目标目录失败: %v", err)
			return "", fmt.Errorf("failed to remove existing directory: %w", err)
		}
		klog.V(6).Infof("[GitCloneTool] 目标目录删除成功: %s", targetPath)
	}

	klog.V(6).Infof("[GitCloneTool] 开始克隆: repoURL=%s, targetDir=%s", args.RepoURL, args.TargetDir)

	gitArgs, _ := json.Marshal(tools.GitCloneArgs{
		RepoURL:   args.RepoURL,
		TargetDir: args.TargetDir,
		Depth:     1, // 浅克隆，加快克隆速度
	})

	result, err := tools.GitClone(gitArgs, t.basePath)
	if err != nil {
		klog.Errorf("[GitCloneTool] 克隆失败: %v", err)
		return "", err
	}

	return result, nil
}

// GenerateRepoDirName 从 repo URL 生成目录名
// 从 URL 中提取仓库名称作为目录名
// 例如: https://github.com/user/repo.git -> repo
func GenerateRepoDirName(repoURL string) string {
	parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}
