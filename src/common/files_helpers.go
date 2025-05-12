// common/files_helpers.go

package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo provides extended information about a file
type FileInfo struct {
	Path      string
	Name      string
	Extension string
	Directory string
	Size      int64
	ModTime   time.Time
	IsDir     bool
}

// NormalizePath provides normalized path
func NormalizePath(path string) string {
	path = strings.TrimSpace(path)
	// Return empty string if path is empty
	if path == "" {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(path))
}

// DirectoryExists checks if a directory exists
func DirectoryExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDirectoryExists ensures the specified directory exists
func EnsureDirectoryExists(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Check if directory already exists
	_, err := os.Stat(path)
	if err == nil {
		return nil // Directory exists
	}

	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", path, err)
		}
		return nil
	}

	return err // Some other error occurred
}

// ListFilesWithExtensions returns a list of files with the specified extensions
func ListFilesWithExtensions(dirPath string, extensions []string, recursive bool) ([]string, error) {
	if !DirectoryExists(dirPath) {
		return nil, fmt.Errorf("directory does not exist: %s", dirPath)
	}

	var result []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if path != dirPath && !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		for _, ext := range extensions {
			if strings.HasSuffix(strings.ToLower(path), strings.ToLower(ext)) {
				result = append(result, path)
				break
			}
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFn)
	if err != nil {
		return nil, fmt.Errorf("error listing files: %v", err)
	}

	return result, nil
}

// GetFileInfo returns extended information about a file
func GetFileInfo(filePath string) (FileInfo, error) {
	var fileInfo FileInfo

	info, err := os.Stat(filePath)
	if err != nil {
		return fileInfo, fmt.Errorf("failed to get file info: %v", err)
	}

	fileInfo.Path = filePath
	fileInfo.Name = info.Name()
	fileInfo.Extension = filepath.Ext(filePath)
	fileInfo.Directory = filepath.Dir(filePath)
	fileInfo.Size = info.Size()
	fileInfo.ModTime = info.ModTime()
	fileInfo.IsDir = info.IsDir()

	return fileInfo, nil
}

// CopyFile copies a file from source to destination
func CopyFile(sourcePath, destPath string) error {
	if !DirectoryExists(filepath.Dir(sourcePath)) {
		return fmt.Errorf("source directory does not exist: %s", filepath.Dir(sourcePath))
	}

	destDir := filepath.Dir(destPath)
	err := EnsureDirectoryExists(destDir)
	if err != nil {
		return fmt.Errorf("failed to ensure destination directory exists: %v", err)
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %v", sourcePath, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %v", destPath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	return nil
}

// MoveFile moves a file from source to destination
func MoveFile(sourcePath, destPath string) error {
	if !DirectoryExists(filepath.Dir(sourcePath)) {
		return fmt.Errorf("source directory does not exist: %s", filepath.Dir(sourcePath))
	}

	destDir := filepath.Dir(destPath)
	err := EnsureDirectoryExists(destDir)
	if err != nil {
		return fmt.Errorf("failed to ensure destination directory exists: %v", err)
	}

	err = os.Rename(sourcePath, destPath)
	if err != nil {
		err = CopyFile(sourcePath, destPath)
		if err != nil {
			return err
		}

		err = os.Remove(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to remove source file after copy: %v", err)
		}
	}

	return nil
}

// DeleteFile deletes a file
func DeleteFile(filePath string) error {
	if !DirectoryExists(filepath.Dir(filePath)) {
		return nil
	}

	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %v", filePath, err)
	}

	return nil
}

// JoinPaths joins path elements into a single path
func JoinPaths(elements ...string) string {
	return filepath.Join(elements...)
}

// ToDbPath converts a filesystem path to a format suitable for Rekordbox database queries
// It ensures paths use forward slashes and adds a trailing slash if needed for LIKE queries
func ToDbPath(path string, addTrailingSlash bool) string {
	// Convert to forward slashes for database consistency
	path = filepath.ToSlash(path)

	// Add trailing slash for LIKE queries if requested
	if addTrailingSlash && !strings.HasSuffix(path, "/") {
		path += "/"
	}

	return path
}
