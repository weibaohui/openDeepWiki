// Search Tools - 搜索工具
// 对应 MCP tools: search.semantic, search.symbol, search.similar_code, search.full_text
// 扩展现有功能: search_files, search_text (已有)

package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SemanticSearchArgs search.semantic 参数
type SemanticSearchArgs struct {
	Query     string   `json:"query"`
	RepoPath  string   `json:"repo_path"`
	TopK      int      `json:"top_k,omitempty"`
	FileTypes []string `json:"file_types,omitempty"`
}

// SemanticSearch 语义搜索（基于关键词的简化实现）
func SemanticSearch(args json.RawMessage, basePath string) (string, error) {
	var params SemanticSearchArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if params.RepoPath == "" {
		params.RepoPath = "."
	}
	if params.TopK <= 0 {
		params.TopK = 10
	}
	if params.TopK > 20 {
		params.TopK = 20
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.RepoPath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
	}

	// 提取查询关键词
	keywords := extractKeywords(params.Query)

	// 文件类型过滤
	var extensions []string
	for _, ft := range params.FileTypes {
		extensions = append(extensions, "."+ft)
	}

	// 搜索结果
	type searchResult struct {
		file    string
		score   float64
		snippet string
	}
	var results []searchResult

	// 遍历文件
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && (info.Name() == ".git" || info.Name() == "vendor" || info.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}

		// 文件类型过滤
		if len(extensions) > 0 {
			ext := filepath.Ext(path)
			found := false
			for _, e := range extensions {
				if e == ext {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		// 只处理代码文件
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" && ext != ".java" && ext != ".rs" && ext != ".md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// 计算相关性分数
		score, snippet := calculateRelevance(string(content), keywords)
		if score > 0 {
			relPath, _ := filepath.Rel(basePath, path)
			results = append(results, searchResult{
				file:    relPath,
				score:   score,
				snippet: snippet,
			})
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching: %w", err)
	}

	// 按分数排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// 限制结果数量
	if len(results) > params.TopK {
		results = results[:params.TopK]
	}

	if len(results) == 0 {
		return "No results found for the query.", nil
	}

	// 格式化输出
	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d results for: '%s'\n", len(results), params.Query))

	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. [%0.2f] %s", i+1, r.score, r.file))
		if r.snippet != "" {
			lines = append(lines, fmt.Sprintf("   %s", r.snippet))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n"), nil
}

// SymbolSearchArgs search.symbol 参数
type SymbolSearchArgs struct {
	SymbolName string `json:"symbol_name"`
	RepoPath   string `json:"repo_path"`
	SymbolType string `json:"symbol_type,omitempty"` // function, class, interface, variable
}

// SymbolSearch 精确符号搜索
func SymbolSearch(args json.RawMessage, basePath string) (string, error) {
	var params SymbolSearchArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.SymbolName == "" {
		return "", fmt.Errorf("symbol_name is required")
	}
	if params.RepoPath == "" {
		params.RepoPath = "."
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.RepoPath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
	}

	// 构建搜索模式
	var patterns []*regexp.Regexp
	
	switch params.SymbolType {
	case "function":
		patterns = append(patterns, regexp.MustCompile(fmt.Sprintf(`\bfunc\s+(\([^)]*\)\s+)?%s\b`, regexp.QuoteMeta(params.SymbolName))))
	case "class", "interface":
		patterns = append(patterns, regexp.MustCompile(fmt.Sprintf(`\btype\s+%s\s+(struct|interface)`, regexp.QuoteMeta(params.SymbolName))))
	case "variable":
		patterns = append(patterns, regexp.MustCompile(fmt.Sprintf(`\b(var|const)\s+%s\b`, regexp.QuoteMeta(params.SymbolName))))
	default:
		// 所有类型
		patterns = append(patterns, regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(params.SymbolName))))
	}

	var results []string

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && (info.Name() == ".git" || info.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}

		// 只处理代码文件
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			for _, pattern := range patterns {
				if pattern.MatchString(line) {
					relPath, _ := filepath.Rel(basePath, path)
					results = append(results, fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line)))
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("Symbol '%s' not found.", params.SymbolName), nil
	}

	return fmt.Sprintf("Found %d occurrences of '%s':\n%s", 
		len(results), params.SymbolName, strings.Join(results, "\n")), nil
}

// SimilarCodeArgs search.similar_code 参数
type SimilarCodeArgs struct {
	CodeSnippet string  `json:"code_snippet"`
	RepoPath    string  `json:"repo_path"`
	Threshold   float64 `json:"threshold,omitempty"`
}

// SimilarCode 查找相似代码
func SimilarCode(args json.RawMessage, basePath string) (string, error) {
	var params SimilarCodeArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.CodeSnippet == "" {
		return "", fmt.Errorf("code_snippet is required")
	}
	if params.RepoPath == "" {
		params.RepoPath = "."
	}
	if params.Threshold <= 0 {
		params.Threshold = 0.8
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.RepoPath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
	}

	// 提取查询代码的特征（简化：使用行和关键词）
	queryFeatures := extractCodeFeatures(params.CodeSnippet)

	type matchResult struct {
		file      string
		similarity float64
		snippet   string
	}
	var matches []matchResult

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		similarity, snippet := findSimilarSnippet(string(content), queryFeatures)
		if similarity >= params.Threshold {
			relPath, _ := filepath.Rel(basePath, path)
			matches = append(matches, matchResult{
				file:       relPath,
				similarity: similarity,
				snippet:    snippet,
			})
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching: %w", err)
	}

	// 按相似度排序
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].similarity > matches[i].similarity {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	if len(matches) == 0 {
		return "No similar code found.", nil
	}

	// 限制结果
	if len(matches) > 10 {
		matches = matches[:10]
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d similar code snippets:\n", len(matches)))

	for i, m := range matches {
		lines = append(lines, fmt.Sprintf("%d. [%0.0f%%] %s", i+1, m.similarity*100, m.file))
		if m.snippet != "" {
			lines = append(lines, fmt.Sprintf("   ```\n   %s\n   ```", m.snippet))
		}
	}

	return strings.Join(lines, "\n"), nil
}

// FullTextSearchArgs search.full_text 参数
type FullTextSearchArgs struct {
	Query         string `json:"query"`
	RepoPath      string `json:"repo_path"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

// FullTextSearch 全文搜索
func FullTextSearch(args json.RawMessage, basePath string) (string, error) {
	var params FullTextSearchArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if params.RepoPath == "" {
		params.RepoPath = "."
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.RepoPath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
	}

	pattern := params.Query
	if !params.CaseSensitive {
		pattern = strings.ToLower(pattern)
	}

	var results []string
	count := 0

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && (info.Name() == ".git" || info.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过二进制文件
		ext := filepath.Ext(path)
		if ext == ".exe" || ext == ".bin" || ext == ".o" {
			return nil
		}

		// 限制文件大小
		if info.Size() > 1024*1024 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		if !params.CaseSensitive {
			contentStr = strings.ToLower(contentStr)
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			checkLine := line
			if !params.CaseSensitive {
				checkLine = strings.ToLower(line)
			}
			
			if strings.Contains(checkLine, pattern) {
				relPath, _ := filepath.Rel(basePath, path)
				results = append(results, fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line)))
				count++
				if count >= 50 {
					return fmt.Errorf("max results")
				}
			}
		}

		return nil
	})

	if err != nil && err.Error() != "max results" {
		return "", fmt.Errorf("error searching: %w", err)
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	result := fmt.Sprintf("Found %d matches:\n%s", len(results), strings.Join(results, "\n"))
	if err != nil && err.Error() == "max results" {
		result += "\n... (results truncated)"
	}

	return result, nil
}

// 辅助函数：提取关键词
func extractKeywords(query string) []string {
	// 简单分词
	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true, 
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"of": true, "for": true, "with": true, "to": true, "from": true,
	}

	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool {
			return r == '.' || r == ',' || r == '?' || r == '!' || r == ';' || r == ':'
		})
		if !stopWords[w] && len(w) > 2 {
			keywords = append(keywords, w)
		}
	}

	return keywords
}

// 辅助函数：计算相关性
func calculateRelevance(content string, keywords []string) (float64, string) {
	if len(keywords) == 0 {
		return 0, ""
	}

	contentLower := strings.ToLower(content)
	matches := 0
	
	for _, kw := range keywords {
		if strings.Contains(contentLower, kw) {
			matches++
		}
	}

	score := float64(matches) / float64(len(keywords))
	
	// 提取片段
	var snippet string
	if score > 0 {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			lineLower := strings.ToLower(line)
			for _, kw := range keywords {
				if strings.Contains(lineLower, kw) {
					snippet = strings.TrimSpace(line)
					if len(snippet) > 100 {
						snippet = snippet[:100] + "..."
					}
					break
				}
			}
			if snippet != "" {
				break
			}
		}
	}

	return score, snippet
}

// 辅助函数：提取代码特征
func extractCodeFeatures(code string) []string {
	// 提取代码中的关键token
	re := regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
	tokens := re.FindAllString(code, -1)
	
	var features []string
	for _, t := range tokens {
		if len(t) > 2 && !isCommonKeyword(t) {
			features = append(features, strings.ToLower(t))
		}
	}
	
	return features
}

// 辅助函数：检查是否是常见关键字
func isCommonKeyword(word string) bool {
	keywords := map[string]bool{
		"if": true, "else": true, "for": true, "while": true, "return": true,
		"func": true, "function": true, "var": true, "let": true, "const": true,
		"import": true, "package": true, "type": true, "struct": true, "interface": true,
		"int": true, "string": true, "bool": true, "float": true, "true": true, "false": true,
	}
	return keywords[strings.ToLower(word)]
}

// 辅助函数：查找相似代码片段
func findSimilarSnippet(content string, queryFeatures []string) (float64, string) {
	if len(queryFeatures) == 0 {
		return 0, ""
	}

	lines := strings.Split(content, "\n")
	bestScore := 0.0
	bestSnippet := ""

	// 滑动窗口比较
	windowSize := 10
	for i := 0; i < len(lines)-windowSize; i++ {
		window := strings.Join(lines[i:i+windowSize], "\n")
		windowFeatures := extractCodeFeatures(window)
		
		// 计算Jaccard相似度
		common := 0
		for _, f1 := range queryFeatures {
			for _, f2 := range windowFeatures {
				if f1 == f2 {
					common++
					break
				}
			}
		}
		
		union := len(queryFeatures) + len(windowFeatures) - common
		if union == 0 {
			continue
		}
		
		similarity := float64(common) / float64(union)
		if similarity > bestScore {
			bestScore = similarity
			bestSnippet = window
		}
	}

	if len(bestSnippet) > 200 {
		bestSnippet = bestSnippet[:200] + "..."
	}

	return bestScore, bestSnippet
}
 