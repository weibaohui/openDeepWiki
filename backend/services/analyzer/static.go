package analyzer

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProjectInfo struct {
	Name          string         `json:"name"`
	Path          string         `json:"path"`
	Type          string         `json:"type"` // go, python, node, java, etc.
	Description   string         `json:"description"`
	Structure     *DirectoryTree `json:"structure"`
	Languages     map[string]int `json:"languages"`
	TotalFiles    int            `json:"total_files"`
	TotalLines    int            `json:"total_lines"`
	KeyFiles      []KeyFile      `json:"key_files"`
	Dependencies  []string       `json:"dependencies"`
	EntryPoints   []string       `json:"entry_points"`
	ConfigFiles   []string       `json:"config_files"`
	ReadmeContent string         `json:"readme_content"`
}

type DirectoryTree struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	IsDir    bool             `json:"is_dir"`
	Children []*DirectoryTree `json:"children,omitempty"`
}

type KeyFile struct {
	Path        string `json:"path"`
	Type        string `json:"type"` // entry, config, readme, main, etc.
	Description string `json:"description"`
	Preview     string `json:"preview"`
}

var languageExtensions = map[string]string{
	".go":     "Go",
	".py":     "Python",
	".js":     "JavaScript",
	".ts":     "TypeScript",
	".tsx":    "TypeScript",
	".jsx":    "JavaScript",
	".java":   "Java",
	".rs":     "Rust",
	".rb":     "Ruby",
	".php":    "PHP",
	".c":      "C",
	".cpp":    "C++",
	".h":      "C/C++ Header",
	".cs":     "C#",
	".swift":  "Swift",
	".kt":     "Kotlin",
	".scala":  "Scala",
	".vue":    "Vue",
	".svelte": "Svelte",
}

var ignoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".next":        true,
}

