package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListDir_IgnoreConfig(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "listdir_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory
	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("foo"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create normal file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

    // Create .idea directory
    if err := os.Mkdir(filepath.Join(tmpDir, ".idea"), 0755); err != nil {
        t.Fatal(err)
    }

	// Test 1: Default behavior (should ignore .git and .idea)
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

	// Test 2: Include config (should show .git and .idea)
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

func TestListDir_Recursive_IgnoreConfig(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "listdir_recursive_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory and content
	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
    if err := os.Mkdir(filepath.Join(tmpDir, ".git", "refs"), 0755); err != nil {
        t.Fatal(err)
    }

	// Create normal dir and file
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test 1: Default behavior (should ignore .git completely)
	args := json.RawMessage(`{"dir": ".", "recursive": true}`)
	result, err := ListDir(args, tmpDir)
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}

	if strings.Contains(result, ".git") {
		t.Errorf("ListDir recursive should ignore .git directory, got: %s", result)
	}
    // ensure contents of .git are not listed
    // Note: refs might match "refs" string if it appears elsewhere, but in this isolated test, it's inside .git
    // However, filepath.Rel output for refs would be ".git/refs" or similar.
    if strings.Contains(result, "refs") {
        t.Errorf("ListDir recursive should not list contents of ignored directory, got: %s", result)
    }

	if !strings.Contains(result, "src/main.go") {
		t.Errorf("ListDir recursive should list src/main.go, got: %s", result)
	}
}
