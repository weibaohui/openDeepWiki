package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestGlobSearchWithoutDoubleStar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "globsearch_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.log"), []byte("test"), 0644)

	results, err := globSearch(tmpDir, "*.txt")
	if err != nil {
		t.Fatalf("globSearch failed: %v", err)
	}

	expected := []string{"file1.txt", "file2.txt"}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

func TestGlobSearchWithDoubleStar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "globsearch_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.log"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("test"), 0644)

	results, err := globSearch(tmpDir, "**/*.txt")
	if err != nil {
		t.Fatalf("globSearch failed: %v", err)
	}

	sort.Strings(results)
	expected := []string{"file3.txt", "subdir/file1.txt"}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

func TestSearchFiles(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)

	args := SearchFilesArgs{
		Pattern: "*.txt",
		Path:    ".",
	}
	argsJSON, _ := json.Marshal(args)

	result, err := SearchFiles(argsJSON, tmpDir)
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}
