package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanner(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glowy-discovery-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create directories
	_ = os.MkdirAll(filepath.Join(tempDir, "docs"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, ".hidden"), 0755)

	// Create files
	fileA := filepath.Join(tempDir, "a.md")
	fileB := filepath.Join(tempDir, "docs", "b.markdown")
	fileIgnoredGit := filepath.Join(tempDir, ".git", "hidden.md")
	fileIgnoredNode := filepath.Join(tempDir, "node_modules", "package.md")
	fileGitIgnoreTarget := filepath.Join(tempDir, "ignored_by_git.md")

	_ = os.WriteFile(fileA, []byte("# A"), 0644)
	time.Sleep(10 * time.Millisecond)
	_ = os.WriteFile(fileB, []byte("# B"), 0644)
	_ = os.WriteFile(fileIgnoredGit, []byte("# Git"), 0644)
	_ = os.WriteFile(fileIgnoredNode, []byte("# Node"), 0644)
	_ = os.WriteFile(fileGitIgnoreTarget, []byte("# Gitignore me"), 0644)

	// Write .gitignore
	_ = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("ignored_by_git.md\n"), 0644)

	// Run scanner
	files, err := ScanDirectory(tempDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected exactly 2 files (a.md and docs/b.markdown), got %d files: %+v", len(files), files)
	}

	// Verify paths
	foundA := false
	foundB := false
	for _, f := range files {
		if f.Name == "a.md" {
			foundA = true
		}
		if f.Name == "b.markdown" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("missing expected files: foundA=%v, foundB=%v", foundA, foundB)
	}

	// Test SortFiles
	SortFiles(files, "name")
	if files[0].Name != "a.md" || files[1].Name != "b.markdown" {
		t.Fatalf("sort by name failed: %+v", files)
	}

	// Test Single File Bypass on an ignored file
	bypassed, err := ScanSingleFile(fileIgnoredGit)
	if err != nil {
		t.Fatalf("ScanSingleFile bypass failed: %v", err)
	}
	if bypassed.Name != "hidden.md" {
		t.Fatalf("expected hidden.md, got %s", bypassed.Name)
	}

	// Test GroupByDirectory
	groups := GroupByDirectory(files)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (root and docs), got %d", len(groups))
	}
}
