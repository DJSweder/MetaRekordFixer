// common/error_handler.go

package common

import (
	"MetaRekordFixer/locales"
	"fmt"
	"log"
	"runtime"
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

// ErrorContext contains additional context for an error
type ErrorContext struct {
	Module      string
	Operation   string
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
		Severity:    ErrorWarning,
		Recoverable: true,
		Timestamp:   time.Now(),
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger      *log.Logger
	lastError   error
	lastContext ErrorContext
	isLogging   bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *log.Logger) *ErrorHandler {
	if logger == nil {
		logger = log.New(log.Writer(), "ERROR: ", log.LstdFlags)
	}

	return &ErrorHandler{
		logger:    logger,
		isLogging: true,
	}
}

// HandleError processes an error, logs it, and displays it to the user if needed
func (h *ErrorHandler) HandleError(err error, context ErrorContext, window fyne.Window, statusLabel *widget.Label) {
	if err == nil {
		return
	}

	if context.StackTrace == "" {
		buf := make([]byte, 8192)
		length := runtime.Stack(buf, false)
		context.StackTrace = string(buf[:length])
	}

	severityStr := "INFO"
	switch context.Severity {
	case ErrorWarning:
		severityStr = "WARNING"
	case ErrorCritical:
		severityStr = "CRITICAL"
	case ErrorFatal:
		severityStr = "FATAL"
	}

	logMessage := fmt.Sprintf("[%s] Module: %s, Operation: %s, Error: %v, Recoverable: %v\n%s",
		severityStr, context.Module, context.Operation, err, context.Recoverable, context.StackTrace)

	h.logger.Println(logMessage)

	if statusLabel != nil {
		statusLabel.SetText(err.Error())
	}

	switch context.Severity {
	case ErrorInfo:
		dialog.ShowInformation(context.Operation, err.Error(), window)
	case ErrorWarning, ErrorCritical:
		h.showCustomErrorDialog(err, context, window)
	case ErrorFatal:
		h.showFatalErrorDialog(err, context, window)
	}
}

// showCustomErrorDialog shows a custom error dialog with more details
func (h *ErrorHandler) showCustomErrorDialog(err error, context ErrorContext, window fyne.Window) {
	message := widget.NewLabel(err.Error())
	message.Wrapping = fyne.TextWrapWord

	detailsLabel := widget.NewLabel(fmt.Sprintf("Module: %s\nOperation: %s", context.Module, context.Operation))
	detailsLabel.Wrapping = fyne.TextWrapWord

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
		locales.Translate("common.dialog.error"),
		locales.Translate("common.button.ok"),
		content,
		window,
	)

	customDialog.Show()
}

// showFatalErrorDialog shows a dialog for fatal errors
func (h *ErrorHandler) showFatalErrorDialog(err error, context ErrorContext, window fyne.Window) {
	message := widget.NewLabel(fmt.Sprintf(locales.Translate("common.dialog.fatal"), err))
	message.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		message,
		widget.NewSeparator(),
		widget.NewButton(locales.Translate("common.button.exit"), func() {
			window.Close()
		}),
	)

	customDialog := dialog.NewCustom(
		locales.Translate("common.dialog.fatal_title"),
		"",
		content,
		window,
	)

	customDialog.SetOnClosed(func() {
		window.Close()
	})

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

// ShowError displays an error message
func ShowError(err error, window fyne.Window) {
	dialog.ShowError(err, window)
}
