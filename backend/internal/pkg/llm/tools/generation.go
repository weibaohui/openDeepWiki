// Generation Tools - 内容生成工具
// 对应 MCP tools: generation.llm_generate, generation.generate_mermaid,
//                 generation.generate_diagram, generation.summarize, generation.translate

package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// LLMGenerateArgs generation.llm_generate 参数
type LLMGenerateArgs struct {
	Prompt      string                 `json:"prompt"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
}

// LLMGenerate 调用 LLM 生成内容（模拟/占位实现）
func LLMGenerate(args json.RawMessage, basePath string) (string, error) {
	var params LLMGenerateArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	// 这是一个占位实现
	// 实际实现应该调用外部 LLM API
	// 返回模拟响应
	return fmt.Sprintf(`[LLM Generation Placeholder]
Prompt: %s
Model: %s
Temperature: %f

Note: This is a placeholder. Actual implementation should call LLM API.
`, params.Prompt, params.Model, params.Temperature), nil
}

// GenerateMermaidArgs generation.generate_mermaid 参数
type GenerateMermaidArgs struct {
	CodeSnippet string `json:"code_snippet"`
	DiagramType string `json:"diagram_type,omitempty"`
	Title       string `json:"title,omitempty"`
}

// GenerateMermaid 从代码生成 Mermaid 图表
func GenerateMermaid(args json.RawMessage, basePath string) (string, error) {
	var params GenerateMermaidArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.CodeSnippet == "" {
		return "", fmt.Errorf("code_snippet is required")
	}
	if params.DiagramType == "" {
		params.DiagramType = "flowchart"
	}

	// 根据代码分析生成 Mermaid 图表
	var mermaid string

	switch params.DiagramType {
	case "flowchart":
		mermaid = generateFlowchart(params.CodeSnippet, params.Title)
	case "sequence":
		mermaid = generateSequence(params.CodeSnippet, params.Title)
	case "class":
		mermaid = generateClassDiagram(params.CodeSnippet, params.Title)
	default:
		mermaid = generateFlowchart(params.CodeSnippet, params.Title)
	}

	return mermaid, nil
}

// GenerateDiagramArgs generation.generate_diagram 参数
type GenerateDiagramArgs struct {
	Description string `json:"description"`
	DiagramType string `json:"diagram_type,omitempty"`
}

// GenerateDiagram 从描述生成架构图
func GenerateDiagram(args json.RawMessage, basePath string) (string, error) {
	var params GenerateDiagramArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Description == "" {
		return "", fmt.Errorf("description is required")
	}

	// 基于描述生成简单的架构图
	return generateArchitectureDiagram(params.Description), nil
}

// SummarizeArgs generation.summarize 参数
type SummarizeArgs struct {
	Text      string `json:"text"`
	MaxLength int    `json:"max_length,omitempty"`
	Style     string `json:"style,omitempty"` // concise, detailed, bullet_points
}

// Summarize 文本摘要
func Summarize(args json.RawMessage, basePath string) (string, error) {
	var params SummarizeArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Text == "" {
		return "", fmt.Errorf("text is required")
	}
	if params.MaxLength <= 0 {
		params.MaxLength = 200
	}
	if params.Style == "" {
		params.Style = "concise"
	}

	// 提取关键句子（简化实现）
	summary := extractSummary(params.Text, params.MaxLength, params.Style)
	return summary, nil
}

// TranslateArgs generation.translate 参数
type TranslateArgs struct {
	Text           string `json:"text"`
	TargetLanguage string `json:"target_language"`
	SourceLanguage string `json:"source_language,omitempty"`
}

// Translate 翻译文本（占位实现）
func Translate(args json.RawMessage, basePath string) (string, error) {
	var params TranslateArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Text == "" {
		return "", fmt.Errorf("text is required")
	}
	if params.TargetLanguage == "" {
		return "", fmt.Errorf("target_language is required")
	}

	// 这是一个占位实现
	// 实际实现应该调用翻译 API
	return fmt.Sprintf(`[Translation Placeholder]
Original (%s): %s
Target: %s

Note: This is a placeholder. Actual implementation should call translation API.
`, params.SourceLanguage, params.Text, params.TargetLanguage), nil
}

// 辅助函数：生成流程图
func generateFlowchart(code, title string) string {
	// 解析代码中的控制流
	lines := strings.Split(code, "\n")

	var nodes []string
	var edges []string
	nodeCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		nodeCount++
		nodeId := fmt.Sprintf("N%d", nodeCount)

		// 识别不同的代码结构
		switch {
		case strings.HasPrefix(line, "if ") || strings.HasPrefix(line, "if("):
			nodes = append(nodes, fmt.Sprintf("    %s{%s}", nodeId, truncate(line, 30)))
			edges = append(edges, fmt.Sprintf("    %s -->|true| %s_y", nodeId, nodeId))
			edges = append(edges, fmt.Sprintf("    %s -->|false| %s_n", nodeId, nodeId))
		case strings.HasPrefix(line, "for ") || strings.HasPrefix(line, "for(") || strings.HasPrefix(line, "range "):
			nodes = append(nodes, fmt.Sprintf("    %s[%s]", nodeId, truncate(line, 30)))
			edges = append(edges, fmt.Sprintf("    %s -->|loop| %s", nodeId, nodeId))
		case strings.HasPrefix(line, "return"):
			nodes = append(nodes, fmt.Sprintf("    %s([%s])", nodeId, truncate(line, 30)))
		default:
			nodes = append(nodes, fmt.Sprintf("    %s[%s]", nodeId, truncate(line, 30)))
		}

		if nodeCount > 1 {
			edges = append(edges, fmt.Sprintf("    N%d --> %s", nodeCount-1, nodeId))
		}
	}

	var result strings.Builder
	result.WriteString("```mermaid\n")
	result.WriteString("flowchart TD\n")

	if title != "" {
		result.WriteString(fmt.Sprintf("    subgraph %s\n", title))
	}

	for _, node := range nodes {
		result.WriteString(node + "\n")
	}

	for _, edge := range edges {
		result.WriteString(edge + "\n")
	}

	if title != "" {
		result.WriteString("    end\n")
	}

	result.WriteString("```")

	return result.String()
}

