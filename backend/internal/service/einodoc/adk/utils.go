package adk

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
)

// BuildWorkflowInput 构建 Workflow 输入
func BuildWorkflowInput(repoURL string) *adk.AgentInput {
	return &adk.AgentInput{
		Messages: []adk.Message{
			{
				Role:    schema.User,
				Content: fmt.Sprintf(`{"repo_url": "%s"}`, repoURL),
			},
		},
	}
}

// ParseAgentEvent 解析 Agent 事件，提取文本内容
func ParseAgentEvent(event *adk.AgentEvent) string {
	if event == nil {
		return ""
	}

	if event.Err != nil {
		return fmt.Sprintf("Error: %v", event.Err)
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		return event.Output.MessageOutput.Message.Content
	}

	return ""
}

// ExtractRepoInfoFromContent 从 Agent 输出内容提取仓库信息
func ExtractRepoInfoFromContent(content string) (*einodoc.RepoDocState, error) {
	// 尝试解析 JSON
	var result struct {
		RepoType  string            `json:"repo_type"`
		TechStack []string          `json:"tech_stack"`
		Summary   string            `json:"summary"`
		Chapters  []einodoc.Chapter `json:"chapters"`
		LocalPath string            `json:"local_path"`
	}

	if err := json.Unmarshal([]byte(extractJSON(content)), &result); err != nil {
		// 如果不是 JSON，返回空状态
		return nil, fmt.Errorf("failed to parse repo info: %w", err)
	}

	state := einodoc.NewRepoDocState("", result.LocalPath)
	state.SetRepoInfo(result.RepoType, result.TechStack)
	state.SetOutline(result.Chapters)

	return state, nil
}

// extractJSON 从文本中提取 JSON 部分
func extractJSON(content string) string {
	start := -1
	end := -1
	depth := 0

	for i, ch := range content {
		if ch == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 && start != -1 {
				end = i + 1
				break
			}
		}
	}

	if start >= 0 && end > start {
		return content[start:end]
	}

	return content
}

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(data)
}
