package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ListDirArgs struct {
	Dir           string `json:"dir"`
	Recursive     bool   `json:"recursive,omitempty"`
	Pattern       string `json:"pattern,omitempty"`
	IncludeConfig bool   `json:"include_config,omitempty"`
}

type ListDirEntry struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Size     int64     `json:"size,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

type ignorePattern struct {
	raw      string
	negate   bool
	onlyDir  bool
	anchored bool
	hasSlash bool
	regex    *regexp.Regexp
}

func ListDir(args json.RawMessage, basePath string) (string, error) {
	var params ListDirArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Dir == "" {
		params.Dir = "."
	}

	fullPath := filepath.Join(basePath, params.Dir)
	if strings.HasPrefix(params.Dir, "/") {
		fullPath = params.Dir
	}
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("dir escapes base directory: %s", params.Dir)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", params.Dir)
		}
		return "", fmt.Errorf("cannot access directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", params.Dir)
	}

	var entries []ListDirEntry

	ignoredNames := map[string]bool{
		".git":         true,
		".idea":        true,
		".vscode":      true,
		".DS_Store":    true,
		"node_modules": true,
		"dist":         true,
		"build":        true,
		"vendor":       true,
	}
	ignorePatterns := loadIgnorePatterns(basePath)

	if params.Recursive {
		err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if path == fullPath {
				return nil
			}

			relToRoot, relErr := filepath.Rel(basePath, path)
			if relErr != nil {
				relToRoot = d.Name()
			}
			if shouldIgnorePath(relToRoot, d.Name(), d.IsDir(), params.IncludeConfig, ignoredNames, ignorePatterns) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			relPath, _ := filepath.Rel(fullPath, path)

			if params.Pattern != "" {
				matched, _ := filepath.Match(params.Pattern, d.Name())
				if !matched && !d.IsDir() {
					return nil
				}
			}

			entry := ListDirEntry{
				Name: relPath,
				Type: "file",
			}
			if d.IsDir() {
				entry.Type = "directory"
			}

			if info, err := d.Info(); err == nil {
				entry.Size = info.Size()
				entry.Modified = info.ModTime()
			}

			entries = append(entries, entry)
			return nil
		})
	} else {
		items, err := os.ReadDir(fullPath)
		if err != nil {
			return "", fmt.Errorf("cannot read directory: %w", err)
		}

		for _, item := range items {
			entryPath := filepath.Join(fullPath, item.Name())
			relToRoot, relErr := filepath.Rel(basePath, entryPath)
			if relErr != nil {
				relToRoot = item.Name()
			}
			if shouldIgnorePath(relToRoot, item.Name(), item.IsDir(), params.IncludeConfig, ignoredNames, ignorePatterns) {
				continue
			}

			if params.Pattern != "" {
				matched, _ := filepath.Match(params.Pattern, item.Name())
				if !matched && !item.IsDir() {
					continue
				}
			}

			entry := ListDirEntry{
				Name: item.Name(),
				Type: "file",
			}
			if item.IsDir() {
				entry.Type = "directory"
			}

			if info, err := item.Info(); err == nil {
				entry.Size = info.Size()
				entry.Modified = info.ModTime()
			}

			entries = append(entries, entry)
		}
	}

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	var lines []string
	for _, e := range entries {
		typeStr := "F"
		if e.Type == "directory" {
			typeStr = "D"
		}
		lines = append(lines, fmt.Sprintf("[%s] %-50s %10d %s",
			typeStr, e.Name, e.Size, e.Modified.Format("2006-01-02 15:04")))
	}

	return strings.Join(lines, "\n"), nil
}

// loadIgnorePatterns 读取 .gitignore 与 .dockerignore 规则并生成忽略模式列表。
func loadIgnorePatterns(basePath string) []ignorePattern {
	ignoreFiles := []string{
		filepath.Join(basePath, ".gitignore"),
		filepath.Join(basePath, ".dockerignore"),
	}
	var patterns []ignorePattern
	for _, ignoreFile := range ignoreFiles {
		if !FileExists(ignoreFile) {
			continue
		}
		content, err := os.ReadFile(ignoreFile)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			pattern, ok := parseIgnoreLine(line)
			if ok {
				patterns = append(patterns, pattern)
			}
		}
	}
	return patterns
}

// parseIgnoreLine 解析单行 ignore 规则并返回模式信息。
func parseIgnoreLine(line string) (ignorePattern, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ignorePattern{}, false
	}
	pattern := ignorePattern{
		raw: trimmed,
	}
	if strings.HasPrefix(trimmed, "!") {
		pattern.negate = true
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "!"))
	}
	if strings.HasSuffix(trimmed, "/") {
		pattern.onlyDir = true
		trimmed = strings.TrimSuffix(trimmed, "/")
	}
	if strings.HasPrefix(trimmed, "/") {
		pattern.anchored = true
		trimmed = strings.TrimPrefix(trimmed, "/")
	}
	trimmed = filepath.ToSlash(trimmed)
	if trimmed == "" {
		return ignorePattern{}, false
	}
	pattern.hasSlash = strings.Contains(trimmed, "/")
	regex, err := buildIgnoreRegex(trimmed)
	if err != nil {
		return ignorePattern{}, false
	}
	pattern.regex = regex
	return pattern, true
}

// buildIgnoreRegex 将 ignore 模式转换为正则表达式。
func buildIgnoreRegex(pattern string) (*regexp.Regexp, error) {
	var builder strings.Builder
	builder.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		char := pattern[i]
		if char == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				builder.WriteString(".*")
				i++
				continue
			}
			builder.WriteString(`[^/]*`)
			continue
		}
		if char == '?' {
			builder.WriteString(`[^/]`)
			continue
		}
		if strings.ContainsRune(`.+()|[]{}^$\\`, rune(char)) {
			builder.WriteByte('\\')
		}
		builder.WriteByte(char)
	}
	builder.WriteString("$")
	return regexp.Compile(builder.String())
}

// matchIgnorePattern 判断路径是否匹配单条 ignore 规则。
func matchIgnorePattern(relPath string, isDir bool, pattern ignorePattern) bool {
	if pattern.onlyDir && !isDir {
		return false
	}
	normalized := path.Clean(filepath.ToSlash(relPath))
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "." || normalized == "" {
		return false
	}
	if pattern.hasSlash {
		if pattern.anchored {
			return pattern.regex.MatchString(normalized)
		}
		parts := strings.Split(normalized, "/")
		for i := 0; i < len(parts); i++ {
			if pattern.regex.MatchString(strings.Join(parts[i:], "/")) {
				return true
			}
		}
		return false
	}
	return pattern.regex.MatchString(path.Base(normalized))
}

// shouldIgnorePath 综合默认忽略规则与 ignore 文件规则判断是否需要跳过。
func shouldIgnorePath(relPath, name string, isDir bool, includeConfig bool, ignoredNames map[string]bool, patterns []ignorePattern) bool {
	if !includeConfig && ignoredNames[name] {
		return true
	}
	ignored := false
	for _, pattern := range patterns {
		if matchIgnorePattern(relPath, isDir, pattern) {
			ignored = !pattern.negate
		}
	}
	return ignored
}
