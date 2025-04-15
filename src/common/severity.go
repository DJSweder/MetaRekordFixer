package common

// Severity represents the severity level of a message or error.
// It is used consistently across logging, error handling, and status messages.
type Severity string

const (
	// SeverityInfo represents informational messages that don't indicate any problem
	SeverityInfo Severity = "INFO"

	// SeverityWarning represents warning messages that indicate potential issues
	// but don't prevent the application from functioning
	SeverityWarning Severity = "WARNING"

	// SeverityError represents error messages that indicate failures in specific operations
	// but allow the application to continue running
	SeverityError Severity = "ERROR"

	// SeverityCritical represents critical errors that may prevent parts of the application
	// from functioning correctly
	SeverityCritical Severity = "CRITICAL"
)
