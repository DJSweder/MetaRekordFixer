// common/files_helpers.go
// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains file system helper functions and structures.
package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo provides extended information about a file or directory.
// This structure is used to store and pass around file metadata in a structured format.
type FileInfo struct {
	Path      string    // full path to the file or directory
	Name      string    // name of the file or directory without path
	Extension string    // file extension (with dot)
	Directory string    // parent directory path
	Size      int64     // file size in bytes
	ModTime   time.Time // last modification time
	IsDir     bool      // whether this is a directory
}

// NormalizePath normalizes a file system path for consistent comparison and usage.
// It trims whitespace, converts path separators to the correct OS format,
// and cleans the path (resolving any . or .. elements).
//
// Parameters:
//   - path: The path to normalize
//
// Returns:
//   - The normalized path, or an empty string if input is empty
func NormalizePath(path string) string {
	// Return empty string if path is empty
	if IsEmptyString(path) {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
}

// DirectoryExists checks if a directory exists and is accessible.
//
// Parameters:
//   - dirPath: The path to check
//
// Returns:
//   - true if the path exists and is a directory
//   - false otherwise (including if the path doesn't exist or isn't a directory)
func DirectoryExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileExists checks if a file exists and is not a directory.
//
// Parameters:
//   - filePath: The path to check
//
// Returns:
//   - true if the path exists and is a regular file
//   - false otherwise (including if the path doesn't exist or is a directory)
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// EnsureDirectoryExists ensures that the specified directory exists, creating it if necessary.
// It creates all parent directories as needed (similar to 'mkdir -p' in Unix).
//
// Parameters:
//   - path: The directory path to ensure exists
//
// Returns:
//   - nil if the directory exists or was successfully created
//   - An error if the path is empty or if directory creation fails
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

// ListFilesWithExtensions returns a list of files with the specified extensions in a directory.
// It can optionally search recursively through subdirectories.
//
// Parameters:
//   - dirPath: The directory to search in
//   - extensions: List of file extensions to include (e.g., [".mp3", ".wav"])
//   - recursive: Whether to search in subdirectories
//
// Returns:
//   - A slice of file paths matching the extensions, or nil if an error occurs
//   - An error if the directory doesn't exist or can't be read
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

// GetAppDataPath returns the full path to the application's data directory.
// On Windows, this typically points to %APPDATA%/MetaRekordFixer/[subDir].
//
// Parameters:
//   - subDir: Optional subdirectory to append to the base app data path
//
// Returns:
//   - The full path to the application data directory, or an error if APPDATA is not set
func GetAppDataPath(subDir string) (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA environment variable not set")
	}

	path := filepath.Join(appData, "MetaRekordFixer")
	if subDir != "" {
		path = filepath.Join(path, subDir)
	}

	return path, nil
}

// LocateOrCreatePath determines the optimal path for a given file (e.g., log or config)
// based on a defined search order and creates necessary directories.
// Search Order:
// 1. Root directory (next to the executable)
// 2. APPDATA directory
// If not found in either, it attempts to create necessary directories:
// 3. APPDATA directory (with write test)
// 4. Root directory (with write test)
// Returns an error if no writable path can be found.
func LocateOrCreatePath(fileName, subDir string) (string, error) {
	// 1. Check if file exists in the root directory
	rootPath := filepath.Join(".", fileName)
	if FileExists(rootPath) {
		return rootPath, nil
	}

	// 2. Check if file exists in the APPDATA directory
	appDataPath := ""
	appData, err := GetAppDataPath(subDir)
	if err == nil {
		appDataPath = filepath.Join(appData, fileName)
		if FileExists(appDataPath) {
			return appDataPath, nil
		}
	}

	// 3. Try to create the directory in APPDATA and test if it's writable
	if appData != "" {
		dirPath := filepath.Dir(appDataPath)

		// Try to create directory if it doesn't exist
		if err := EnsureDirectoryExists(dirPath); err == nil {
			// Even if directory exists or was created, we need to verify it's actually writable
			if err := IsDirWritable(dirPath); err == nil {
				// Directory is writable, we can use it
				return appDataPath, nil
			} else {
				// Directory exists but is not writable
				CaptureEarlyLog(SeverityWarning, "User's folder ('%s') exists but is not writable: %v", dirPath, err)
				CaptureEarlyLog(SeverityWarning, "Attempt to write to the application installation folder: %s", rootPath)
			}
		} else {
			// If the directory creation failed, we log a message and switch to fallback
			CaptureEarlyLog(SeverityWarning, "User's folder ('%s') is not writable: %v", dirPath, err)
			CaptureEarlyLog(SeverityWarning, "Attempt to write to the application installation folder: %s", rootPath)
		}
	}

	// 4. Fallback: Create directory in root and test if it's writable
	rootDir := filepath.Dir(rootPath)
	if err := EnsureDirectoryExists(rootDir); err != nil {
		return "", fmt.Errorf("failed to create root directory for %s: %w", fileName, err)
	}

	// Test if root directory is writable using existing IsDirWritable function
	if err := IsDirWritable(rootDir); err != nil {
		return "", fmt.Errorf("root directory '%s' is not writable: %w", rootDir, err)
	}

	return rootPath, nil
}

