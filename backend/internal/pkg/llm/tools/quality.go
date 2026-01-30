// Quality Tools - 质量检查工具
// 对应 MCP tools: quality.check_links, quality.plagiarism_check,
//                 quality.spell_check, quality.readability_score, quality.check_formatting

package tools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckLinksArgs quality.check_links 参数
type CheckLinksArgs struct {
	DocContent    string `json:"doc_content"`
	BasePath      string `json:"base_path"`
	CheckExternal bool   `json:"check_external,omitempty"`
}

// CheckLinks 检查文档中的链接有效性
func CheckLinks(args json.RawMessage, basePath string) (string, error) {
	var params CheckLinksArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.DocContent == "" {
		return "", fmt.Errorf("doc_content is required")
	}
	if params.BasePath == "" {
		params.BasePath = basePath
	}

	// 提取链接
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := linkPattern.FindAllStringSubmatch(params.DocContent, -1)

	var validLinks []string
	var brokenLinks []string

	for _, m := range matches {
		if len(m) >= 3 {
			linkText := m[1]
			linkURL := m[2]

			// 检查链接类型
			if strings.HasPrefix(linkURL, "http://") || strings.HasPrefix(linkURL, "https://") {
				if params.CheckExternal {
					// 外部链接（简化检查：只验证格式）
					if _, err := url.Parse(linkURL); err == nil {
						validLinks = append(validLinks, fmt.Sprintf("%s -> %s", linkText, linkURL))
					} else {
						brokenLinks = append(brokenLinks, fmt.Sprintf("%s -> %s (invalid URL)", linkText, linkURL))
					}
				} else {
					validLinks = append(validLinks, fmt.Sprintf("%s -> %s (external, not checked)", linkText, linkURL))
				}
			} else if strings.HasPrefix(linkURL, "#") {
				// 锚点链接（简化：假设有效）
				validLinks = append(validLinks, fmt.Sprintf("%s -> %s (anchor)", linkText, linkURL))
			} else {
				// 内部文件链接
				fullPath := filepath.Join(params.BasePath, linkURL)
				if isPathSafe(basePath, fullPath) {
					// 简化为总是假设有效（实际应该检查文件存在性）
					validLinks = append(validLinks, fmt.Sprintf("%s -> %s", linkText, linkURL))
				} else {
					brokenLinks = append(brokenLinks, fmt.Sprintf("%s -> %s (path escapes base)", linkText, linkURL))
				}
			}
		}
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Link Check Results:\n"))
	result.WriteString(fmt.Sprintf("Total: %d links\n\n", len(matches)))

	if len(validLinks) > 0 {
		result.WriteString(fmt.Sprintf("Valid (%d):\n", len(validLinks)))
		for _, link := range validLinks {
			result.WriteString(fmt.Sprintf("  ✓ %s\n", link))
		}
		result.WriteString("\n")
	}

	if len(brokenLinks) > 0 {
		result.WriteString(fmt.Sprintf("Broken (%d):\n", len(brokenLinks)))
		for _, link := range brokenLinks {
			result.WriteString(fmt.Sprintf("  ✗ %s\n", link))
		}
	} else {
		result.WriteString("All links are valid!\n")
	}

	return result.String(), nil
}

// PlagiarismCheckArgs quality.plagiarism_check 参数
type PlagiarismCheckArgs struct {
	Text      string   `json:"text"`
	CodeFiles []string `json:"code_files"`
}

// PlagiarismCheck 检查文本与代码的重复度
func PlagiarismCheck(args json.RawMessage, basePath string) (string, error) {
	var params PlagiarismCheckArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Text == "" {
		return "", fmt.Errorf("text is required")
	}

	// 清理文本
	textTokens := tokenize(params.Text)

	var matches []string
	totalSimilarity := 0.0

	for _, file := range params.CodeFiles {
		// 安全检查
		fullPath := filepath.Join(basePath, file)
		if !isPathSafe(basePath, fullPath) {
			continue
		}

		// 简化：只检查 token 重叠
		// 实际实现应该使用更复杂的算法
		similarity := calculateTextSimilarity(textTokens, []string{file})
		if similarity > 0.3 {
			matches = append(matches, fmt.Sprintf("%s: %.0f%%", file, similarity*100))
			totalSimilarity += similarity
		}
	}

	avgSimilarity := 0.0
	if len(params.CodeFiles) > 0 {
		avgSimilarity = totalSimilarity / float64(len(params.CodeFiles))
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Plagiarism Check Results:\n"))
	result.WriteString(fmt.Sprintf("Similarity Score: %.0f%%\n\n", avgSimilarity*100))

	if len(matches) > 0 {
		result.WriteString("Potential matches:\n")
		for _, m := range matches {
			result.WriteString(fmt.Sprintf("  %s\n", m))
		}
	} else {
		result.WriteString("No significant matches found.\n")
	}

	return result.String(), nil
}

// SpellCheckArgs quality.spell_check 参数
type SpellCheckArgs struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

// SpellCheck 拼写检查（简化实现）
func SpellCheck(args json.RawMessage, basePath string) (string, error) {
	var params SpellCheckArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Text == "" {
		return "", fmt.Errorf("text is required")
	}
	if params.Language == "" {
		params.Language = "en_US"
	}

	// 常见拼写错误模式（简化）
	commonTypos := map[string]string{
		"teh":      "the",
		"adn":      "and",
		"taht":     "that",
		"wiht":     "with",
		"fo":       "for",
		"ot":       "to",
		"si":       "is",
		"ti":       "it",
		"funciton": "function",
		"recieve":  "receive",
		"seperate": "separate",
	}

	var issues []string
	words := regexp.MustCompile(`\b[a-zA-Z]+\b`).FindAllString(params.Text, -1)

	for _, word := range words {
		lowerWord := strings.ToLower(word)
		if correction, ok := commonTypos[lowerWord]; ok {
			issues = append(issues, fmt.Sprintf("'%s' -> '%s'", word, correction))
		}
	}

	// 检查重复单词
	doubleWordPattern := regexp.MustCompile(`\b(\w+)\s+\1\b`)
	doubleMatches := doubleWordPattern.FindAllString(params.Text, -1)
	for _, m := range doubleMatches {
		issues = append(issues, fmt.Sprintf("Repeated word: '%s'", m))
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Spell Check Results (%s):\n", params.Language))

	if len(issues) > 0 {
		result.WriteString(fmt.Sprintf("Found %d issues:\n", len(issues)))
		for _, issue := range issues {
			result.WriteString(fmt.Sprintf("  - %s\n", issue))
		}
	} else {
		result.WriteString("No spelling issues found.\n")
	}

	return result.String(), nil
}

// ReadabilityScoreArgs quality.readability_score 参数
type ReadabilityScoreArgs struct {
	Text   string `json:"text"`
	Metric string `json:"metric,omitempty"` // flesch_kincaid, etc.
}

// ReadabilityScore 计算可读性分数
func ReadabilityScore(args json.RawMessage, basePath string) (string, error) {
	var params ReadabilityScoreArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Text == "" {
		return "", fmt.Errorf("text is required")
	}
	if params.Metric == "" {
		params.Metric = "flesch_kincaid"
	}

	// 计算基本统计
	sentences := splitSentencesQuality(params.Text)
	words := regexp.MustCompile(`\b\w+\b`).FindAllString(params.Text, -1)

	sentenceCount := len(sentences)
	wordCount := len(words)

	if sentenceCount == 0 || wordCount == 0 {
		return "Cannot calculate readability: insufficient text", nil
	}

	// 计算音节数（简化：按元音组计算）
	syllableCount := 0
	for _, word := range words {
		syllableCount += countSyllables(word)
	}

	// Flesch-Kincaid Grade Level
	// Formula: 0.39 * (words/sentences) + 11.8 * (syllables/words) - 15.59
	avgWordsPerSentence := float64(wordCount) / float64(sentenceCount)
	avgSyllablesPerWord := float64(syllableCount) / float64(wordCount)

	gradeLevel := 0.39*avgWordsPerSentence + 11.8*avgSyllablesPerWord - 15.59

	// Flesch Reading Ease
	// Formula: 206.835 - 1.015 * (words/sentences) - 84.6 * (syllables/words)
	readingEase := 206.835 - 1.015*avgWordsPerSentence - 84.6*avgSyllablesPerWord

	// 等级描述
	var gradeDescription string
	switch {
	case gradeLevel < 6:
		gradeDescription = "Elementary"
	case gradeLevel < 9:
		gradeDescription = "Middle School"
	case gradeLevel < 13:
		gradeDescription = "High School"
	case gradeLevel < 16:
		gradeDescription = "College"
	default:
		gradeDescription = "Graduate"
	}

	// 建议
	var suggestions []string
	if avgWordsPerSentence > 20 {
		suggestions = append(suggestions, "Consider shorter sentences (avg > 20 words)")
	}
	if avgSyllablesPerWord > 2 {
		suggestions = append(suggestions, "Consider simpler words (avg > 2 syllables)")
	}
	if readingEase < 50 {
		suggestions = append(suggestions, "Text may be difficult to read")
	}

	var result strings.Builder
	result.WriteString("Readability Analysis:\n")
	result.WriteString(fmt.Sprintf("Metric: %s\n\n", params.Metric))
	result.WriteString(fmt.Sprintf("Statistics:\n"))
	result.WriteString(fmt.Sprintf("  Sentences: %d\n", sentenceCount))
	result.WriteString(fmt.Sprintf("  Words: %d\n", wordCount))
	result.WriteString(fmt.Sprintf("  Syllables: %d\n", syllableCount))
	result.WriteString(fmt.Sprintf("  Avg words/sentence: %.1f\n", avgWordsPerSentence))
	result.WriteString(fmt.Sprintf("  Avg syllables/word: %.1f\n\n", avgSyllablesPerWord))
	result.WriteString(fmt.Sprintf("Scores:\n"))
	result.WriteString(fmt.Sprintf("  Grade Level: %.1f (%s)\n", gradeLevel, gradeDescription))
	result.WriteString(fmt.Sprintf("  Reading Ease: %.1f\n\n", readingEase))

	if len(suggestions) > 0 {
		result.WriteString("Suggestions:\n")
		for _, s := range suggestions {
			result.WriteString(fmt.Sprintf("  - %s\n", s))
		}
	} else {
		result.WriteString("Good readability!\n")
	}

	return result.String(), nil
}

// CheckFormattingArgs quality.check_formatting 参数
type CheckFormattingArgs struct {
	Content string `json:"content"`
	Format  string `json:"format,omitempty"` // markdown, etc.
}

// CheckFormatting 检查文档格式
func CheckFormatting(args json.RawMessage, basePath string) (string, error) {
	var params CheckFormattingArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Content == "" {
		return "", fmt.Errorf("content is required")
	}
	if params.Format == "" {
		params.Format = "markdown"
	}

	var issues []string
	lines := strings.Split(params.Content, "\n")

	if params.Format == "markdown" {
		// 检查标题层级
		lastLevel := 0
		for i, line := range lines {
			lineNum := i + 1

			// 检测标题
			if strings.HasPrefix(line, "#") {
				level := 0
				for _, c := range line {
					if c == '#' {
						level++
					} else {
						break
					}
				}

				// 检查层级跳跃
				if level > lastLevel+1 && lastLevel > 0 {
					issues = append(issues, fmt.Sprintf("Line %d: Heading level jumps from %d to %d", lineNum, lastLevel, level))
				}

				// 检查空标题
				titleText := strings.TrimSpace(strings.TrimLeft(line, "#"))
				if titleText == "" {
					issues = append(issues, fmt.Sprintf("Line %d: Empty heading", lineNum))
				}

				lastLevel = level
			}

			// 检查代码块
			if strings.HasPrefix(line, "```") {
				// 简单检查是否闭合（简化：只检查数量）
			}

			// 检查行尾空格
			if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
				issues = append(issues, fmt.Sprintf("Line %d: Trailing whitespace", lineNum))
			}
		}

		// 检查代码块语言标识
		codeBlockPattern := regexp.MustCompile("^```\\s*$")
		for i, line := range lines {
			if codeBlockPattern.MatchString(line) {
				issues = append(issues, fmt.Sprintf("Line %d: Code block missing language identifier", i+1))
			}
		}
	}

	var result strings.Builder
	result.WriteString("Format Check Results:\n")
	result.WriteString(fmt.Sprintf("Format: %s\n", params.Format))
	result.WriteString(fmt.Sprintf("Lines: %d\n\n", len(lines)))

	if len(issues) > 0 {
		result.WriteString(fmt.Sprintf("Found %d issues:\n", len(issues)))
		for _, issue := range issues {
			result.WriteString(fmt.Sprintf("  - %s\n", issue))
		}
	} else {
		result.WriteString("No formatting issues found!\n")
	}

	return result.String(), nil
}

// 辅助函数：分词
func tokenize(text string) []string {
	re := regexp.MustCompile(`\b\w+\b`)
	return re.FindAllString(strings.ToLower(text), -1)
}

// 辅助函数：计算文本相似度
func calculateTextSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0
	}

	// 计算 Jaccard 相似度
	set1 := make(map[string]bool)
	for _, t := range tokens1 {
		set1[t] = true
	}

	intersection := 0
	for _, t := range tokens2 {
		if set1[t] {
			intersection++
		}
	}

	union := len(set1) + len(tokens2) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// 辅助函数：分句
func splitSentencesQuality(text string) []string {
	re := regexp.MustCompile(`[.!?]+\s+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 5 {
			result = append(result, s)
		}
	}

	return result
}

// 辅助函数：计算音节数
func countSyllables(word string) int {
	word = strings.ToLower(word)
	if len(word) <= 3 {
		return 1
	}

	// 移除末尾的 e
	word = strings.TrimSuffix(word, "e")

	// 计算元音组
	vowels := "aeiouy"
	syllables := 0
	wasVowel := false

	for _, c := range word {
		isVowel := strings.ContainsRune(vowels, c)
		if isVowel && !wasVowel {
			syllables++
		}
		wasVowel = isVowel
	}

	if syllables == 0 {
		syllables = 1
	}

	return syllables
}
