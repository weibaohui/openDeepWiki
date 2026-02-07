package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func ReadFile(args json.RawMessage, basePath string) (string, error) {
	var params ReadFileArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	fullPath := filepath.Join(basePath, params.Path)
	if strings.HasPrefix(params.Path, "/") {
		fullPath = params.Path
	}
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", params.Path)
		}
		return "", fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", params.Path)
	}

	const maxFileSize = 1024 * 1024
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file too large (max 1MB): %s (%d bytes)", params.Path, info.Size())
	}

	offset := params.Offset
	if offset < 1 {
		offset = 1
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	currentLine := 0

	for scanner.Scan() {
		currentLine++
		if currentLine < offset {
			continue
		}
		if currentLine >= offset+limit {
			lines = append(lines, fmt.Sprintf("... (%d more lines)", int(info.Size())/100))
			break
		}
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no content at offset %d", offset)
	}

	return strings.Join(lines, "\n"), nil
}
