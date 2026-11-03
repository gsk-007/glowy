package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

// MarkdownFile represents a discovered Markdown document with its metadata.
type MarkdownFile struct {
	Path         string
	RelativePath string
	Name         string
	Directory    string
	ModifiedAt   time.Time
	CreatedAt    time.Time
	Size         int64
}

var defaultIgnoredDirs = map[string]bool{
	".git":                      true,
	"node_modules":              true,
	"vendor":                    true,
	"appdata":                   true,
	"$recycle.bin":              true,
	"system volume information": true,
}

// isIgnoredDir checks if a directory name should be skipped.
func isIgnoredDir(dirName string) bool {
	lower := strings.ToLower(dirName)
	if defaultIgnoredDirs[lower] {
		return true
	}
	// Skip hidden dot-directories
	if strings.HasPrefix(dirName, ".") && dirName != "." && dirName != ".." {
		return true
	}
	return false
}

// isMarkdownFile checks if a filename ends with .md or .markdown (case-insensitive).
func isMarkdownFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

// ScanSingleFile creates a MarkdownFile directly from a path, bypassing ignore rules.
func ScanSingleFile(targetPath string) (*MarkdownFile, error) {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		absPath = targetPath
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	modTime, createTime := GetFileTimes(absPath, info)
	dir := filepath.ToSlash(filepath.Dir(targetPath))
	if dir == "" {
		dir = "."
	}

	return &MarkdownFile{
		Path:         absPath,
		RelativePath: filepath.ToSlash(targetPath),
		Name:         filepath.Base(absPath),
		Directory:    dir,
		ModifiedAt:   modTime,
		CreatedAt:    createTime,
		Size:         info.Size(),
	}, nil
}

// ScanDirectory recursively discovers Markdown files in rootDir while respecting ignores.
func ScanDirectory(rootDir string) ([]MarkdownFile, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		absRoot = rootDir
	}

	var gitIgnore *ignore.GitIgnore
	gitignorePath := filepath.Join(absRoot, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		if gi, err := ignore.CompileIgnoreFile(gitignorePath); err == nil {
			gitIgnore = gi
		}
	}

	var results []MarkdownFile

	err = filepath.WalkDir(absRoot, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // Skip unreadable paths
		}

		name := d.Name()

		if d.IsDir() {
			if currentPath != absRoot && isIgnoredDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		if !isMarkdownFile(name) {
			return nil
		}

		// Calculate relative path for display and gitignore check
		relPath, err := filepath.Rel(absRoot, currentPath)
		if err != nil {
			relPath = name
		}
		relSlash := filepath.ToSlash(relPath)

		// Check gitignore
		if gitIgnore != nil && gitIgnore.MatchesPath(relSlash) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		modTime, createTime := GetFileTimes(currentPath, info)
		dir := filepath.ToSlash(filepath.Dir(relSlash))
		if dir == "" {
			dir = "."
		}

		results = append(results, MarkdownFile{
			Path:         currentPath,
			RelativePath: relSlash,
			Name:         name,
			Directory:    dir,
			ModifiedAt:   modTime,
			CreatedAt:    createTime,
			Size:         info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Default sort by ModifiedAt descending (newest first)
	SortFiles(results, "modified")

	return results, nil
}

// SortFiles sorts a slice of MarkdownFile in-place according to mode:
// "modified" (newest first), "name" (A-Z), "created" (newest first).
func SortFiles(files []MarkdownFile, mode string) {
	switch strings.ToLower(mode) {
	case "name":
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].RelativePath) < strings.ToLower(files[j].RelativePath)
		})
	case "created":
		sort.Slice(files, func(i, j int) bool {
			return files[i].CreatedAt.After(files[j].CreatedAt)
		})
	case "modified":
		fallthrough
	default:
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModifiedAt.After(files[j].ModifiedAt)
		})
	}
}

// GroupByDirectory organizes a slice of MarkdownFile by their directory.
func GroupByDirectory(files []MarkdownFile) map[string][]MarkdownFile {
	groups := make(map[string][]MarkdownFile)
	for _, f := range files {
		dir := f.Directory
		if dir == "" {
			dir = "."
		}
		groups[dir] = append(groups[dir], f)
	}
	return groups
}
