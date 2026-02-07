package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListDir_IgnoreConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("foo"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, ".idea"), 0755); err != nil {
		t.Fatal(err)
	}

	args := json.RawMessage(`{"dir": "."}`)
	result, err := ListDir(args, tmpDir)
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}

	if strings.Contains(result, ".git") {
		t.Errorf("ListDir should ignore .git by default, got: %s", result)
	}
	if strings.Contains(result, ".idea") {
		t.Errorf("ListDir should ignore .idea by default, got: %s", result)
	}
	if !strings.Contains(result, "main.go") {
		t.Errorf("ListDir should list main.go, got: %s", result)
	}

	args = json.RawMessage(`{"dir": ".", "include_config": true}`)
	result, err = ListDir(args, tmpDir)
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}

	if !strings.Contains(result, ".git") {
		t.Errorf("ListDir should show .git when include_config is true, got: %s", result)
	}
	if !strings.Contains(result, ".idea") {
		t.Errorf("ListDir should show .idea when include_config is true, got: %s", result)
	}
}
