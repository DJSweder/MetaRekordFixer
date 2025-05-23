// common/validator.go

package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"MetaRekordFixer/locales"
)

// Validator provides centralized validation functionality for modules.
type Validator struct {
	module       Module         // Reference to the module using the validator
	configMgr    *ConfigManager // For configuration access
	dbMgr        *DBManager     // For database operations
	errorHandler *ErrorHandler  // For error handling
}

// NewValidator creates a new instance of Validator.
func NewValidator(module Module, configMgr *ConfigManager, dbMgr *DBManager, errorHandler *ErrorHandler) *Validator {
	return &Validator{
		module:       module,
		configMgr:    configMgr,
		dbMgr:        dbMgr,
		errorHandler: errorHandler,
	}
}

// Validate performs all necessary validations and returns an error if any validation fails.
// The action parameter specifies which validation rules should be applied based on
// the action being performed (e.g., "standard", "custom", etc.). If a field has
// ValidateOnActions specified, it will only be validated when the current action
// matches one of the specified actions. If ValidateOnActions is empty, the field
// will be validated for all actions.
func (v *Validator) Validate(action string) error {
	// Save configuration
	v.module.SaveConfig()

	// Get base module functionality
	base, ok := v.module.(interface {
		ClearStatusMessages()
		AddInfoMessage(string)
		AddErrorMessage(string)
	})
	if !ok {
		return fmt.Errorf("module %s must implement status message methods", v.module.GetName())
	}

	// Clear any previous status messages
	base.ClearStatusMessages()

	// Add initial status message
	base.AddInfoMessage(locales.Translate("validator.status.start"))

	// Validate input fields
	if err := v.validateFields(action); err != nil {
		base.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return err
	}
	base.AddInfoMessage(locales.Translate("validator.status.entries"))

	// Validate database if needed
	if v.module.GetDatabaseRequirements().NeedsDatabase {
		if err := v.validateDatabase(); err != nil {
			base.AddErrorMessage(locales.Translate("common.err.statusfinal"))
			return err
		}
		base.AddInfoMessage(locales.Translate("validator.status.dbconnect"))

		// Create database backup
		if err := v.backupDatabase(); err != nil {
			base.AddErrorMessage(locales.Translate("common.err.statusfinal"))
			return err
		}
		base.AddInfoMessage(locales.Translate("common.db.backupdone"))
		base.AddInfoMessage(locales.Translate("common.status.start"))
	} else {
		// If no database is needed, just add start message
		base.AddInfoMessage(locales.Translate("common.status.start"))
	}

	return nil
}

// validateFields checks all fields according to their definitions and validation rules.
func (v *Validator) validateFields(action string) error {
	// Get module configuration
	cfg := v.configMgr.GetModuleConfig(v.module.GetConfigName())
	if IsNilConfig(cfg) {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateFields",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := fmt.Errorf("configuration not found for module %s", v.module.GetName())
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	// Check if Fields map exists
	if cfg.Fields == nil {
		return nil // No fields to validate
	}

	// Validate each field
	for key, field := range cfg.Fields {
		// Skip validation if field's ValidateOnActions doesn't include current action
		if len(field.ValidateOnActions) > 0 {
			actionFound := false
			for _, validAction := range field.ValidateOnActions {
				if validAction == action {
					actionFound = true
					break
				}
			}
			if !actionFound {
				continue
			}
		}

		value := cfg.Get(key, "")

		// Skip validation if field depends on another field and condition is not met
		if field.DependsOn != "" {
			dependentValue := cfg.Get(field.DependsOn, "")
			if dependentValue != field.ActiveWhen {
				continue
			}
		}

		// Create standard error context
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateFields",
			Severity:    SeverityCritical,
			Recoverable: false,
		}

		// Validate date format if needed
		if field.FieldType == "date" {
			if !IsEmptyString(value) && !IsValidDateFormat(value) {
				err := errors.New(locales.Translate("validator.err.invaliddate"))
				v.errorHandler.ShowStandardError(err, context)
				return err
			}
		}

		// Check required fields
		if field.Required && IsEmptyString(value) {
			var err error
			switch field.FieldType {
			case "folder":
				err = errors.New(locales.Translate("validator.err.nofolder"))
			case "playlist":
				err = errors.New(locales.Translate("validator.err.noplaylist"))
			case "date":
				err = errors.New(locales.Translate("validator.err.invaliddate"))
			default:
				err = errors.New(locales.Translate("validator.err.required"))
			}

			v.errorHandler.ShowStandardError(err, context)
			return err
		}

		// Skip further validation if field is empty (and not required)
		if IsEmptyString(value) {
			continue
		}

		// Validate field value based on validation type
		if !IsEmptyString(field.ValidationType) {
			switch field.ValidationType {
			case "exists":
				// Use DirectoryExists for folders, FileExists pro files
				var exists bool
				if field.FieldType == "folder" {
					exists = DirectoryExists(value)
				} else {
					exists = FileExists(value)
				}

				if !exists {
					// For error dialog get only foldername instead of path
					displayName := filepath.Base(value)
					msg := fmt.Sprintf(locales.Translate("validator.err.foldernotexist"), displayName)
					err := errors.New(msg)
					v.errorHandler.ShowStandardError(err, context)
					return err
				}

			case "exists | write":
				// Check if folder exists
				if !DirectoryExists(value) {
					// Get foldername only for error dialog
					displayName := filepath.Base(value)
					msg := fmt.Sprintf(locales.Translate("validator.err.foldernotexist"), displayName)
					err := errors.New(msg)
					v.errorHandler.ShowStandardError(err, context)
					return err
				}

				// Check write permissions by trying to create a temporary file
				tempFile := filepath.Join(value, ".write_test")
				f, err := os.Create(tempFile)
				if err != nil {
					// Get foldername only for error dialog
					displayName := filepath.Base(value)
					msg := fmt.Sprintf(locales.Translate("validator.err.nowriteaccess"), displayName)
					err := errors.New(msg)
					v.errorHandler.ShowStandardError(err, context)
					return err
				}
				defer func() {
					f.Close()
					os.Remove(tempFile)
				}()
			}
		}
	}

	return nil
}