// GetFileInfo returns extended information about a file or directory.
// This function wraps os.Stat() and provides a more structured response.
//
// Parameters:
//   - filePath: The path to the file or directory to inspect
//
// Returns:
//   - A FileInfo struct containing file metadata, or an error if the file doesn't exist or can't be accessed
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

// CopyFile copies a file from source to destination.
// It creates the destination directory if it doesn't exist.
//
// Parameters:
//   - sourcePath: The path to the source file
//   - destPath: The destination path where the file should be copied
//
// Returns:
//   - An error if the source file doesn't exist, can't be read, or the copy fails
//   - nil if the copy is successful
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

// MoveFile moves a file from source to destination.
// It first tries a simple rename operation, falling back to copy+delete if needed.
// Creates the destination directory if it doesn't exist.
//
// Parameters:
//   - sourcePath: The path to the source file
//   - destPath: The destination path where the file should be moved
//
// Returns:
//   - An error if the source file doesn't exist, can't be moved, or the operation fails
//   - nil if the move is successful
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

// DeleteFile deletes a file from the filesystem.
// If the file doesn't exist, it silently succeeds (returns nil).
//
// Note: This behavior might mask some error conditions where the parent directory
// exists but the file can't be deleted due to permissions or other issues.
//
// Parameters:
//   - filePath: The path to the file to be deleted
//
// Returns:
//   - An error if the file exists but can't be deleted
//   - nil if the file was deleted or didn't exist
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

// JoinPaths joins multiple path elements into a single path.
// This is a convenience wrapper around filepath.Join.
//
// Parameters:
//   - elements: The path elements to join
//
// Returns:
//   - A single path string with the elements joined by the OS-specific path separator
func JoinPaths(elements ...string) string {
	return filepath.Join(elements...)
}

// ToDbPath converts a filesystem path to a format suitable for Rekordbox database queries.
// It ensures paths use forward slashes (regardless of OS) and can optionally add a trailing slash
// which is commonly needed for LIKE queries in the Rekordbox database.
//
// Parameters:
//   - path: The filesystem path to convert
//   - addTrailingSlash: Whether to ensure the path ends with a forward slash
//
// Returns:
//   - A path string formatted for use in Rekordbox database queries
func ToDbPath(path string, addTrailingSlash bool) string {
	// Convert to forward slashes for database consistency
	path = filepath.ToSlash(path)

	// Add trailing slash for LIKE queries if requested
	if addTrailingSlash && !strings.HasSuffix(path, "/") {
		path += "/"
	}

	return path
}

// IsDirWritable checks if a directory is writable by attempting to create a temporary file.
// This is a more reliable check than just checking file permissions, as it verifies
// that the actual write operation succeeds.
//
// Parameters:
//   - dirPath: The directory path to check for write access
//
// Returns:
//   - nil if the directory is writable
//   - An error if the directory doesn't exist or isn't writable
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
