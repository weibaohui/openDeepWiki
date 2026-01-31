package einodoc

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
func (t *GitCloneTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	klog.V(6).Infof("[GitCloneTool] 执行克隆: arguments=%s", arguments)

	var args struct {
		RepoURL   string `json:"repo_url"`
		TargetDir string `json:"target_dir"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[GitCloneTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.TargetDir == "" {
		args.TargetDir = generateRepoDirName(args.RepoURL)
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

	klog.V(6).Infof("[GitCloneTool] 克隆成功: resultLength=%d", len(result))
	return result, nil
}

// ListDirTool 目录列表工具
// 实现 Eino 的 tool.BaseTool 接口，用于列出目录内容
type ListDirTool struct {
	basePath string // 基础路径
}

// NewListDirTool 创建目录列表工具
// basePath: 操作的基础路径
func NewListDirTool(basePath string) *ListDirTool {
	klog.V(6).Infof("[ListDirTool] 创建工具实例: basePath=%s", basePath)
	return &ListDirTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *ListDirTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[ListDirTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "list_dir",
		Desc: "List directory contents with file information",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"dir": {
				Type: schema.String,
				Desc: "Directory path to list",
			},
			"recursive": {
				Type: schema.Boolean,
				Desc: "List recursively",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 列出指定目录的内容
func (t *ListDirTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	klog.V(6).Infof("[ListDirTool] 执行目录列表: arguments=%s", arguments)

	var args struct {
		Dir       string `json:"dir"`
		Recursive bool   `json:"recursive"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[ListDirTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	klog.V(6).Infof("[ListDirTool] 列出目录: dir=%s, recursive=%v", args.Dir, args.Recursive)

	listArgs, _ := json.Marshal(tools.ListDirArgs{
		Dir:       args.Dir,
		Recursive: args.Recursive,
	})

	result, err := tools.ListDir(listArgs, t.basePath)
	if err != nil {
		klog.Errorf("[ListDirTool] 列出目录失败: %v", err)
		return "", err
	}

	klog.V(6).Infof("[ListDirTool] 目录列表成功: 内容长度=%d", len(result))
	return result, nil
}

// ReadFileTool 文件读取工具
// 实现 Eino 的 tool.BaseTool 接口，用于读取文件内容
type ReadFileTool struct {
	basePath string // 基础路径
}

// NewReadFileTool 创建文件读取工具
// basePath: 操作的基础路径
func NewReadFileTool(basePath string) *ReadFileTool {
	klog.V(6).Infof("[ReadFileTool] 创建工具实例: basePath=%s", basePath)
	return &ReadFileTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[ReadFileTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "Read content of a file",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type: schema.String,
				Desc: "File path to read",
			},
			"limit": {
				Type: schema.Integer,
				Desc: "Maximum lines to read (optional, default 100)",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 读取指定文件的内容
func (t *ReadFileTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	klog.V(6).Infof("[ReadFileTool] 执行文件读取: arguments=%s", arguments)

	var args struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[ReadFileTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Limit == 0 {
		args.Limit = 100 // 默认读取 100 行
	}

	klog.V(6).Infof("[ReadFileTool] 读取文件: path=%s, limit=%d", args.Path, args.Limit)

	readArgs, _ := json.Marshal(tools.ReadFileArgs{
		Path:  args.Path,
		Limit: args.Limit,
	})

	result, err := tools.ReadFile(readArgs, t.basePath)
	if err != nil {
		klog.Errorf("[ReadFileTool] 读取文件失败: %v", err)
		return "", err
	}

	klog.V(6).Infof("[ReadFileTool] 读取文件成功: 内容长度=%d", len(result))
	return result, nil
}

// SearchFilesTool 文件搜索工具
// 实现 Eino 的 tool.BaseTool 接口，用于搜索匹配的文件
type SearchFilesTool struct {
	basePath string // 基础路径
}

// NewSearchFilesTool 创建文件搜索工具
// basePath: 操作的基础路径
func NewSearchFilesTool(basePath string) *SearchFilesTool {
	klog.V(6).Infof("[SearchFilesTool] 创建工具实例: basePath=%s", basePath)
	return &SearchFilesTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *SearchFilesTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[SearchFilesTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "search_files",
		Desc: "Search for files matching a pattern",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {
				Type: schema.String,
				Desc: "Glob pattern to match files",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 搜索匹配的文件
func (t *SearchFilesTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	klog.V(6).Infof("[SearchFilesTool] 执行文件搜索: arguments=%s", arguments)

	var args struct {
		Pattern string `json:"pattern"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[SearchFilesTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	klog.V(6).Infof("[SearchFilesTool] 搜索文件: pattern=%s", args.Pattern)

	searchArgs, _ := json.Marshal(tools.SearchFilesArgs{
		Pattern: args.Pattern,
	})

	result, err := tools.SearchFiles(searchArgs, t.basePath)
	if err != nil {
		klog.Errorf("[SearchFilesTool] 搜索文件失败: %v", err)
		return "", err
	}

	klog.V(6).Infof("[SearchFilesTool] 搜索文件成功: 匹配文件数=%d", len(result))
	return result, nil
}

// CreateTools 创建工具列表
// 返回所有可用的 Eino Tools
// basePath: 工具操作的基础路径
func CreateTools(basePath string) []tool.BaseTool {
	klog.V(6).Infof("[CreateTools] 创建工具列表: basePath=%s", basePath)
	tools := []tool.BaseTool{
		NewGitCloneTool(basePath),
		NewListDirTool(basePath),
		NewReadFileTool(basePath),
		NewSearchFilesTool(basePath),
	}
	klog.V(6).Infof("[CreateTools] 工具列表创建完成: count=%d", len(tools))
	return tools
}

// generateRepoDirName 从 repo URL 生成目录名
// 从 URL 中提取仓库名称作为目录名
// 例如: https://github.com/user/repo.git -> repo
func generateRepoDirName(repoURL string) string {
	parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}

// isPathSafe 检查路径是否在基础路径内
// 防止目录遍历攻击
// basePath: 允许的基础路径
// targetPath: 目标路径
// 返回: 是否安全
func isPathSafe(basePath, targetPath string) bool {
	cleanBase := filepath.Clean(basePath)
	cleanTarget := filepath.Clean(targetPath)
	return strings.HasPrefix(cleanTarget, cleanBase)
}
