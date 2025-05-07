// common/logger.go

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

// Deprecated: Will be removed after new error handling implementation is complete.
// Use SeverityXXX constants from severity.go instead.
const (
	LogLevelDebug   LogLevel = "DEBUG"   // Use SeverityInfo instead
	LogLevelInfo    LogLevel = "INFO"    // Use SeverityInfo instead
	LogLevelWarning LogLevel = "WARNING" // Use SeverityWarning instead
	LogLevelError   LogLevel = "ERROR"   // Use SeverityError instead
)

// Logger handles application logging with file rotation
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
		logPath: logPath, 
		maxSizeMB: maxSizeMB,
		maxAgeDays: maxAgeDays,
 	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Check if rotation is needed on startup
	if err := logger.checkRotation(); err != nil {
		return nil, fmt.Errorf("failed to check log rotation: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger.logFile = file
	if info, err := file.Stat(); err == nil {
		logger.currentSize = info.Size()
	}

	return logger, nil
}

// Log writes a message to the log file
func (l *Logger) Log(level LogLevel, format string, args ...interface{}) error {
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

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(LogLevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(LogLevelInfo, format, args...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	l.Log(LogLevelWarning, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(LogLevelError, format, args...)
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
