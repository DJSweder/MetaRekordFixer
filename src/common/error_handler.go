// common/error_handler.go
// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains error handling functionality.

package common

import (
	"MetaRekordFixer/locales"
	"fmt"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// Severity represents the severity level of a message or error.
// It is used consistently across logging, error handling, and status messages.
type Severity string

const (
	// SeverityInfo represents informational messages that don't indicate any problem
	SeverityInfo Severity = "INFO "

	// SeverityWarning represents warning messages that indicate potential issues
	// but don't prevent the application from functioning
	SeverityWarning Severity = "WARN"

	// SeverityError represents error messages that indicate failures in specific operations
	// but allow the application to continue running
	SeverityError Severity = "ERROR"

	// SeverityCritical represents critical errors that may prevent parts of the application
	// from functioning correctly
	SeverityCritical Severity = "CRITICAL"
)

// ErrorContext provides additional information about an error
type ErrorContext struct {
	Module      string    // module where the error occurred
	Operation   string    // operation that caused the error
	Error       error     // the actual error
	Severity    Severity  // severity level of the error
	Recoverable bool      // whether the error is recoverable
	Timestamp   time.Time // when the error occurred
	StackTrace  string    // stack trace for debugging
}

// NewErrorContext creates a new error context with default values.
// This function initializes an ErrorContext structure with the specified module and operation,
// setting default values for other fields (SeverityInfo, Recoverable=true, current timestamp).
//
// Parameters:
//   - module: The name of the module where the error occurred
//   - operation: The name of the operation that failed
//
// Returns:
//   - An initialized ErrorContext structure with default values
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
	mutex  sync.Mutex
}

// NewErrorHandler creates a new error handler instance.
// This function initializes an ErrorHandler with the provided logger and window.
// If the logger is nil, the application will exit as this is a critical dependency.
//
// Parameters:
//   - logger: A pointer to a Logger instance for error logging
//   - window: The main application window for displaying error dialogs
//
// Returns:
//   - A pointer to the initialized ErrorHandler
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

// GetLogger returns the logger instance associated with this error handler.
// This method provides access to the internal logger for external logging needs.
//
// Returns:
//   - A pointer to the Logger instance
func (h *ErrorHandler) GetLogger() *Logger {
	return h.logger
}

// ShowError displays an error dialog and logs the error.
// This method logs the error message and displays a standard error dialog
// if a window is available. If the error is nil, no action is taken.
//
// Parameters:
//   - err: The error to display and log
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

// ShowErrorWithContext displays an error dialog with context and logs the error.
// This method logs the error with additional context information (module, operation)
// and displays a standard error dialog if a window is available.
// If the error in the context is nil, no action is taken.
//
// Parameters:
//   - context: The ErrorContext containing the error and additional information
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

// ShowPanicError displays a critical error dialog and logs the error from a recovered panic.
// This method formats the panic information with a stack trace and displays it in a
// specialized panic dialog. It also logs the panic details with the ERROR severity level.
//
// Parameters:
//   - r: The recovered panic value (typically from recover())
//   - stackTrace: The stack trace string showing where the panic occurred
func (h *ErrorHandler) ShowPanicError(r interface{}, stackTrace string) {
	title := locales.Translate("common.dialog.criticalheader")
	content := fmt.Sprintf("%s\n\n%s:\n%v\n\n%s:\n%s",
		locales.Translate("common.err.panic"),
		locales.Translate("common.err.panicdetails"),
		r,
		locales.Translate("common.err.panicstack"),
		stackTrace)

	h.logger.Error("PANIC RECOVERED: %v\n%s", r, stackTrace)

	if h.window != nil {
		ShowPanicDialog(h.window, title, content)
	}
}

// ShowStandardError displays an error with standard formatting and context.
// This method logs the error with context information if available.
// It then displays a standard error dialog if a window is available.
// If the error is nil, no action is taken.
//
// Parameters:
//   - err: The error to display and log
//   - context: Additional context information about the error (may be nil)
func (h *ErrorHandler) ShowStandardError(err error, context *ErrorContext) {
	if err == nil {
		return
	}

	// Log error with context
	if context != nil {
		h.logger.Error("Module: %s, Operation: %s - %s", context.Module, context.Operation, err.Error())
	} else {
		h.logger.Error("%s", err.Error())
	}

	// Update context with error and show dialog
	context.Error = err
	if h.window != nil {
		ShowStandardError(h.window, err, context)
	}
}
