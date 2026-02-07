package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileLineCount(t *testing.T) {
	tempDir := t.TempDir()

	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, "Line "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")
	testFile := filepath.Join(tempDir, "lines.txt")
	os.WriteFile(testFile, []byte(content), 0644)

	tests := []struct {
		name     string
		offset   int
		limit    int
		expected int
	}{
		{
			name:     "default limit (100)",
			offset:   1,
			limit:    0,
			expected: 101,
		},
		{
			name:     "custom limit 50",
			offset:   1,
			limit:    50,
			expected: 51,
		},
		{
			name:     "max limit 500",
			offset:   1,
			limit:    1000,
			expected: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := ReadFileArgs{
				Path:   "lines.txt",
				Offset: tt.offset,
				Limit:  tt.limit,
			}
			argsJSON, _ := json.Marshal(args)
			result, err := ReadFile(argsJSON, tempDir)
			if err != nil {
				t.Fatalf("ReadFile() unexpected error: %v", err)
			}

			resultLines := strings.Split(strings.TrimSpace(result), "\n")
			if len(resultLines) != tt.expected {
				t.Errorf("ReadFile() expected %d lines, got %d", tt.expected, len(resultLines))
			}
		})
	}
}
