package einodoc

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/opendeepwiki/backend/internal/pkg/llm/tools"
)

// GitCloneTool Git 克隆工具
type GitCloneTool struct {
	basePath string
}

// NewGitCloneTool 创建 Git 克隆工具
func NewGitCloneTool(basePath string) *GitCloneTool {
	return &GitCloneTool{basePath: basePath}
}

// Info 返回工具信息
func (t *GitCloneTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
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
func (t *GitCloneTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	var args struct {
		RepoURL   string `json:"repo_url"`
		TargetDir string `json:"target_dir"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.TargetDir == "" {
		args.TargetDir = generateRepoDirName(args.RepoURL)
	}

	gitArgs, _ := json.Marshal(tools.GitCloneArgs{
		RepoURL:   args.RepoURL,
		TargetDir: args.TargetDir,
		Depth:     1,
	})

	return tools.GitClone(gitArgs, t.basePath)
}

// ListDirTool 目录列表工具
type ListDirTool struct {
	basePath string
}

// NewListDirTool 创建目录列表工具
func NewListDirTool(basePath string) *ListDirTool {
	return &ListDirTool{basePath: basePath}
}

// Info 返回工具信息
func (t *ListDirTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
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
func (t *ListDirTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Dir       string `json:"dir"`
		Recursive bool   `json:"recursive"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	listArgs, _ := json.Marshal(tools.ListDirArgs{
		Dir:       args.Dir,
		Recursive: args.Recursive,
	})

	return tools.ListDir(listArgs, t.basePath)
}

// ReadFileTool 文件读取工具
type ReadFileTool struct {
	basePath string
}

// NewReadFileTool 创建文件读取工具
func NewReadFileTool(basePath string) *ReadFileTool {
	return &ReadFileTool{basePath: basePath}
}

// Info 返回工具信息
func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
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
func (t *ReadFileTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Limit == 0 {
		args.Limit = 100
	}

	readArgs, _ := json.Marshal(tools.ReadFileArgs{
		Path:  args.Path,
		Limit: args.Limit,
	})

	return tools.ReadFile(readArgs, t.basePath)
}

// SearchFilesTool 文件搜索工具
type SearchFilesTool struct {
	basePath string
}

// NewSearchFilesTool 创建文件搜索工具
func NewSearchFilesTool(basePath string) *SearchFilesTool {
	return &SearchFilesTool{basePath: basePath}
}

// Info 返回工具信息
func (t *SearchFilesTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
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
func (t *SearchFilesTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Pattern string `json:"pattern"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	searchArgs, _ := json.Marshal(tools.SearchFilesArgs{
		Pattern: args.Pattern,
	})

	return tools.SearchFiles(searchArgs, t.basePath)
}

// CreateTools 创建工具列表
func CreateTools(basePath string) []tool.BaseTool {
	return []tool.BaseTool{
		NewGitCloneTool(basePath),
		NewListDirTool(basePath),
		NewReadFileTool(basePath),
		NewSearchFilesTool(basePath),
	}
}

// generateRepoDirName 从 repo URL 生成目录名
func generateRepoDirName(repoURL string) string {
	parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}

// isPathSafe 检查路径是否在基础路径内
func isPathSafe(basePath, targetPath string) bool {
	cleanBase := filepath.Clean(basePath)
	cleanTarget := filepath.Clean(targetPath)
	return strings.HasPrefix(cleanTarget, cleanBase)
}
