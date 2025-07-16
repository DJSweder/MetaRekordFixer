// common/config_manager.go
// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains configuration management functionality.

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// GlobalConfig holds global application settings that are shared across all modules.
// These settings typically include application-wide preferences and configurations.
type GlobalConfig struct {
	DatabasePath string
	Language     string
}

// ModuleConfig defines a configuration structure for individual modules.
// Each module can define its own set of configuration fields with different types and validation rules.
type ModuleConfig struct {
	Fields map[string]FieldDefinition
}

// ConfigManager handles loading, saving, and managing application configuration.
// It provides thread-safe access to both global and module-specific settings.
type ConfigManager struct {
	configPath    string
	globalConfig  GlobalConfig
	moduleConfigs map[string]ModuleConfig
	mutex         sync.Mutex
}

// FieldDefinition defines the structure and validation rules for a configuration field.
// It supports various field types, dependencies, and validation requirements.
type FieldDefinition struct {
	FieldType         string   // field type (folder, date, checkbox, select, playlist, file)
	Required          bool     // whether the field is required
	DependsOn         string   // field that this field depends on
	ActiveWhen        string   // condition when this field is active
	ValidationType    string   // validation type (exists, valid_date, filled, exists | write)
	Value             string   // field value
	ValidateOnActions []string // actions that trigger validation
}

// NewConfigManager initializes a new configuration manager instance.
// It attempts to load existing configuration from the specified path, or creates a new one if it doesn't exist.
//
// Parameters:
//   - configPath: Path to the configuration file
//
// Returns:
//   - *ConfigManager: Initialized configuration manager
//   - error: Any error that occurred during initialization
func NewConfigManager(configPath string) (*ConfigManager, error) {
	mgr := &ConfigManager{
		configPath:    configPath,
		moduleConfigs: make(map[string]ModuleConfig),
	}

	// Try to load the configuration. An error is not critical here if the file is simply new and empty.
	// A default config will be created on the first save if necessary.
	if err := mgr.loadConfig(); err != nil {
		// If the file doesn't exist, it means LocateOrCreatePath failed, which is a critical error.
		// For other errors like malformed JSON, we log it but continue, as a new config will be saved.
		CaptureEarlyLog(SeverityInfo, "Creating a new configuration file '%s' is necessary because it does not exist in the usual location.: %v", configPath, err)
	}

	return mgr, nil
}

// GetGlobalConfig returns a copy of the current global configuration.
// The returned value is a copy to prevent race conditions and ensure thread safety.
func (mgr *ConfigManager) GetGlobalConfig() GlobalConfig {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	return mgr.globalConfig
}

// SaveGlobalConfig updates and saves the global configuration.
// This operation is thread-safe and will persist changes to disk.
//
// Parameters:
//   - config: The new global configuration to save
//
// Returns:
//   - error: Any error that occurred during the save operation
func (mgr *ConfigManager) SaveGlobalConfig(config GlobalConfig) error {
	mgr.mutex.Lock()
	mgr.globalConfig = config
	mgr.mutex.Unlock()

	return mgr.saveConfig()
}

// GetModuleConfig retrieves the configuration for a specific module.
// If the module doesn't have a configuration yet, a new empty one is created.
//
// Parameters:
//   - moduleName: The name of the module to get configuration for
//
// Returns:
//   - ModuleConfig: The module's configuration (never nil)
func (mgr *ConfigManager) GetModuleConfig(moduleName string) ModuleConfig {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	if config, exists := mgr.moduleConfigs[moduleName]; exists {
		if config.Fields == nil {
			config.Fields = make(map[string]FieldDefinition)
			mgr.moduleConfigs[moduleName] = config
		}
		return config
	}
	return NewModuleConfig()
}

// SaveModuleConfig updates and saves the configuration for a specific module.
// The configuration is immediately persisted to disk.
//
// Parameters:
//   - moduleName: The name of the module to save configuration for
//   - config: The module configuration to save
func (mgr *ConfigManager) SaveModuleConfig(moduleName string, config ModuleConfig) {
	mgr.mutex.Lock()
	mgr.moduleConfigs[moduleName] = config
	mgr.mutex.Unlock()

	mgr.saveConfig()
}

// loadConfig loads the application configuration from the configuration file.
// This is an internal method that is called during initialization.
//
// Returns:
//   - error: Any error that occurred during loading
func (mgr *ConfigManager) loadConfig() error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	data, err := os.ReadFile(mgr.configPath)
	if err != nil {
		return err // The file should exist thanks to LocateOrCreatePath
	}

	// If the file is empty, it's a new config, so we just return.
	if len(data) == 0 {
		return nil
	}

	// Parse JSON data
	var configData struct {
		Global  GlobalConfig            `json:"global"`
		Modules map[string]ModuleConfig `json:"modules"`
	}
	err = json.Unmarshal(data, &configData)
	if err != nil {
		return fmt.Errorf("ConfigManager.loadConfig: failed to unmarshal config data from %s: %w", mgr.configPath, err)
	}

	mgr.globalConfig = configData.Global
	mgr.moduleConfigs = configData.Modules

	return nil
}

