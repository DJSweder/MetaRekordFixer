// common/error_handler.go

package common

import (
	"MetaRekordFixer/locales"
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	ErrorInfo ErrorSeverity = iota
	ErrorWarning
	ErrorCritical
	ErrorFatal
)

// ErrorContext provides additional information about an error
type ErrorContext struct {
	Module      string
	Operation   string
	Error       error
	Severity    ErrorSeverity
	Recoverable bool
	Timestamp   time.Time
	StackTrace  string
}

// NewErrorContext creates a new error context with defaults
func NewErrorContext(module, operation string) ErrorContext {
	return ErrorContext{
		Module:      module,
		Operation:   operation,
		Severity:    ErrorInfo,
		Recoverable: true,
		Timestamp:   time.Now(),
	}
}

// ErrorHandler handles application errors and logging
type ErrorHandler struct {
	logger    *Logger
	window    fyne.Window
	isLogging bool
}

// NewErrorHandler creates a new error handler instance
func NewErrorHandler(logger *Logger) *ErrorHandler {
	if logger == nil {
		// This should never happen in production
		os.Exit(1)
	}

	return &ErrorHandler{
		logger:    logger,
		isLogging: true,
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

	if h.isLogging {
		h.logger.Error("%v", err)
	}

	if h.window != nil {
		dialog.ShowError(err, h.window)
	}
}

// ShowErrorWithContext displays an error dialog with context and logs the error
func (h *ErrorHandler) ShowErrorWithContext(context ErrorContext) {
	if context.Error == nil {
		return
	}

	if h.isLogging {
		h.logger.Error("%s: %v", context.Operation, context.Error)
	}

	if h.window != nil {
		h.showErrorDialog(context)
	}
}

func (h *ErrorHandler) showErrorDialog(context ErrorContext) {
	message := widget.NewLabel(context.Error.Error())
	message.Wrapping = fyne.TextWrapWord
	message.Resize(fyne.NewSize(400, message.MinSize().Height))

	detailsLabel := widget.NewLabel(fmt.Sprintf("Module: %s\nOperation: %s", context.Module, context.Operation))
	detailsLabel.Wrapping = fyne.TextWrapWord
	detailsLabel.Resize(fyne.NewSize(400, detailsLabel.MinSize().Height))

	content := container.NewVBox(
		message,
		widget.NewSeparator(),
		detailsLabel,
	)

	showStackTraceBtn := widget.NewButtonWithIcon(locales.Translate("common.button.showdetails"), theme.InfoIcon(), nil)

	if context.StackTrace != "" {
		stackTraceArea := widget.NewMultiLineEntry()
		stackTraceArea.SetText(context.StackTrace)
		stackTraceArea.Disable()

		showStackTraceBtn.OnTapped = func() {
			if strings.Contains(showStackTraceBtn.Text, locales.Translate("common.button.showdetails")) {
				content.Add(stackTraceArea)
				showStackTraceBtn.SetText(locales.Translate("common.button.hidedetails"))
			} else {
				content.Remove(stackTraceArea)
				showStackTraceBtn.SetText(locales.Translate("common.button.showdetails"))
			}
		}

		content.Add(showStackTraceBtn)
	}

	customDialog := dialog.NewCustom(
		locales.Translate("common.dialog.errorheader"),
		locales.Translate("common.button.ok"),
		content,
		h.window,
	)

	customDialog.Show()
}

// FormatError creates a standardized error message
func (h *ErrorHandler) FormatError(operation string, err error) error {
	return fmt.Errorf("%s: %v", operation, err)
}

// IsRecoverable checks if the error is recoverable
func (h *ErrorHandler) IsRecoverable(err error) bool {
	return true
}

// SetLoggingEnabled enables or disables error logging
func (h *ErrorHandler) SetLoggingEnabled(enabled bool) {
	h.isLogging = enabled
}

// ShowStandardError displays an error with standard formatting and context
func (h *ErrorHandler) ShowStandardError(err error, context *ErrorContext) {
	if err == nil {
		return
	}

	context.Error = err
	h.showErrorDialog(*context)
}
