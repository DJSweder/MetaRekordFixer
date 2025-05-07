package common

import (
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

// ErrorContext provides additional information about an error
type ErrorContext struct {
	Module      string
	Operation   string
	Error       error
	Severity    Severity
	Recoverable bool
	Timestamp   time.Time
	StackTrace  string
}

// NewErrorContext creates a new error context with defaults
func NewErrorContext(module, operation string) ErrorContext {
	return ErrorContext{
		Module:      module,
		Operation:   operation,
		Severity:    SeverityInfo,
		Recoverable: true,
		Timestamp:   time.Now(),
	}
}

// ErrorHandler handles application errors and logging
type ErrorHandler struct {
	logger *Logger
	window fyne.Window
}

// NewErrorHandler creates a new error handler instance
func NewErrorHandler(logger *Logger, window fyne.Window) *ErrorHandler {
	if logger == nil {
		// This should never happen in production
		os.Exit(1)
	}

	return &ErrorHandler{
		logger: logger,
		window: window,
	}
}

// SetWindow sets the window for displaying error dialogs
func (h *ErrorHandler) SetWindow(window fyne.Window) {
	h.window = window
}

// GetLogger returns the logger instance
func (h *ErrorHandler) GetLogger() *Logger {
	return h.logger
}

// ShowError displays an error dialog and logs the error
func (h *ErrorHandler) ShowError(err error) {
	if err == nil {
		return
	}

	// Log error without context
	h.logger.Error("%s", err.Error())

	if h.window != nil {
		context := NewErrorContext("", "")
		context.Error = err
		ShowStandardError(h.window, err, &context)
	}
}

// ShowErrorWithContext displays an error dialog with context and logs the error
func (h *ErrorHandler) ShowErrorWithContext(context ErrorContext) {
	if context.Error == nil {
		return
	}

	// Log error with context
	h.logger.Error("Module: %s, Operation: %s - %s", context.Module, context.Operation, context.Error.Error())

	if h.window != nil {
		ShowStandardError(h.window, context.Error, &context)
	}
}

// FormatError creates a standardized error message
func (h *ErrorHandler) FormatError(operation string, err error) error {
	return fmt.Errorf("%s: %v", operation, err)
}

// IsRecoverable checks if the error is recoverable
func (h *ErrorHandler) IsRecoverable(err error) bool {
	return true
}

// ShowStandardError displays an error with standard formatting and context
func (h *ErrorHandler) ShowStandardError(err error, context *ErrorContext) {
	if err == nil {
		return
	}

	// Log error with context
	if context != nil {
		h.logger.Error("Module: %s, Operation: %s - %s", context.Module, context.Operation, err.Error())
	} else {
		h.logger.Error(err.Error())
	}

	// Update context with error and show dialog
	context.Error = err
	if h.window != nil {
		ShowStandardError(h.window, err, context)
	}
}
