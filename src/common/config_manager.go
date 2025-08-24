// common/config_manager.go
// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains configuration management functionality.

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

// GlobalConfig holds global application settings that are shared across all modules.
// These settings typically include application-wide preferences and configurations.
type GlobalConfig struct {
	DatabasePath string
	Language     string
}

// ConfigManager handles loading, saving, and managing application configuration.
// It provides thread-safe access to both global and module-specific settings.
type ConfigManager struct {
	configPath   string
	globalConfig GlobalConfig
	cfg          *Cfg // Typed configuration structure
	mutex        sync.Mutex
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
		configPath: configPath,
		cfg:        &Cfg{},
	}

	// Try to load the configuration. An error is not critical here if the file is simply new and empty.
	// A default config will be created on the first save if necessary.
	if err := mgr.loadConfig(); err != nil {
		// If the file doesn't exist, it means LocateOrCreatePath failed, which is a critical error.
		// For other errors like malformed JSON, we log it but continue, as a new config will be saved.
		CaptureEarlyLog(SeverityInfo, "Creating a new configuration file '%s' is necessary because it does not exist in the usual location.: %v", configPath, err)
	}

	// Try to load typed configuration
	if err := mgr.LoadCfg(); err != nil {
		// If typed config doesn't exist, it will be created on first save
		CaptureEarlyLog(SeverityInfo, "Typed configuration not found, will create on first save: %v", err)
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

// SaveGlobalConfig updates and saves the global configuration using the typed configuration system.
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

	// Also update typed config if it exists
	if mgr.cfg != nil {
		mgr.cfg.Global.DatabasePath = config.DatabasePath
		mgr.cfg.Global.Language = config.Language
	}
	mgr.mutex.Unlock()

	return mgr.SaveCfg()
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
		Global GlobalConfig `json:"global"`
	}
	err = json.Unmarshal(data, &configData)
	if err != nil {
		return fmt.Errorf("ConfigManager.loadConfig: failed to unmarshal config data from %s: %w", mgr.configPath, err)
	}

	mgr.globalConfig = configData.Global

	return nil
}

// LoadCfg loads the typed configuration from the configuration file.
// This is the primary configuration loading method used by the application.
func (mgr *ConfigManager) LoadCfg() error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	data, err := os.ReadFile(mgr.configPath)
	if err != nil {
		return err
	}

	var cfg Cfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("ConfigManager.LoadCfg: failed to unmarshal config data: %w", err)
	}

	mgr.cfg = &cfg
	return nil
}

// SaveCfg saves the typed configuration to the configuration file.
// This is the primary configuration saving method used by the application.
func (mgr *ConfigManager) SaveCfg() error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	if mgr.cfg == nil {
		return fmt.Errorf("ConfigManager.SaveCfg: no typed configuration loaded")
	}

	data, err := json.MarshalIndent(mgr.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("ConfigManager.SaveCfg: failed to marshal config data: %w", err)
	}

	if err = os.WriteFile(mgr.configPath, data, 0644); err != nil {
		return fmt.Errorf("ConfigManager.SaveCfg: failed to write config file %s: %w", mgr.configPath, err)
	}
	return nil
}

// isEmptyModuleConfig checks if a module configuration is empty or contains only empty fields
func isEmptyModuleConfig(config interface{}) bool {
	if config == nil {
		return true
	}

	// Use reflection to check if all FieldCfg values are empty
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return true
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return true
	}

	// Check all fields of the struct
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() == reflect.Struct {
			// Assume this is a FieldCfg struct, check its Value field
			valueField := field.FieldByName("Value")
			if valueField.IsValid() && valueField.Kind() == reflect.String {
				if valueField.String() != "" {
					return false // Found non-empty value
				}
			}
		}
	}

	return true // All fields are empty
}

