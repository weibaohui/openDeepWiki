// Code Analysis Tools - 代码分析工具
// 对应 MCP tools: code.parse_ast, code.extract_functions, code.get_call_graph,
//                 code.calculate_complexity, code.get_file_tree, code.get_snippet,
//                 code.get_dependencies, code.find_definitions

package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractFunctionsArgs code.extract_functions 参数
type ExtractFunctionsArgs struct {
	FilePath    string `json:"file_path"`
	IncludeBody bool   `json:"include_body,omitempty"`
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name       string   `json:"name"`
	Signature  string   `json:"signature"`
	LineStart  int      `json:"line_start"`
	LineEnd    int      `json:"line_end"`
	Params     []string `json:"params,omitempty"`
	Returns    []string `json:"returns,omitempty"`
	Complexity int      `json:"complexity,omitempty"`
	Body       string   `json:"body,omitempty"`
}

// ExtractFunctions 提取文件中的函数列表
func ExtractFunctions(args json.RawMessage, basePath string) (string, error) {
	var params ExtractFunctionsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.FilePath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("file_path escapes base directory: %s", params.FilePath)
	}

	// 解析 Go 文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	var functions []FunctionInfo

	// 遍历 AST 查找函数
	ast.Inspect(f, func(n ast.Node) bool {
		var fn *ast.FuncDecl
		var ok bool
		if fn, ok = n.(*ast.FuncDecl); !ok {
			return true
		}

		info := FunctionInfo{
			Name: fn.Name.Name,
		}

		// 获取位置信息
		if fn.Pos().IsValid() && fn.End().IsValid() {
			pos := fset.Position(fn.Pos())
			end := fset.Position(fn.End())
			info.LineStart = pos.Line
			info.LineEnd = end.Line
		}

		// 构建签名
		var sigParts []string
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recv := fn.Recv.List[0]
			var recvName string
			if len(recv.Names) > 0 {
				recvName = recv.Names[0].Name
			}
			recvType := exprToString(recv.Type)
			sigParts = append(sigParts, fmt.Sprintf("(%s %s)", recvName, recvType))
		}
		sigParts = append(sigParts, fn.Name.Name)

		// 参数
		if fn.Type.Params != nil {
			var params []string
			for _, p := range fn.Type.Params.List {
				typ := exprToString(p.Type)
				for _, name := range p.Names {
					params = append(params, fmt.Sprintf("%s %s", name.Name, typ))
					info.Params = append(info.Params, fmt.Sprintf("%s:%s", name.Name, typ))
				}
			}
			sigParts = append(sigParts, fmt.Sprintf("(%s)", strings.Join(params, ", ")))
		} else {
			sigParts = append(sigParts, "()")
		}

		// 返回值
		if fn.Type.Results != nil {
			var results []string
			for _, r := range fn.Type.Results.List {
				typ := exprToString(r.Type)
				if len(r.Names) > 0 {
					for _, name := range r.Names {
						results = append(results, fmt.Sprintf("%s %s", name.Name, typ))
						info.Returns = append(info.Returns, fmt.Sprintf("%s:%s", name.Name, typ))
					}
				} else {
					results = append(results, typ)
					info.Returns = append(info.Returns, typ)
				}
			}
			sigParts = append(sigParts, fmt.Sprintf("(%s)", strings.Join(results, ", ")))
		}

		info.Signature = strings.Join(sigParts, " ")

		// 计算圈复杂度
		info.Complexity = calculateCyclomaticComplexity(fn)

		// 如果需要包含函数体
		if params.IncludeBody && fn.Body != nil {
			body, _ := getFileLines(fullPath, info.LineStart, info.LineEnd)
			info.Body = body
		}

		functions = append(functions, info)
		return true
	})

	if len(functions) == 0 {
		return "No functions found in file.", nil
	}

	// 格式化输出
	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d functions:\n", len(functions)))

	for _, fn := range functions {
		lines = append(lines, fmt.Sprintf("- %s", fn.Signature))
		lines = append(lines, fmt.Sprintf("  Lines: %d-%d, Complexity: %d", fn.LineStart, fn.LineEnd, fn.Complexity))
		if len(fn.Params) > 0 {
			lines = append(lines, fmt.Sprintf("  Params: %s", strings.Join(fn.Params, ", ")))
		}
		if len(fn.Returns) > 0 {
			lines = append(lines, fmt.Sprintf("  Returns: %s", strings.Join(fn.Returns, ", ")))
		}
		if fn.Body != "" {
			lines = append(lines, fmt.Sprintf("  Body:\n%s", fn.Body))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n"), nil
}

// GetCodeSnippetArgs code.get_snippet 参数
type GetCodeSnippetArgs struct {
	FilePath  string `json:"file_path"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Context   int    `json:"context,omitempty"`
}

// GetCodeSnippet 获取代码片段
func GetCodeSnippet(args json.RawMessage, basePath string) (string, error) {
	var params GetCodeSnippetArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if params.LineStart <= 0 {
		return "", fmt.Errorf("line_start must be positive")
	}
	if params.LineEnd < params.LineStart {
		return "", fmt.Errorf("line_end must be >= line_start")
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.FilePath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("file_path escapes base directory: %s", params.FilePath)
	}

	// 添加上下文
	start := params.LineStart
	end := params.LineEnd
	if params.Context > 0 {
		start -= params.Context
		if start < 1 {
			start = 1
		}
		end += params.Context
	}

	// 限制行数
	if end-start > 200 {
		end = start + 200
	}

	snippet, err := getFileLines(fullPath, start, end)
	if err != nil {
		return "", err
	}

	// 添加行号
	lines := strings.Split(snippet, "\n")
	var numbered []string
	for i, line := range lines {
		lineNum := start + i
		marker := "  "
		if lineNum >= params.LineStart && lineNum <= params.LineEnd {
			marker = "> "
		}
		numbered = append(numbered, fmt.Sprintf("%s%4d | %s", marker, lineNum, line))
	}

	return strings.Join(numbered, "\n"), nil
}

// GetFileTreeArgs code.get_file_tree 参数
type GetFileTreeArgs struct {
	RepoPath     string `json:"repo_path"`
	IncludeTests bool   `json:"include_tests,omitempty"`
}

// GetFileTree 获取代码文件树
func GetFileTree(args json.RawMessage, basePath string) (string, error) {
	var params GetFileTreeArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.RepoPath == "" {
		params.RepoPath = "."
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.RepoPath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
	}

	var files []string
	languageCount := make(map[string]int)
	totalFiles := 0

	extToLang := map[string]string{
		".go":   "Go",
		".py":   "Python",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".jsx":  "JavaScript",
		".tsx":  "TypeScript",
		".java": "Java",
		".rs":   "Rust",
		".cpp":  "C++",
		".c":    "C",
		".h":    "C/C++",
		".hpp":  "C++",
		".rb":   "Ruby",
		".php":  "PHP",
	}

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// 跳过隐藏目录和 vendor
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过测试文件（如果需要）
		if !params.IncludeTests {
			name := info.Name()
			if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".spec.js") {
				return nil
			}
		}

		relPath, _ := filepath.Rel(fullPath, path)
		files = append(files, relPath)
		totalFiles++

		// 统计语言
		ext := filepath.Ext(path)
		if lang, ok := extToLang[ext]; ok {
			languageCount[lang]++
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	// 格式化输出
	var lines []string
	lines = append(lines, fmt.Sprintf("Total files: %d\n", totalFiles))

	lines = append(lines, "Languages:")
	for lang, count := range languageCount {
		lines = append(lines, fmt.Sprintf("  %s: %d", lang, count))
	}

	lines = append(lines, "\nFile tree:")
	for _, f := range files {
		if len(lines) > 150 {
			lines = append(lines, fmt.Sprintf("... (%d more files)", len(files)-len(lines)+150))
			break
		}
		lines = append(lines, f)
	}

	return strings.Join(lines, "\n"), nil
}

// CalculateComplexityArgs code.calculate_complexity 参数
type CalculateComplexityArgs struct {
	FilePath     string `json:"file_path"`
	FunctionName string `json:"function_name,omitempty"`
}

// CalculateComplexity 计算代码圈复杂度
func CalculateComplexity(args json.RawMessage, basePath string) (string, error) {
	var params CalculateComplexityArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.FilePath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("file_path escapes base directory: %s", params.FilePath)
	}

	// 解析 Go 文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	var functions []FunctionInfo
	var totalComplexity int

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// 如果指定了函数名，只计算该函数
		if params.FunctionName != "" && fn.Name.Name != params.FunctionName {
			return true
		}

		complexity := calculateCyclomaticComplexity(fn)
		functions = append(functions, FunctionInfo{
			Name:       fn.Name.Name,
			Complexity: complexity,
		})
		totalComplexity += complexity

		return true
	})

	if len(functions) == 0 {
		if params.FunctionName != "" {
			return "", fmt.Errorf("function not found: %s", params.FunctionName)
		}
		return "No functions found in file.", nil
	}

	// 计算评分
	rating := "A"
	avgComplexity := totalComplexity / len(functions)
	switch {
	case avgComplexity > 20:
		rating = "F"
	case avgComplexity > 15:
		rating = "E"
	case avgComplexity > 10:
		rating = "D"
	case avgComplexity > 7:
		rating = "C"
	case avgComplexity > 4:
		rating = "B"
	}

	// 格式化输出
	var lines []string
	lines = append(lines, fmt.Sprintf("File: %s", params.FilePath))
	lines = append(lines, fmt.Sprintf("Rating: %s (avg complexity: %d)\n", rating, avgComplexity))
	lines = append(lines, "Functions:")

	for _, fn := range functions {
		indicator := "✓"
		if fn.Complexity > 10 {
			indicator = "⚠"
		}
		lines = append(lines, fmt.Sprintf("  %s %s: %d", indicator, fn.Name, fn.Complexity))
	}

	return strings.Join(lines, "\n"), nil
}

// FindDefinitionsArgs code.find_definitions 参数
type FindDefinitionsArgs struct {
	Symbol   string `json:"symbol"`
	RepoPath string `json:"repo_path"`
	Type     string `json:"type,omitempty"` // function, class, interface, variable
}

// FindDefinitions 查找符号定义
func FindDefinitions(args json.RawMessage, basePath string) (string, error) {
	var params FindDefinitionsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Symbol == "" {
		return "", fmt.Errorf("symbol is required")
	}
	if params.RepoPath == "" {
		params.RepoPath = "."
	}

	// 安全检查
	fullPath := params.RepoPath
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
	}

	var definitions []string

	// 遍历文件查找定义
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 只处理代码文件
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}

		// 简单的文本搜索
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// 简单匹配：func Symbol, type Symbol, var Symbol, const Symbol
			patterns := []string{
				fmt.Sprintf("func %s", params.Symbol),
				fmt.Sprintf("func (%s) ", params.Symbol),
				fmt.Sprintf("type %s ", params.Symbol),
				fmt.Sprintf("var %s ", params.Symbol),
				fmt.Sprintf("const %s ", params.Symbol),
			}

			for _, pattern := range patterns {
				if strings.Contains(line, pattern) {
					relPath, _ := filepath.Rel(basePath, path)
					definitions = append(definitions, fmt.Sprintf("%s:%d - %s", relPath, i+1, strings.TrimSpace(line)))
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching: %w", err)
	}

	if len(definitions) == 0 {
		return fmt.Sprintf("No definitions found for symbol: %s", params.Symbol), nil
	}

	return fmt.Sprintf("Found %d definitions for '%s':\n%s",
		len(definitions), params.Symbol, strings.Join(definitions, "\n")), nil
}

// 辅助函数：将表达式转换为字符串
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprToString(e.Len), exprToString(e.Elt))
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.FuncType:
		return "func"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprToString(e.Key), exprToString(e.Value))
	case *ast.ChanType:
		return "chan"
	case *ast.BasicLit:
		return e.Value
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// 辅助函数：计算圈复杂度
func calculateCyclomaticComplexity(fn *ast.FuncDecl) int {
	complexity := 1

	if fn.Body == nil {
		return complexity
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		}
		return true
	})

	return complexity
}

// 辅助函数：获取文件指定行范围
func getFileLines(filePath string, start, end int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	currentLine := 0

	for scanner.Scan() {
		currentLine++
		if currentLine < start {
			continue
		}
		if currentLine > end {
			break
		}
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// 添加必要的 import
var _ = regexp.MatchString // 用于避免未使用 import 的错误
