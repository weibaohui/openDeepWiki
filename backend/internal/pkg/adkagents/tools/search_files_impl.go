package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

type SearchFilesArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func SearchFiles(args json.RawMessage, basePath string) (string, error) {
	var params SearchFilesArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	klog.V(6).Infof("搜索basePath=%s, path=%s, pattern=%s", basePath, params.Path, params.Pattern)

	fullPath := filepath.Join(basePath, params.Path)
	if strings.HasPrefix(params.Path, "/") {
		fullPath = params.Path
	}

	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	klog.V(6).Infof("最终执行搜索路径为:[%s]", fullPath)
	files, err := globSearch(fullPath, params.Pattern)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	const maxResults = 100
	if len(files) > maxResults {
		files = files[:maxResults]
		files = append(files, fmt.Sprintf("... (%d more results truncated)", len(files)-maxResults))
	}

	if len(files) == 0 {
		return fmt.Sprintf("No files found matching the pattern in %s.", params.Path), nil
	}

	return strings.Join(files, "\n"), nil
}

func globSearch(root, pattern string) ([]string, error) {
	var results []string

	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			rel, _ := filepath.Rel(root, m)
			results = append(results, rel)
		}
		return results, nil
	}

	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid pattern with multiple **: %s", pattern)
	}

	prefix := strings.TrimPrefix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		if prefix != "" && !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		if suffix != "" {
			matched, err := filepath.Match(suffix, filepath.Base(relPath))
			if err != nil || !matched {
				return nil
			}
		}

		results = append(results, relPath)
		return nil
	})

	return results, err
}
