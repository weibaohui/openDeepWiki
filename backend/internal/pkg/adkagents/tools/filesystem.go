package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
		".git":      true,
		".idea":     true,
		".vscode":   true,
		".DS_Store": true,
	}

	if params.Recursive {
		err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if path == fullPath {
				return nil
			}

			if !params.IncludeConfig && ignoredNames[d.Name()] {
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
			if !params.IncludeConfig && ignoredNames[item.Name()] {
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
