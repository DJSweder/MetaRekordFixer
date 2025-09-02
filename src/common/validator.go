// common/validator.go

// Package common implements shared functionality used across the MetaRekordFixer application.
// This file is responsible for validating input fields and backing up the database before any writes.

package common

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"MetaRekordFixer/locales"
)

// Validator handles validation of module inputs and database operations.
type Validator struct {
	module       Module         // The module being validated
	configMgr    *ConfigManager // For configuration management
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
	// Get base module functionality
	base := v.module.(interface {
		ClearStatusMessages()
		AddInfoMessage(string)
		AddErrorMessage(string)
	})

	// Clear any previous status messages
	base.ClearStatusMessages()

	// Add initial status message
	base.AddInfoMessage(locales.Translate("validator.status.start"))

	// Validate input fields using new typed system
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

		// Generic preflight: check input folder accessibility and (optionally) presence of files by configured extensions
		moduleKey := v.module.GetConfigName()
		if typedCfg, err := v.configMgr.GetModuleCfg(moduleKey, moduleKey); err == nil && typedCfg != nil {
			// Extract FieldCfg map
			fields, _ := extractFieldConfigs(typedCfg)

			// Resolve source folder (prefer sourceFolder -> folder -> any active folder field)
			sourceFolder := ""
			if f, ok := fields["sourceFolder"]; ok && isFieldActive(f, fields) {
				sourceFolder = NormalizePath(f.Value)
			} else if f, ok := fields["folder"]; ok && isFieldActive(f, fields) {
				sourceFolder = NormalizePath(f.Value)
			} else {
				for _, fld := range fields {
					if fld.FieldType == "folder" && isFieldActive(fld, fields) {
						sourceFolder = NormalizePath(fld.Value)
						break
					}
				}
			}

			// Resolve recursive flag if present
			recursive := false
			if r, ok := fields["recursive"]; ok {
				recursive = strings.ToLower(strings.TrimSpace(r.Value)) == "true"
			}

			// Parse extensions if present
			var extensions []string
			if e, ok := fields["extensions"]; ok {
				extensions = parseExtensionsCSV(e.Value)
			}

			if !IsEmptyString(sourceFolder) {
				files, skippedDirs, err := GetFilesInFolder(v.dbMgr.logger, sourceFolder, extensions, recursive)

				// Log any skipped directories
				if len(skippedDirs) > 0 {
					for _, dir := range skippedDirs {
						v.dbMgr.logger.Warning("%s %s", fmt.Sprintf(locales.Translate("common.log.folder"), dir), locales.Translate("common.log.foldernoread"))
					}
				}

				if err != nil {
					// Check if the error is the specific permission error using sentinel error
					if errors.Is(err, ErrDirectoryNotReadable) {
						// Create localized error for root directory access issue
						displayName := filepath.Base(sourceFolder)
						localizedErr := fmt.Errorf(locales.Translate("common.err.noreadaccess"), displayName)
						context := &ErrorContext{
							Module:      v.module.GetName(),
							Operation:   "ValidateInputFiles",
							Severity:    SeverityCritical,
							Recoverable: false,
						}
						v.errorHandler.ShowStandardError(localizedErr, context)
						base.AddErrorMessage(locales.Translate("common.err.statusfinal"))
						return localizedErr
					}
					// For other errors, show them as-is
					context := &ErrorContext{
						Module:      v.module.GetName(),
						Operation:   "ValidateInputFiles",
						Severity:    SeverityCritical,
						Recoverable: false,
					}
					v.errorHandler.ShowStandardError(err, context)
					base.AddErrorMessage(locales.Translate("common.err.statusfinal"))
					return err
				}
				// Skipped directories are logged above, no need to store them
				if len(skippedDirs) > 0 {
					base.AddInfoMessage(locales.Translate("common.status.foldersdeny"))
				}
				// If extensions are configured and no files found, treat as critical
				if len(extensions) > 0 && len(files) == 0 {
					context := &ErrorContext{
						Module:      v.module.GetName(),
						Operation:   "ValidateInputFiles",
						Severity:    SeverityCritical,
						Recoverable: false,
					}
					err := errors.New(locales.Translate("common.err.nofiles"))
					v.errorHandler.ShowStandardError(err, context)
					base.AddErrorMessage(locales.Translate("common.err.nofiles"))
					return err
				}
			}
		}

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
// Uses the new typed configuration system via reflection to extract FieldCfg fields.
func (v *Validator) validateFields(action string) error {
	// Get typed configuration from module via ConfigManager
	moduleType := v.module.GetConfigName()
	typedCfg, err := v.configMgr.GetModuleCfg(moduleType, moduleType)
	if err != nil {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateInputFields",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := fmt.Errorf(locales.Translate("common.err.confignotfound"), v.module.GetName())
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	if typedCfg == nil {
		return nil // No configuration to validate
	}

	// Extract FieldCfg fields via reflection
	fields, err := extractFieldConfigs(typedCfg)
	if err != nil {
		return fmt.Errorf("failed to extract field for validation: %w", err)
	}

	// Validate each field
	for _, field := range fields {
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

		value := field.Value

		// Skip validation if field depends on another field and condition is not met
		if field.DependsOn != "" {
			// Find dependent field value in the same config
			if dependentField, exists := fields[field.DependsOn]; exists {
				if dependentField.Value != field.ActiveWhen {
					continue
				}
			} else {
				// Dependent field not found, skip validation
				continue
			}
		}

		// Create standard error context
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateInputFields",
			Severity:    SeverityCritical,
			Recoverable: false,
		}

		// Validate date format if needed
		if field.FieldType == "date" {
			if !IsEmptyString(value) && !IsValidDateFormat(value) {
				err := errors.New(locales.Translate("validator.err.invaliddate")) // In this case, it is intentional that the user gets a more general message about the date entered.
				// In the GUI, user see what entered, so "bad date" also means the case where "no date" is entered.
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
				// Use DirectoryExists for folders, FileExists for files
				var exists bool
				if field.FieldType == "folder" {
					exists = DirectoryExists(value)
				} else {
					exists = FileExists(value)
				}

				if !exists {
					// For error dialog get only foldername instead of path
					displayName := filepath.Base(value)
					err := fmt.Errorf(locales.Translate("validator.err.foldernotexist"), displayName)
					v.errorHandler.ShowStandardError(err, context)
					return err
				}

			case "exists | write":
				// Check if folder exists
				if !DirectoryExists(value) {
					// Get foldername only for error dialog
					displayName := filepath.Base(value)
					err := fmt.Errorf(locales.Translate("validator.err.foldernotexist"), displayName)
					v.errorHandler.ShowStandardError(err, context)
					return err
				}

				// Check write permissions by trying to create a temporary file
				if err := IsDirWritable(value); err != nil {
					err := fmt.Errorf("%s: %w", locales.Translate("validator.err.nowriteaccess"), err)
					v.errorHandler.ShowStandardError(err, context)
					return err
				}
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

	// Note: We only run the preflight DB connection test when the module does NOT require
	// immediate access (NeedsImmediateAccess == false). Modules that read from the DB right
	// after the GUI opens skip this pre-test; the first real DB call handles a lazy connect.
	// This reduces I/O and prevents the immediate connect -> finalize -> connect sequence.
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
			Operation:   "ValidateDbPath",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := errors.New(locales.Translate("common.err.dbpath"))
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	if !FileExists(dbPath) {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateDbFile",
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
	if err := IsDirWritable(dbDir); err != nil {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "BackupDatabase",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
		err := fmt.Errorf("%s: %w", locales.Translate("common.err.nodbwriteaccess"), err)
		v.errorHandler.ShowStandardError(err, context)
		return err
	}

	return nil
}

// validateDatabaseConnection tests database connection.
func (v *Validator) validateDatabaseConnection() error {
	if err := v.dbMgr.Connect(); err != nil {
		context := &ErrorContext{
			Module:      v.module.GetName(),
			Operation:   "ValidateDatabaseConnection",
			Severity:    SeverityCritical,
			Recoverable: false,
		}
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

// extractFieldConfigs uses reflection to extract all FieldCfg fields from a given config struct.
// Keys are taken from the struct field's `json` tag (e.g., "sourceFolder", "folder", "recursive", "extensions").
func extractFieldConfigs(config interface{}) (map[string]FieldCfg, error) {
	result := make(map[string]FieldCfg)
	if config == nil {
		return result, nil
	}

	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return result, nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("extractFieldConfigs: expected struct, got %s", val.Kind().String())
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		// Only process fields of type FieldCfg
		if fieldVal.Kind() == reflect.Struct && fieldVal.Type().Name() == "FieldCfg" {
			key := fieldType.Tag.Get("json")
			if key == "" {
				key = strings.TrimSpace(fieldType.Name)
				key = strings.ToLower(key[:1]) + key[1:]
			}
			if !fieldVal.CanInterface() {
				continue
			}
			if cfg, ok := fieldVal.Interface().(FieldCfg); ok {
				result[key] = cfg
			}
		}
	}

	return result, nil
}

// parseExtensionsCSV parses CSV/space/semicolon/pipe-separated extensions into normalized dot-prefixed, lowercased list.
// Empty or whitespace-only input yields nil (meaning: no filter configured).
func parseExtensionsCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', ' ', '|':
			return true
		default:
			return false
		}
	})
	seen := make(map[string]struct{})
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		res = append(res, p)
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

// isFieldActive returns true if the field is active based on DependsOn/ActiveWhen conditions within the same config.
func isFieldActive(field FieldCfg, fields map[string]FieldCfg) bool {
	if field.DependsOn == "" {
		return true
	}
	if dep, ok := fields[field.DependsOn]; ok {
		return dep.Value == field.ActiveWhen
	}
	// If dependency is not found, consider it active to avoid false negatives
	return true
}

// GetSkippedDirs is no longer needed as validator doesn't store skipped directories
// Main processing logic handles its own directory counting
func (v *Validator) GetSkippedDirs() []string {
	return nil
}

// ... (rest of the code remains the same)

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
