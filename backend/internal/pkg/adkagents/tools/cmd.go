package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// RunTerminalCommandTool allows agents to execute terminal commands.
type RunTerminalCommandTool struct {
	// WorkingDir is the default working directory for commands
	WorkingDir string
	// Timeout is the maximum duration for command execution
	Timeout time.Duration
	// AllowedCommands is a list of allowed command prefixes (empty = allow all)
	AllowedCommands []string
}

// RunTerminalCommandArgs defines the arguments for run_terminal_command tool.
type RunTerminalCommandArgs struct {
	// Command is the shell command to execute
	Command string `json:"command"`
	// WorkingDir optionally specifies the working directory
	WorkingDir string `json:"working_dir,omitempty"`
}

// NewRunTerminalCommandTool creates a new run_terminal_command tool.
func NewRunTerminalCommandTool(workingDir string) *RunTerminalCommandTool {
	return &RunTerminalCommandTool{
		WorkingDir: workingDir,
		Timeout:    30 * time.Second,
		// AllowedCommands: []string{
		// 	"uv", "python", "cd", "file", "find", "grep", "tree", "wc", "cat", "echo", "ls", "pwd", "head", "tail",
		// 	"sort", "uniq", "cut", "awk", "sed", "tr", "dirname", "basename",
		// 	"git", "go", "npm", "yarn", "node", "sleep", "yes",
		// },
	}
}

// Info returns the tool's schema information.
func (t *RunTerminalCommandTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run_terminal_command",
		Desc: `Execute a terminal/shell command and return the output.
Use this tool to:
- Run git commands (git status, git diff, git commit,tree --charset utf-8 etc.)
- Execute build commands
- Run scripts (python3 <script>,uv run <script>)
- Check file contents with cat, ls, etc.

Returns stdout and stderr from the command execution.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "The shell command to execute (e.g., 'git status', 'ls -la','tree --charset utf-8')",
				Required: true,
			},
			"working_dir": {
				Type:     schema.String,
				Desc:     "Optional working directory for the command. Defaults to current directory.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool and returns the command output.
func (t *RunTerminalCommandTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args RunTerminalCommandArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Check allowed commands if configured
	if len(t.AllowedCommands) > 0 {
		allowed := false
		for _, prefix := range t.AllowedCommands {
			if strings.HasPrefix(args.Command, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("command not allowed: %s", args.Command)
		}
	}

	// Determine working directory
	workingDir := t.WorkingDir
	if args.WorkingDir != "" {
		workingDir = args.WorkingDir
	}

	// Create command with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "sh", "-c", args.Command)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Build result
	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString("STDOUT:\n")
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("STDERR:\n")
		result.WriteString(stderr.String())
	}

	if err != nil {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("EXIT ERROR: %v", err))
	}

	if result.Len() == 0 {
		return "(command completed with no output)", nil
	}

	return result.String(), nil
}

// Ensure RunTerminalCommandTool implements tool.InvokableTool
var _ tool.InvokableTool = (*RunTerminalCommandTool)(nil)
