// common/config_manager.go

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// GlobalConfig holds global application settings
type GlobalConfig struct {
	DatabasePath string
	Language     string
}

// ModuleConfig defines a configuration structure for modules
type ModuleConfig struct {
	Fields map[string]FieldDefinition
}

// ConfigManager handles application configuration
type ConfigManager struct {
	configPath    string
	globalConfig  GlobalConfig
	moduleConfigs map[string]ModuleConfig
	mutex         sync.Mutex
}

// FieldDefinition defines validation rules for a configuration field
type FieldDefinition struct {
	FieldType         string // folder, date, checkbox, select, playlist, file
	Required          bool
	DependsOn         string
	ActiveWhen        string
	ValidationType    string // exists, valid_date, filled, exists | write
	Value             string
	ValidateOnActions []string // list of actions for selected validation (eg. for modules with more functions with separated starting)
}

// NewConfigManager initializes a new configuration manager
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

// GetGlobalConfig returns the global configuration
func (mgr *ConfigManager) GetGlobalConfig() GlobalConfig {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	return mgr.globalConfig
}

// SaveGlobalConfig saves the global configuration
func (mgr *ConfigManager) SaveGlobalConfig(config GlobalConfig) error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	mgr.globalConfig = config

	return mgr.saveConfig()
}

// GetModuleConfig retrieves a module's configuration
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

// SaveModuleConfig saves a module's configuration
func (mgr *ConfigManager) SaveModuleConfig(moduleName string, config ModuleConfig) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	mgr.moduleConfigs[moduleName] = config
	mgr.saveConfig()
}

// loadConfig loads the configuration from a file
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

// saveConfig saves the configuration to a file
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

// NewModuleConfig creates a new empty module configuration
func NewModuleConfig() ModuleConfig {
	return ModuleConfig{
		Fields: make(map[string]FieldDefinition),
	}
}

// Get retrieves a string value from the module configuration
func (c ModuleConfig) Get(key string, defaultValue string) string {
	if field, exists := c.Fields[key]; exists {
		return field.Value
	}
	return defaultValue
}

// Set stores a string value in the module configuration
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

func (c *ModuleConfig) SetWithDependency(key string, value string, fieldType string, required bool, dependsOn string, activeWhen string, validationType string) {
	c.SetWithDefinition(key, value, fieldType, required, validationType)
	if field, exists := c.Fields[key]; exists {
		field.DependsOn = dependsOn
		field.ActiveWhen = activeWhen
		c.Fields[key] = field
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

func (c *ModuleConfig) SetBoolWithDependency(key string, value bool, required bool, dependsOn string, activeWhen string, validationType string) {
	c.SetBoolWithDefinition(key, value, required, validationType)
	if field, exists := c.Fields[key]; exists {
		field.DependsOn = dependsOn
		field.ActiveWhen = activeWhen
		c.Fields[key] = field
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

func (c *ModuleConfig) SetIntWithDependency(key string, value int, required bool, dependsOn string, activeWhen string) {
	c.SetIntWithDefinition(key, value, required)
	if field, exists := c.Fields[key]; exists {
		field.DependsOn = dependsOn
		field.ActiveWhen = activeWhen
		c.Fields[key] = field
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
