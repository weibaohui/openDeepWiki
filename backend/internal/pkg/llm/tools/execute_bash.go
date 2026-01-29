package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ExecuteBashArgs execute_bash 工具参数
type ExecuteBashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// 危险命令黑名单模式
var dangerousPatterns = []string{
	`\brm\s+-[rf]*[rf]`, // rm -rf, rm -f, rm -r
	`\brm\s+\/`,          // rm /
	`>\s*\/`,             // 重定向到根目录
	`>>\s*\/`,            // 追加重定向到根目录
	`;`,                  // 命令分隔符
	`\|\s*rm`,            // 管道到 rm
	`\|\s*sh`,            // 管道到 sh
	`\$\(`,                // 命令替换 $()
	"`",                  // 反引号命令替换
	`&&\s*rm`,            // && rm
	`\bdd\s`,              // dd 命令
	`\bmv\s+.*\/`,         // mv 到系统目录
	`\bchmod\s+.*\/`,      // chmod 系统目录
	`\bchown\s+.*\/`,      // chown 系统目录
	`curl.*\|`,            // curl | sh
	`wget.*\|`,            // wget | sh
	`eval\s*\(`,           // eval
	`exec\s*\(`,           // exec
	`system\s*\(`,         // system
	`>.*\.env`,            // 覆盖环境文件
	`>.*config`,           // 覆盖配置文件
}

// 允许的命令白名单（可选使用更严格的策略）
var allowedCommands = []string{
	"find", "grep", "wc", "cat", "echo", "ls", "pwd", "head", "tail",
	"sort", "uniq", "cut", "awk", "sed", "tr", "dirname", "basename",
	"git", "go", "npm", "yarn", "node", "sleep", "yes",
}

// ExecuteBash 执行 bash 命令
func ExecuteBash(args json.RawMessage, basePath string) (string, error) {
	var params ExecuteBashArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// 安全检查
	if err := validateCommand(params.Command); err != nil {
		return "", err
	}

	// 设置超时
	timeout := params.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 120 {
		timeout = 120 // 最大 120 秒
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 执行命令
	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
	cmd.Dir = basePath // 设置工作目录

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 处理超时
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %d seconds", timeout)
	}

	if err != nil {
		// 命令执行出错，但可能有输出
		if outputStr != "" {
			return outputStr, nil // 返回输出，即使命令返回非零退出码
		}
		return "", fmt.Errorf("command failed: %w", err)
	}

	// 限制输出长度
	const maxOutput = 10000
	if len(outputStr) > maxOutput {
		outputStr = outputStr[:maxOutput] + fmt.Sprintf("\n... (%d more bytes truncated)", len(outputStr)-maxOutput)
	}

	return outputStr, nil
}

// validateCommand 验证命令安全性
func validateCommand(command string) error {
	// 1. 检查危险模式
	for _, pattern := range dangerousPatterns {
		matched, err := regexp.MatchString(pattern, command)
		if err != nil {
			continue
		}
		if matched {
			return fmt.Errorf("dangerous pattern detected in command: %s", pattern)
		}
	}

	// 2. 检查是否包含路径遍历
	if strings.Contains(command, "..") {
		return fmt.Errorf("path traversal not allowed in command")
	}

	// 3. 可选：严格模式 - 只允许白名单命令
	// 解析命令中的第一个词
	cmd := strings.Fields(command)
	if len(cmd) == 0 {
		return fmt.Errorf("empty command")
	}

	// 提取基础命令名（去除路径）
	baseCmd := filepath.Base(cmd[0])

	// 检查是否在白名单中
	allowed := false
	for _, c := range allowedCommands {
		if baseCmd == c {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("command not in allowed list: %s (allowed: %s)",
			baseCmd, strings.Join(allowedCommands, ", "))
	}

	return nil
}