func Analyze(repoPath string) (*ProjectInfo, error) {
	info := &ProjectInfo{
		Name:      filepath.Base(repoPath),
		Path:      repoPath,
		Languages: make(map[string]int),
		KeyFiles:  []KeyFile{},
	}

	info.Structure = buildDirectoryTree(repoPath, repoPath, 3)

	filepath.Walk(repoPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fi.IsDir() {
			if ignoreDirs[fi.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		info.TotalFiles++

		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := languageExtensions[ext]; ok {
			info.Languages[lang]++
			lines := countLines(path)
			info.TotalLines += lines
		}

		relPath, _ := filepath.Rel(repoPath, path)
		detectKeyFile(info, relPath, path)

		return nil
	})

	info.Type = detectProjectType(info)
	info.ReadmeContent = readReadme(repoPath)
	info.Dependencies = detectDependencies(repoPath, info.Type)
	info.EntryPoints = detectEntryPoints(repoPath, info.Type)
	info.ConfigFiles = detectConfigFiles(repoPath)

	return info, nil
}

func buildDirectoryTree(basePath, currentPath string, maxDepth int) *DirectoryTree {
	if maxDepth <= 0 {
		return nil
	}

	info, err := os.Stat(currentPath)
	if err != nil {
		return nil
	}

	relPath, _ := filepath.Rel(basePath, currentPath)
	if relPath == "." {
		relPath = ""
	}

	tree := &DirectoryTree{
		Name:  info.Name(),
		Path:  relPath,
		IsDir: info.IsDir(),
	}

	if !info.IsDir() {
		return tree
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return tree
	}

	for _, entry := range entries {
		if ignoreDirs[entry.Name()] {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".env.example" {
			continue
		}

		childPath := filepath.Join(currentPath, entry.Name())
		child := buildDirectoryTree(basePath, childPath, maxDepth-1)
		if child != nil {
			tree.Children = append(tree.Children, child)
		}
	}

	return tree
}

func countLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}

func detectKeyFile(info *ProjectInfo, relPath, fullPath string) {
	name := strings.ToLower(filepath.Base(relPath))

	var keyFile *KeyFile

	switch {
	case name == "readme.md" || name == "readme.txt" || name == "readme":
		keyFile = &KeyFile{Path: relPath, Type: "readme", Description: "项目说明文档"}
	case name == "main.go" || name == "main.py" || name == "index.js" || name == "index.ts" || name == "app.py":
		keyFile = &KeyFile{Path: relPath, Type: "entry", Description: "入口文件"}
	case name == "go.mod":
		keyFile = &KeyFile{Path: relPath, Type: "config", Description: "Go 模块配置"}
	case name == "package.json":
		keyFile = &KeyFile{Path: relPath, Type: "config", Description: "Node.js 项目配置"}
	case name == "requirements.txt" || name == "pyproject.toml":
		keyFile = &KeyFile{Path: relPath, Type: "config", Description: "Python 依赖配置"}
	case name == "pom.xml":
		keyFile = &KeyFile{Path: relPath, Type: "config", Description: "Maven 项目配置"}
	case name == "build.gradle" || name == "build.gradle.kts":
		keyFile = &KeyFile{Path: relPath, Type: "config", Description: "Gradle 项目配置"}
	case name == "cargo.toml":
		keyFile = &KeyFile{Path: relPath, Type: "config", Description: "Rust 项目配置"}
	case name == "dockerfile" || name == "docker-compose.yml" || name == "docker-compose.yaml":
		keyFile = &KeyFile{Path: relPath, Type: "deploy", Description: "Docker 配置"}
	case name == "makefile":
		keyFile = &KeyFile{Path: relPath, Type: "build", Description: "构建脚本"}
	case strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml"):
		if strings.Contains(relPath, "config") || strings.Contains(relPath, "deploy") {
			keyFile = &KeyFile{Path: relPath, Type: "config", Description: "配置文件"}
		}
	}

	if keyFile != nil {
		keyFile.Preview = readFilePreview(fullPath, 20)
		info.KeyFiles = append(info.KeyFiles, *keyFile)
	}
}

func readFilePreview(path string, maxLines int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() && len(lines) < maxLines {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}

func detectProjectType(info *ProjectInfo) string {
	maxLang := ""
	maxCount := 0

	for lang, count := range info.Languages {
		if count > maxCount {
			maxCount = count
			maxLang = lang
		}
	}

	typeMap := map[string]string{
		"Go":         "go",
		"Python":     "python",
		"JavaScript": "node",
		"TypeScript": "node",
		"Java":       "java",
		"Rust":       "rust",
		"Ruby":       "ruby",
		"PHP":        "php",
	}

	if t, ok := typeMap[maxLang]; ok {
		return t
	}
	return "unknown"
}

func readReadme(repoPath string) string {
	readmeFiles := []string{"README.md", "readme.md", "README.txt", "README"}
	for _, name := range readmeFiles {
		path := filepath.Join(repoPath, name)
		content, err := os.ReadFile(path)
		if err == nil {
			if len(content) > 5000 {
				return string(content[:5000]) + "\n...(truncated)"
			}
			return string(content)
		}
	}
	return ""
}

func detectDependencies(repoPath, projectType string) []string {
	var deps []string

	switch projectType {
	case "go":
		content, err := os.ReadFile(filepath.Join(repoPath, "go.mod"))
		if err == nil {
			lines := strings.Split(string(content), "\n")
			inRequire := false
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "require (" {
					inRequire = true
					continue
				}
				if line == ")" {
					inRequire = false
					continue
				}
				if inRequire && line != "" && !strings.HasPrefix(line, "//") {
					parts := strings.Fields(line)
					if len(parts) >= 1 {
						deps = append(deps, parts[0])
					}
				}
			}
		}
	case "node":
		// Parse package.json dependencies
		content, err := os.ReadFile(filepath.Join(repoPath, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if err := parseJSON(content, &pkg); err == nil {
				if dependencies, ok := pkg["dependencies"].(map[string]interface{}); ok {
					for dep := range dependencies {
						deps = append(deps, dep)
					}
				}
			}
		}
	}

	sort.Strings(deps)
	if len(deps) > 20 {
		deps = deps[:20]
	}
	return deps
}

func detectEntryPoints(repoPath, projectType string) []string {
	var entries []string

	patterns := map[string][]string{
		"go":     {"main.go", "cmd/*/main.go"},
		"node":   {"index.js", "index.ts", "src/index.js", "src/index.ts", "app.js", "app.ts"},
		"python": {"main.py", "app.py", "__main__.py", "manage.py"},
		"java":   {"src/main/java/**/Main.java", "src/main/java/**/Application.java"},
	}

	if globs, ok := patterns[projectType]; ok {
		for _, pattern := range globs {
			matches, _ := filepath.Glob(filepath.Join(repoPath, pattern))
			for _, match := range matches {
				rel, _ := filepath.Rel(repoPath, match)
				entries = append(entries, rel)
			}
		}
	}

	return entries
}

func detectConfigFiles(repoPath string) []string {
	var configs []string
	configPatterns := []string{
		"*.yaml", "*.yml", "*.json", "*.toml", "*.ini", "*.conf",
		".env.example", "config/*", "configs/*",
	}

	for _, pattern := range configPatterns {
		matches, _ := filepath.Glob(filepath.Join(repoPath, pattern))
		for _, match := range matches {
			rel, _ := filepath.Rel(repoPath, match)
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() {
				configs = append(configs, rel)
			}
		}
	}

	if len(configs) > 15 {
		configs = configs[:15]
	}
	return configs
}

func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