// GetModuleCfg returns configuration for a specific module in typed format
func (mgr *ConfigManager) GetModuleCfg(moduleType string, moduleName string) (interface{}, error) {
	// If typed configuration is not loaded, try to load it first
	if mgr.cfg == nil {
		if err := mgr.LoadCfg(); err != nil {
			// If loading fails, create empty typed config and use defaults
			mgr.cfg = &Cfg{
				Global: GlobalCfg{
					DatabasePath: "",
					Language:     "",
				},
				Modules: ModuleCfgs{},
			}
		}
	}

	// Get the appropriate module configuration based on type
	var moduleConfig interface{}
	switch strings.ToLower(moduleType) {
	case ModuleKeyFlacFixer:
		moduleConfig = mgr.cfg.Modules.FlacFixer
	case ModuleKeyFormatConverter:
		moduleConfig = mgr.cfg.Modules.FormatConverter
	case ModuleKeyDatesMaster:
		moduleConfig = mgr.cfg.Modules.DatesMaster
	case ModuleKeyDataDuplicator:
		moduleConfig = mgr.cfg.Modules.DataDuplicator
	case ModuleKeyFormatUpdater:
		moduleConfig = mgr.cfg.Modules.FormatUpdater
	default:
		return nil, fmt.Errorf("unknown module type: %s", moduleType)
	}

	// Check if configuration is empty and return default if needed
	if isEmptyModuleConfig(moduleConfig) {
		defaultConfig := GetDefaultModuleCfg(strings.ToLower(moduleType))
		if defaultConfig != nil {
			// Save the default configuration for future use
			mgr.SaveModuleCfg(moduleType, moduleName, defaultConfig)
			return defaultConfig, nil
		}
	}

	return moduleConfig, nil
}

// SaveModuleCfg saves configuration for a specific module in typed format
func (mgr *ConfigManager) SaveModuleCfg(moduleType string, moduleName string, config interface{}) error {
	if mgr.cfg == nil {
		return fmt.Errorf("typed configuration not loaded")
	}

	// Update the appropriate module configuration based on type
	switch strings.ToLower(moduleType) {
	case ModuleKeyFlacFixer:
		if cfg, ok := config.(FlacFixerCfg); ok {
			mgr.cfg.Modules.FlacFixer = cfg
		} else {
			return fmt.Errorf("invalid configuration type for flacfixer")
		}
	case ModuleKeyFormatConverter:
		if cfg, ok := config.(FormatConverterCfg); ok {
			mgr.cfg.Modules.FormatConverter = cfg
		} else {
			return fmt.Errorf("invalid configuration type for formatconverter")
		}
	case ModuleKeyDatesMaster:
		if cfg, ok := config.(DatesMasterCfg); ok {
			mgr.cfg.Modules.DatesMaster = cfg
		} else {
			return fmt.Errorf("invalid configuration type for datesmaster")
		}
	case ModuleKeyDataDuplicator:
		if cfg, ok := config.(DataDuplicatorCfg); ok {
			mgr.cfg.Modules.DataDuplicator = cfg
		} else {
			return fmt.Errorf("invalid configuration type for dataduplicator")
		}
	case ModuleKeyFormatUpdater:
		if cfg, ok := config.(FormatUpdaterCfg); ok {
			mgr.cfg.Modules.FormatUpdater = cfg
		} else {
			return fmt.Errorf("invalid configuration type for formatupdater")
		}
	default:
		return fmt.Errorf("unknown module type: %s", moduleType)
	}

	return mgr.SaveCfg()
}

// CreateCfgFile creates a configuration file with default settings for typed configuration
func CreateCfgFile(cfgPath string) error {
	// Ensure the directory exists before creating the config file
	dir := filepath.Dir(cfgPath)
	if err := EnsureDirectoryExists(dir); err != nil {
		return fmt.Errorf("CreateCfgFile: failed to ensure directory %s exists: %w", dir, err)
	}

	// Attempt to detect Rekordbox database path
	detectedDbPath, _ := DetectRekordboxDatabase() // Ignore error in CreateCfgFile, empty path is acceptable

	defaultConfig := Cfg{
		Global: GlobalCfg{
			DatabasePath: detectedDbPath,
			Language:     "",
		},
		Modules: ModuleCfgs{
			FormatConverter: FormatConverterCfg{},
			DatesMaster:     DatesMasterCfg{},
			FlacFixer:       FlacFixerCfg{},
			DataDuplicator:  DataDuplicatorCfg{},
			FormatUpdater:   FormatUpdaterCfg{},
		},
	}

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("CreateCfgFile: failed to marshal default config data: %w", err)
	}

	if err = os.WriteFile(cfgPath, data, 0644); err != nil {
		return fmt.Errorf("CreateCfgFile: failed to write default config file %s: %w", cfgPath, err)
	}
	return nil
}