// validateDatabase checks database dependencies and validates database access.
// Returns error if any validation fails.
func (v *Validator) validateDatabase() error {
	// Validate database path
	if err := v.validateDatabasePath(); err != nil {
		return err
	}

	// Validate database access
	if err := v.validateDatabaseAccess(); err != nil {
		return err
	}

	// Test database connection if immediate access is not required
	if !v.module.GetDatabaseRequirements().NeedsImmediateAccess {
		if err := v.validateDatabaseConnection(); err != nil {
			return err
		}
	}

	return nil
}

// validateDatabasePath checks if database path is set and exists.
func (v *Validator) validateDatabasePath() error {
	dbPath := v.dbMgr.GetDatabasePath()
	if IsEmptyString(dbPath) {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateDatabase",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := errors.New(locales.Translate("common.err.nodbpath"))
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	if !FileExists(dbPath) {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateDatabase",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := errors.New(locales.Translate("common.err.dbnotexist"))
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	return nil
}

// validateDatabaseAccess checks write permissions to database directory.
func (v *Validator) validateDatabaseAccess() error {
	dbDir := filepath.Dir(v.dbMgr.GetDatabasePath())

	// Try to create a temporary file to test write permissions
	tempFile := filepath.Join(dbDir, ".write_test")
	f, err := os.Create(tempFile)
	if err != nil {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateDatabase",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := errors.New(locales.Translate("common.err.nodbwriteaccess"))
		v.errorHandler.ShowStandardError(err, context)
		return err
	}
	defer func() {
		f.Close()
		os.Remove(tempFile)
	}()

	return nil
}

// validateDatabaseConnection tests database connection.
func (v *Validator) validateDatabaseConnection() error {
	if err := v.dbMgr.Connect(); err != nil {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateDatabase",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		msg := fmt.Sprintf(locales.Translate("common.err.connectdb"), err)
		err := errors.New(msg)
		v.errorHandler.ShowStandardError(err, context)
		return err
	}
	defer v.dbMgr.Finalize()

	return nil
}

// IsValidDateFormat checks if the given string is a valid date in the format YYYY-MM-DD.
func IsValidDateFormat(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

// IsEmptyString checks if a string is empty or contains only whitespace.
func IsEmptyString(s string) bool {
	return strings.TrimSpace(s) == ""
}

// backupDatabase creates a backup of the database.
// It uses DBManager to create a backup of the current database file.
// The backup is created in the same directory as the original database
// with a timestamp suffix.
// Returns error if backup creation fails.
func (v *Validator) backupDatabase() error {
	context := &ErrorContext{
		Module:      v.module.GetName(),
		Operation:   "BackupDatabase",
		Severity:    SeverityCritical,
		Recoverable: false,
	}

	if err := v.dbMgr.BackupDatabase(); err != nil {
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	return nil
}
