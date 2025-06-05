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
	// Return empty string if path is empty
	if IsEmptyString(path) {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
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
	if IsEmptyString(path) {
		return fmt.Errorf("path cannot be empty")
	}

	// Check if directory already exists
	_, err := os.Stat(path)
	if err == nil {
		return nil // Directory exists
	}

	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", path, err)
		}
		return nil
	}

	// Some other error occurred with os.Stat (e.g., permission denied to stat)
	return fmt.Errorf("failed to check existence of directory '%s': %w", path, err)
}

// ListFilesWithExtensions returns a list of files with the specified extensions
func ListFilesWithExtensions(dirPath string, extensions []string, recursive bool) ([]string, error) {
	if !DirectoryExists(dirPath) {
		return nil, fmt.Errorf("directory does not exist: %s", dirPath)
	}

	var result []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// This error comes from accessing file info, pass it up to filepath.Walk
			return fmt.Errorf("error accessing path '%s': %w", path, err)
		}

		if info.IsDir() {
			if path != dirPath && !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		for _, ext := range extensions {
			if strings.HasSuffix(strings.ToLower(info.Name()), strings.ToLower(ext)) { // Use info.Name() for suffix check
				result = append(result, path)
				break
			}
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFn)
	if err != nil {
		return nil, fmt.Errorf("error listing files in directory '%s': %w", dirPath, err)
	}

	return result, nil
}

// GetFileInfo returns extended information about a file
func GetFileInfo(filePath string) (FileInfo, error) {
	var fileInfo FileInfo

	info, err := os.Stat(filePath)
	if err != nil {
		return fileInfo, fmt.Errorf("failed to get file info for '%s': %w", filePath, err)
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
		return fmt.Errorf("failed to ensure destination directory for copy operation: %w", err)
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content from %s to %s: %w", sourcePath, destPath, err)
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
		return fmt.Errorf("failed to ensure destination directory %s for move operation: %w", destDir, err)
	}

	originalErr := os.Rename(sourcePath, destPath)
	if originalErr != nil {
		copyErr := CopyFile(sourcePath, destPath)
		if copyErr != nil {
			return fmt.Errorf("failed to move file %s to %s (rename failed: %v, fallback copy also failed): %w", sourcePath, destPath, originalErr, copyErr)
		}

		removeErr := os.Remove(sourcePath)
		if removeErr != nil {
			return fmt.Errorf("file copied successfully from %s to %s, but failed to remove original source file (original rename error: %v): %w", sourcePath, destPath, originalErr, removeErr)
		}
	}

	return nil
}

// DeleteFile deletes a file
func DeleteFile(filePath string) error {
	// Developer note: The current logic to check DirectoryExists first and return nil
	// might mask os.IsNotExist errors from os.Remove if the file itself doesn't exist
	// but the directory does. Consider if os.Remove should always be called and then
	// its error checked (e.g., with os.IsNotExist) if specific handling for non-existence is needed.
	if !DirectoryExists(filepath.Dir(filePath)) {
		// If the directory doesn't exist, the file also doesn't. Silently returning nil.
		return nil
	}

	err := os.Remove(filePath)
	if err != nil {
		// If os.Remove fails (e.g., file doesn't exist, permission issue), wrap and return the error.
		return fmt.Errorf("failed to delete file '%s': %w", filePath, err)
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

// IsDirWritable checks if a directory is writable by attempting to create a temporary file
func IsDirWritable(dirPath string) error {
    if !DirectoryExists(dirPath) {
        return fmt.Errorf("directory does not exist: %s", dirPath)
    }
    
    tempFile := filepath.Join(dirPath, ".write_test")
    f, err := os.Create(tempFile)
    if err != nil {
        return fmt.Errorf("failed to create test file in directory '%s': %w", dirPath, err)
    }
    
    // Close and delete test file
    if err := f.Close(); err != nil {
        return fmt.Errorf("failed to close test file in directory '%s': %w", dirPath, err)
    }
    
    if err := os.Remove(tempFile); err != nil {
        return fmt.Errorf("failed to remove test file in directory '%s': %w", dirPath, err)
    }
    
    return nil
}