// 辅助函数：生成时序图
func generateSequence(code, title string) string {
	// 提取函数调用关系
	callPattern := regexp.MustCompile(`(\w+)\.(\w+)\(`)
	matches := callPattern.FindAllStringSubmatch(code, -1)

	var participants []string
	calls := make(map[string]bool)

	for _, m := range matches {
		if len(m) >= 3 {
			participants = append(participants, m[1])
			call := fmt.Sprintf("    %s->>%s: %s()", m[1], m[1], m[2])
			calls[call] = true
		}
	}

	// 去重参与者
	participants = uniqueStrings(participants)

	var result strings.Builder
	result.WriteString("```mermaid\n")
	result.WriteString("sequenceDiagram\n")

	for _, p := range participants {
		result.WriteString(fmt.Sprintf("    participant %s\n", p))
	}

	for call := range calls {
		result.WriteString(call + "\n")
	}

	result.WriteString("```")

	return result.String()
}

// 辅助函数：生成类图
func generateClassDiagram(code, title string) string {
	// 识别结构体/类定义
	structPattern := regexp.MustCompile(`type\s+(\w+)\s+struct\s*{([^}]*)}`)
	matches := structPattern.FindAllStringSubmatch(code, -1)

	var result strings.Builder
	result.WriteString("```mermaid\n")
	result.WriteString("classDiagram\n")

	for _, m := range matches {
		if len(m) >= 3 {
			className := m[1]
			fields := m[2]

			result.WriteString(fmt.Sprintf("    class %s {\n", className))

			// 解析字段
			fieldLines := strings.Split(fields, "\n")
			for _, field := range fieldLines {
				field = strings.TrimSpace(field)
				if field != "" && !strings.HasPrefix(field, "//") {
					parts := strings.Fields(field)
					if len(parts) >= 2 {
						result.WriteString(fmt.Sprintf("        +%s %s\n", parts[0], parts[1]))
					}
				}
			}

			result.WriteString("    }\n")
		}
	}

	result.WriteString("```")

	return result.String()
}

// 辅助函数：生成架构图
func generateArchitectureDiagram(description string) string {
	// 从描述中提取组件
	componentPattern := regexp.MustCompile(`(\w+(?:\s+\w+){0,2})\s+(?:service|component|module|layer|database|cache|api|gateway)`)
	matches := componentPattern.FindAllStringSubmatch(strings.ToLower(description), -1)

	var components []string
	for _, m := range matches {
		if len(m) >= 2 {
			components = append(components, strings.ReplaceAll(m[1], " ", "_"))
		}
	}

	// 添加一些常见组件
	if len(components) == 0 {
		components = []string{"client", "api_gateway", "service", "database"}
	}

	var result strings.Builder
	result.WriteString("```mermaid\n")
	result.WriteString("graph TB\n")

	// 添加节点
	for _, c := range components {
		result.WriteString(fmt.Sprintf("    %s[%s]\n", c, strings.ReplaceAll(c, "_", " ")))
	}

	// 添加连接（简单的链式连接）
	for i := 0; i < len(components)-1; i++ {
		result.WriteString(fmt.Sprintf("    %s --> %s\n", components[i], components[i+1]))
	}

	result.WriteString("```")

	return result.String()
}

// 辅助函数：提取摘要
func extractSummary(text string, maxLength int, style string) string {
	// 分句
	sentences := splitSentences(text)

	switch style {
	case "bullet_points":
		// 提取关键句作为要点
		var points []string
		for _, s := range sentences {
			if len(s) > 20 && len(points) < 5 {
				points = append(points, "- "+s)
			}
		}
		return strings.Join(points, "\n")

	case "detailed":
		// 连接前几个句子
		var result []string
		length := 0
		for _, s := range sentences {
			if length+len(s) <= maxLength {
				result = append(result, s)
				length += len(s)
			} else {
				break
			}
		}
		return strings.Join(result, " ")

	default: // concise
		// 只返回第一句
		if len(sentences) > 0 {
			summary := sentences[0]
			if len(summary) > maxLength {
				summary = summary[:maxLength] + "..."
			}
			return summary
		}
	}

	return text
}

// 辅助函数：分句
func splitSentences(text string) []string {
	// 简单分句
	re := regexp.MustCompile(`[.!?]+\s+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 10 {
			result = append(result, s)
		}
	}

	return result
}

// 辅助函数：截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// 辅助函数：字符串去重
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
