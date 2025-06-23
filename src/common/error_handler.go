// common/error_handler.go

package common

import (
	"MetaRekordFixer/locales"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
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

// ShowPanicError displays a critical error dialog and logs the error
func (h *ErrorHandler) ShowPanicError(r interface{}, stackTrace string) {
	title := locales.Translate("err.panic.title")
	content := fmt.Sprintf("%s\n\n%s:\n%v\n\n%s:\n%s",
		locales.Translate("err.panic.message"),
		locales.Translate("err.panic.details"),
		r,
		locales.Translate("err.panic.stacktrace"),
		stackTrace)

	h.logger.Error("PANIC RECOVERED: %v\n%s", r, stackTrace)

	if h.window != nil {
		ShowPanicDialog(h.window, title, content)
	}
}

// ShowInitializationErrorDialog displays a specific dialog for errors that occur during application startup (e.g., config loading).
// It informs the user that the application will continue in a limited capacity.
func (h *ErrorHandler) ShowInitializationErrorDialog(initError error) {
	if initError == nil {
		return
	}

	// Log the initialization error
	h.logger.Error("Initialization Error: %v", initError)

	if h.window != nil {
		// This dialog is intentionally simple, as it's shown on startup before full context is available.
		title := locales.Translate("common.err.inittitle")
		message := fmt.Sprintf("%s\n\n%s:\n%v",
			locales.Translate("common.err.initmessage"),
			locales.Translate("common.err.details"),
			initError)

		// We use ShowInformation because it allows a custom title, which is better for this context
		// than the generic "Error" title from ShowError.
		dialog.ShowInformation(title, message, h.window)
	}
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
