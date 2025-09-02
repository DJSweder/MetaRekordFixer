// common/logger.go

// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains logging functionality.

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// earlyLogBuffer stores log messages before logger is initialized
var earlyLogBuffer []string
var earlyLogMutex sync.Mutex

// CaptureEarlyLog captures a log message before the logger is initialized
func CaptureEarlyLog(level Severity, format string, args ...interface{}) {
	earlyLogMutex.Lock()
	defer earlyLogMutex.Unlock()

	// Format log message with timestamp and level
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("%s [%s] %s", timestamp, level, fmt.Sprintf(format, args...))

	earlyLogBuffer = append(earlyLogBuffer, message)
}

// FlushEarlyLogs writes all captured early logs to the logger
func FlushEarlyLogs(logger *Logger) {
	earlyLogMutex.Lock()
	defer earlyLogMutex.Unlock()

	if logger == nil || len(earlyLogBuffer) == 0 {
		return
	}

	logger.Info("--- Flushing %d early log messages ---", len(earlyLogBuffer))

	for _, message := range earlyLogBuffer {
		// Extract severity from the message
		parts := strings.SplitN(message, "]", 2)
		if len(parts) != 2 {
			// Fallback if message format is unexpected
			logger.Info("Early log: %s", message)
			continue
		}

		// Write directly to log file to preserve original timestamp
		logger.mutex.Lock()
		logger.logFile.WriteString(message + "\n")
		logger.mutex.Unlock()
	}

	// Clear the buffer after flushing
	earlyLogBuffer = nil
	logger.Info("--- End of early logs ---")
}

type Logger struct {
	logPath     string
	logFile     *os.File
	mutex       sync.Mutex
	maxSizeMB   int
	maxAgeDays  int
	currentSize int64
}

// NewLogger creates a new logger instance
func NewLogger(logPath string, maxSizeMB int, maxAgeDays int) (*Logger, error) {
	// Default values for maxSizeMB and maxAgeDays
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if maxAgeDays <= 0 {
		maxAgeDays = 7
	}
	logger := &Logger{
		logPath:    logPath,
		maxSizeMB:  maxSizeMB,
		maxAgeDays: maxAgeDays,
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		// If we can't create the directory, try fallback to root directory
		rootLogPath := filepath.Join(".", filepath.Base(logPath))
		logger.logPath = rootLogPath

		// Log the fallback attempt
		CaptureEarlyLog(SeverityWarning, "Failed to create log directory at '%s': %v", filepath.Dir(logPath), err)
		CaptureEarlyLog(SeverityWarning, "Attempting fallback to root directory: %s", rootLogPath)
	}

	// Check if rotation is needed on startup
	if err := logger.checkRotation(); err != nil {
		// Non-critical error, just log it
		CaptureEarlyLog(SeverityWarning, "Failed to check log rotation: %v", err)
	}

	// Try to open log file
	file, err := os.OpenFile(logger.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// If we can't open the file and we're not already using the root directory, try fallback
		if logger.logPath != filepath.Join(".", filepath.Base(logPath)) {
			rootLogPath := filepath.Join(".", filepath.Base(logPath))
			logger.logPath = rootLogPath

			// Log the fallback attempt
			CaptureEarlyLog(SeverityWarning, "Failed to open log file at '%s': %v", logPath, err)
			CaptureEarlyLog(SeverityWarning, "Attempting fallback to root directory: %s", rootLogPath)

			// Try to open the file in the root directory
			file, err = os.OpenFile(rootLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open log file at primary and fallback locations: %w", err)
			}
		} else {
			// We're already using the root directory and still can't open the file
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	logger.logFile = file
	if info, err := file.Stat(); err == nil {
		logger.currentSize = info.Size()
	}

	return logger, nil
}

// Log writes a message to the log file
func (l *Logger) Log(level Severity, format string, args ...interface{}) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Format log message with timestamp and level
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("%s [%s] %s\n", timestamp, level, fmt.Sprintf(format, args...))

	// Check if rotation is needed
	if l.currentSize >= int64(l.maxSizeMB*1024*1024) {
		if err := l.rotate(); err != nil {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	// Write to log file
	n, err := l.logFile.WriteString(message)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	l.currentSize += int64(n)
	return nil
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(SeverityInfo, format, args...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	l.Log(SeverityWarning, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(SeverityError, format, args...)
}

// Close closes the log file
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// checkRotation checks if log rotation is needed based on age or size
func (l *Logger) checkRotation() error {
	if !FileExists(l.logPath) {
		return nil
	}

	info, err := os.Stat(l.logPath)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	// Check file age
	age := time.Since(info.ModTime())
	if age.Hours() >= float64(l.maxAgeDays*24) {
		return l.rotate()
	}

	// Check file size
	if info.Size() >= int64(l.maxSizeMB*1024*1024) {
		return l.rotate()
	}

	return nil
}

// rotate performs log rotation
func (l *Logger) rotate() error {
	if l.logFile != nil {
		l.logFile.Close()
	}

	// Generate new filename with timestamp
	timestamp := time.Now().Format("2006-01-02@15_04_05")
	dir := filepath.Dir(l.logPath)
	rotatedPath := filepath.Join(dir, fmt.Sprintf("metarekordfixer_%s.log", timestamp))

	// Rename current log file
	if err := os.Rename(l.logPath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Create new log file
	file, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.logFile = file
	l.currentSize = 0

	// Clean old log files
	l.cleanOldLogs()

	return nil
}

// cleanOldLogs removes log files older than 1 year
func (l *Logger) cleanOldLogs() {
	dir := filepath.Dir(l.logPath)
	base := filepath.Base(l.logPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	files, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%s_*%s", name, ext)))
	if err != nil {
		return
	}

	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(oneYearAgo) {
			os.Remove(file)
		}
	}
}
