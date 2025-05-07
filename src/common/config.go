// common/config.go

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
	Extra map[string]string
}

// ConfigManager handles application configuration
type ConfigManager struct {
	configPath    string
	globalConfig  GlobalConfig
	moduleConfigs map[string]ModuleConfig
	mutex         sync.Mutex
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
		if config.Extra == nil {
			config.Extra = make(map[string]string)
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
		Extra: make(map[string]string),
	}
}

// Get retrieves a string value from the module configuration
func (c ModuleConfig) Get(key string, defaultValue string) string {
	if value, exists := c.Extra[key]; exists {
		return value
	}
	return defaultValue
}

// Set stores a string value in the module configuration
func (c *ModuleConfig) Set(key string, value string) {
	c.Extra[key] = value
}

// GetBool retrieves a boolean value from the module configuration
func (c ModuleConfig) GetBool(key string, defaultValue bool) bool {
	if value, exists := c.Extra[key]; exists {
		return value == "true"
	}
	return defaultValue
}

// SetBool stores a boolean value in the module configuration
func (c *ModuleConfig) SetBool(key string, value bool) {
	c.Extra[key] = fmt.Sprintf("%t", value)
}

// GetInt retrieves an integer value from the module configuration
func (c ModuleConfig) GetInt(key string, defaultValue int) int {
	if value, exists := c.Extra[key]; exists {
		intValue, err := parseInt(value)
		if err == nil {
			return intValue
		}
	}
	return defaultValue
}

// SetInt stores an integer value in the module configuration
func (c *ModuleConfig) SetInt(key string, value int) {
	c.Extra[key] = fmt.Sprintf("%d", value)
}

// GetFloat retrieves a float value from the module configuration
func (c ModuleConfig) GetFloat(key string, defaultValue float64) float64 {
	if value, exists := c.Extra[key]; exists {
		floatValue, err := parseFloat(value)
		if err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// SetFloat stores a float value in the module configuration
func (c *ModuleConfig) SetFloat(key string, value float64) {
	c.Extra[key] = fmt.Sprintf("%f", value)
}

// IsNilConfig checks if a given configuration is nil
func IsNilConfig(cfg ModuleConfig) bool {
	return cfg.Extra == nil
}

// parseInt safely parses an integer string
func parseInt(value string) (int, error) {
	var parsedValue int
	_, err := fmt.Sscanf(value, "%d", &parsedValue)
	return parsedValue, err
}

// parseFloat safely parses a float string
func parseFloat(value string) (float64, error) {
	var parsedValue float64
	_, err := fmt.Sscanf(value, "%f", &parsedValue)
	return parsedValue, err
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
