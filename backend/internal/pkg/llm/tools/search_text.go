package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchTextArgs search_text 工具参数
type SearchTextArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Glob    string `json:"glob,omitempty"`
}

// SearchResult 单个搜索结果
type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// SearchText 在文件中搜索文本
func SearchText(args json.RawMessage, basePath string) (string, error) {
	var params SearchTextArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// 检查路径参数是否为绝对路径
	if filepath.IsAbs(params.Path) {
		return "", fmt.Errorf("absolute paths not allowed: %s", params.Path)
	}

	// 编译正则表达式
	re, err := regexp.Compile(params.Pattern)
	if err != nil {
		// 如果不是有效的正则，作为普通字符串搜索
		re, _ = regexp.Compile(regexp.QuoteMeta(params.Pattern))
	}

	// 搜索路径
	searchPath := basePath
	if params.Path != "" {
		searchPath = filepath.Join(basePath, params.Path)
	}

	// 安全检查
	if !isPathSafe(basePath, searchPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	// 执行搜索
	results, err := searchInDir(searchPath, re, params.Glob)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	// 限制结果数量
	const maxResults = 50
	truncated := false
	if len(results) > maxResults {
		results = results[:maxResults]
		truncated = true
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	// 格式化输出
	var output strings.Builder
	for _, r := range results {
		output.WriteString(fmt.Sprintf("%s:%d: %s\n", r.Path, r.Line, truncate(r.Content, 100)))
	}

	if truncated {
		output.WriteString(fmt.Sprintf("\n... (%d more matches truncated)", len(results)-maxResults))
	}

	return output.String(), nil
}

// searchInDir 在目录中递归搜索
func searchInDir(root string, re *regexp.Regexp, glob string) ([]SearchResult, error) {
	var results []string
	var searchResults []SearchResult

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		// 应用 glob 过滤
		if glob != "" {
			matched, _ := filepath.Match(glob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		// 跳过二进制文件和过大的文件
		info, err := d.Info()
		if err != nil || info.Size() > 1024*1024 {
			return nil
		}

		// 搜索文件内容
		fileResults := searchInFile(path, relPath, re)
		searchResults = append(searchResults, fileResults...)

		// 限制总结果数
		if len(searchResults) > 100 {
			return fmt.Errorf("too many matches")
		}

		return nil
	})

	_ = results // 避免未使用警告
	return searchResults, err
}

// searchInFile 在单个文件中搜索
func searchInFile(fullPath, relPath string, re *regexp.Regexp) []SearchResult {
	var results []SearchResult

	file, err := os.Open(fullPath)
	if err != nil {
		return results
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			results = append(results, SearchResult{
				Path:    relPath,
				Line:    lineNum,
				Content: line,
			})
		}
	}

	return results
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
