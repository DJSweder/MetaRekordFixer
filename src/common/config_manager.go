// common/config_manager.go

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"MetaRekordFixer/locales"
)

// GlobalConfig holds global application settings
type GlobalConfig struct {
	DatabasePath string
	Language     string
}

// ModuleConfig defines a configuration structure for modules
type ModuleConfig struct {
	Extra  map[string]string // will be deprecated in the future
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
	FieldType      string // folder, date, checkbox, select, playlist, file
	Required       bool
	DependsOn      string
	ActiveWhen     string
	ValidationType string // exists, valid_date, filled, exists | write
	Value          string
}

// ValidateField validates a single field based on its definition and value
func (f *FieldDefinition) ValidateField(value string) error {
	return nil
}

// NewConfigManager initializes a new configuration manager
func NewConfigManager(configPath string) (*ConfigManager, error) {
	mgr := &ConfigManager{
		configPath:    configPath,
		moduleConfigs: make(map[string]ModuleConfig),
	}

	err := mgr.loadConfig()
	if err != nil {
		mgr.saveConfig()
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
	mgr.globalConfig = config
	mgr.mutex.Unlock()

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
	mgr.moduleConfigs[moduleName] = config
	mgr.mutex.Unlock()
	mgr.saveConfig()
}

// loadConfig loads the configuration from a file
func (mgr *ConfigManager) loadConfig() error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	if !FileExists(mgr.configPath) {
		return fmt.Errorf(locales.Translate("common.config.filenotfound"), mgr.configPath)
	}

	data, err := os.ReadFile(mgr.configPath)
	if err != nil {
		return fmt.Errorf(locales.Translate("common.config.readerr"), err)
	}

	var config struct {
		Global  GlobalConfig            `json:"global"`
		Modules map[string]ModuleConfig `json:"modules"`
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf(locales.Translate("common.config.parseerr"), err)
	}

	mgr.globalConfig = config.Global
	mgr.moduleConfigs = config.Modules

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
		mgr.globalConfig.Language = "en" // Default language
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
		return fmt.Errorf(locales.Translate("common.config.marshalerr"), err)
	}

	// Ensure the directory exists before writing the file
	dir := filepath.Dir(mgr.configPath)
	if err := EnsureDirectoryExists(dir); err != nil {
		return fmt.Errorf("failed to ensure config directory exists: %v", err)
	}

	return os.WriteFile(mgr.configPath, data, 0644)
}

// NewModuleConfig creates a new empty module configuration
func NewModuleConfig() ModuleConfig {
	return ModuleConfig{
		Extra:  make(map[string]string), // Will be deprecated in the future
		Fields: make(map[string]FieldDefinition),
	}
}

// Get retrieves a string value from the module configuration
func (c ModuleConfig) Get(key string, defaultValue string) string {
	if field, exists := c.Fields[key]; exists {
		return field.Value
	}
	// For backward compatibility. Will be deprecated in the future.
	if value, exists := c.Extra[key]; exists {
		return value
	}
	return defaultValue
}

// Set stores a string value in the module configuration. Will be deprecated in the future
func (c *ModuleConfig) Set(key string, value string) {
	c.Extra[key] = value
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

// GetBool retrieves a boolean value from the module configuration
func (c ModuleConfig) GetBool(key string, defaultValue bool) bool {
	if field, exists := c.Fields[key]; exists {
		return field.Value == "true"
	}
	// For backward compatibility. Will be deprecated in the future
	if value, exists := c.Extra[key]; exists {
		return value == "true"
	}
	return defaultValue
}

// SetBool stores a boolean value in the module configuration. Will be deprecated in the future
func (c *ModuleConfig) SetBool(key string, value bool) {
	c.Extra[key] = fmt.Sprintf("%t", value)
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

// SetInt stores an integer value in the module configuration. Will be deprecated in the future
func (c *ModuleConfig) SetInt(key string, value int) {
	c.Extra[key] = fmt.Sprintf("%d", value)
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

// SetFloat stores a float value in the module configuration
func (c *ModuleConfig) SetFloat(key string, value float64) {
	c.Extra[key] = fmt.Sprintf("%f", value)
}

// IsNilConfig checks if a given configuration is nil
func IsNilConfig(cfg ModuleConfig) bool {
	return cfg.Fields == nil
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CreateConfigFile creates a configuration file with default settings.
func CreateConfigFile(configPath string) error {
	// Ensure the directory exists before creating the config file
	dir := filepath.Dir(configPath)
	if err := EnsureDirectoryExists(dir); err != nil {
		return fmt.Errorf("failed to ensure directory exists: %v", err)
	}

	defaultConfig := struct {
		Global  GlobalConfig            `json:"global"`
		Modules map[string]ModuleConfig `json:"modules"`
	}{
		Global: GlobalConfig{
			DatabasePath: "",
			Language:     "en",
		},
		Modules: make(map[string]ModuleConfig),
	}

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf(locales.Translate("common.config.marshalerr"), err)
	}

	return os.WriteFile(configPath, data, 0644)
}