// saveConfig saves the current configuration to the configuration file.
// This is an internal method that handles the actual file I/O operations.
// It ensures the configuration is properly formatted and includes default values.
//
// Returns:
//   - error: Any error that occurred during the save operation
func (mgr *ConfigManager) saveConfig() error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	// Ensure default values
	if mgr.globalConfig.DatabasePath == "" {
		mgr.globalConfig.DatabasePath = "" // Set default value
	}
	if mgr.globalConfig.Language == "" {
		mgr.globalConfig.Language = "" // Default language
	}

	config := struct {
		Global  GlobalConfig            `json:"global"`
		Modules map[string]ModuleConfig `json:"modules"`
	}{
		Global:  mgr.globalConfig,
		Modules: mgr.moduleConfigs,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("ConfigManager.saveConfig: failed to marshal config data: %w", err)
	}

	// The directory and file should already exist, so we just write to it.
	if err = os.WriteFile(mgr.configPath, data, 0644); err != nil {
		return fmt.Errorf("ConfigManager.saveConfig: failed to write config file %s: %w", mgr.configPath, err)
	}

	return nil
}

// NewModuleConfig creates a new empty module configuration with an initialized fields map.
// This ensures that the Fields map is never nil when working with a new ModuleConfig.
//
// Returns:
//   - ModuleConfig: A new, empty module configuration
func NewModuleConfig() ModuleConfig {
	return ModuleConfig{
		Fields: make(map[string]FieldDefinition),
	}
}

// Get retrieves a string value from the module configuration.
// If the specified key doesn't exist, it returns the provided default value.
//
// Parameters:
//   - key: The configuration key to retrieve
//   - defaultValue: The value to return if the key doesn't exist
//
// Returns:
//   - string: The configuration value or the default value if not found
func (c ModuleConfig) Get(key string, defaultValue string) string {
	if field, exists := c.Fields[key]; exists {
		return field.Value
	}
	return defaultValue
}

// Set stores a string value in the module configuration with a default field type of "folder".
// This is a convenience method for simple string values without complex validation.
//
// Parameters:
//   - key: The configuration key to set
//   - value: The string value to store
func (c *ModuleConfig) Set(key string, value string) {
	if c.Fields == nil {
		c.Fields = make(map[string]FieldDefinition)
	}
	c.Fields[key] = FieldDefinition{
		FieldType: "folder",
		Value:     value,
	}
}

func (c *ModuleConfig) SetWithDefinition(key string, value string, fieldType string, required bool, validationType string) {

	// Save definition
	if c.Fields == nil {
		c.Fields = make(map[string]FieldDefinition)
	}
	c.Fields[key] = FieldDefinition{
		FieldType:      fieldType,
		Required:       required,
		Value:          value,
		ValidationType: validationType,
	}
}

// SetWithDefinitionAndActions stores a string value in the module configuration with field definition and validation actions
func (cfg *ModuleConfig) SetWithDefinitionAndActions(key string, value string, fieldType string, required bool, validationType string, validateOnActions []string) {
	if cfg.Fields == nil {
		cfg.Fields = make(map[string]FieldDefinition)
	}
	cfg.Fields[key] = FieldDefinition{
		FieldType:         fieldType,
		Required:          required,
		ValidationType:    validationType,
		Value:             value,
		ValidateOnActions: validateOnActions,
	}
}

// SetWithDependencyAndActions stores a string value in the module configuration with dependency and validation actions
func (cfg *ModuleConfig) SetWithDependencyAndActions(key string, value string, fieldType string, required bool, dependsOn string, activeWhen string, validationType string, validateOnActions []string) {
	if cfg.Fields == nil {
		cfg.Fields = make(map[string]FieldDefinition)
	}
	cfg.Fields[key] = FieldDefinition{
		FieldType:         fieldType,
		Required:          required,
		DependsOn:         dependsOn,
		ActiveWhen:        activeWhen,
		ValidationType:    validationType,
		Value:             value,
		ValidateOnActions: validateOnActions,
	}
}

// GetBool retrieves a boolean value from the module configuration
func (c ModuleConfig) GetBool(key string, defaultValue bool) bool {
	if field, exists := c.Fields[key]; exists {
		return field.Value == "true"
	}
	return defaultValue
}

func (c *ModuleConfig) SetBoolWithDefinition(key string, value bool, required bool, validationType string) {
	if c.Fields == nil {
		c.Fields = make(map[string]FieldDefinition)
	}
	c.Fields[key] = FieldDefinition{
		FieldType:      "checkbox",
		Required:       required,
		Value:          fmt.Sprintf("%t", value),
		ValidationType: validationType,
	}
}

// SetIntWithDefinition stores an integer value in the module configuration
func (c *ModuleConfig) SetIntWithDefinition(key string, value int, required bool) {
	if c.Fields == nil {
		c.Fields = make(map[string]FieldDefinition)
	}
	c.Fields[key] = FieldDefinition{
		FieldType: "number", // for numeric values
		Required:  required,
		Value:     fmt.Sprintf("%d", value),
	}
}

// IsNilConfig checks if a given configuration is nil
func IsNilConfig(cfg ModuleConfig) bool {
	return cfg.Fields == nil
}

// CreateConfigFile creates a configuration file with default settings.
func CreateConfigFile(configPath string) error {
	// Ensure the directory exists before creating the config file
	dir := filepath.Dir(configPath)
	if err := EnsureDirectoryExists(dir); err != nil {
		return fmt.Errorf("CreateConfigFile: failed to ensure directory %s exists: %w", dir, err)
	}

	defaultConfig := struct {
		Global  GlobalConfig            `json:"global"`
		Modules map[string]ModuleConfig `json:"modules"`
	}{
		Global: GlobalConfig{
			DatabasePath: "",
			Language:     "", // Set to empty string to force system language detection on first run
		},
		Modules: make(map[string]ModuleConfig),
	}

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("CreateConfigFile: failed to marshal default config data: %w", err)
	}

	if err = os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("CreateConfigFile: failed to write default config file %s: %w", configPath, err)
	}

	return nil
}
