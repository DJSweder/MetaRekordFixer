// common/language_manager.go

package common

import (
	"MetaRekordFixer/locales"
	"strings"
	"sync"
)

type LanguageItem struct {
	Code string
	Name string
}

// languageMutex ensures thread safety when setting the application language
var languageMutex sync.Mutex

// DetectAndSetLanguage sets the application language based on the following priorities:
// This function is thread-safe due to the use of languageMutex
// 1. Configured language (if configMgr is available and language is set and valid).
// 2. System language (if configMgr is available and system lang is supported, then save to config).
// 3. Fallback to English (if configMgr is available, save to config).
// If configMgr is nil, it attempts system language then fallback to English, without saving.
func DetectAndSetLanguage(configMgr *ConfigManager, logger *Logger) string {
	languageMutex.Lock()
	defer languageMutex.Unlock()
	supportedLangs := locales.GetAvailableLanguages()
	logger.Info("Supported languages: %v", supportedLangs)

	// Handle case where ConfigManager is not available (e.g., config file failed to load/create)
	if configMgr == nil {
		logger.Warning("ConfigManager is nil. Attempting to use system language or fallback to English.")
		systemLang := getSystemLanguage()
		if len(systemLang) >= 2 {
			systemLang = systemLang[:2] // Use only the primary language subtag (e.g., "en" from "en-US")
		}
		logger.Info("Detected system language (without config): %s", systemLang)

		for _, lang := range supportedLangs {
			if strings.EqualFold(systemLang, lang) {
				logger.Info("Using system language (without config): %s", lang)
				if err := locales.LoadTranslations(lang); err != nil {
					logger.Error("Failed to load system language translations (without config) for %s: %v", lang, err)
					// Fall through to English if loading fails
				} else {
					return lang
				}
			}
		}

		logger.Info("Using fallback language (without config): en")
		if err := locales.LoadTranslations("en"); err != nil {
			logger.Error("Failed to load fallback translations (without config) for en: %v", err)
		}
		return "en"
	}

	// Proceed with ConfigManager available
	globalConfig := configMgr.GetGlobalConfig()
	configLang := strings.ToLower(globalConfig.Language)
	logger.Info("Current configuration language: '%s'", configLang)

	// 1. Check if language is configured and valid.
	// An empty string in configLang signifies that the language has not been set yet.
	if configLang != "" {
		for _, lang := range supportedLangs {
			if strings.EqualFold(configLang, lang) {
				logger.Info("Loading configured language: %s", lang)
				if err := locales.LoadTranslations(lang); err != nil {
					logger.Error("Failed to load translations for configured language %s: %v. Attempting system language.", lang, err)
					// Fall through to system language detection
				} else {
					logger.Info("Successfully loaded translations for %s", lang)
					return lang
				}
			}
		}
		logger.Warning("Configured language '%s' is not supported or invalid. Attempting system language.", configLang)
	} else {
		logger.Info("No language configured. Attempting system language detection.")
	}

	// 2. Try system language if no valid configuration exists or configLang was empty
	systemLang := getSystemLanguage()
	if len(systemLang) >= 2 {
		systemLang = systemLang[:2] // Use only the primary language subtag
	}
	logger.Info("Detected system language: %s", systemLang)

	for _, lang := range supportedLangs {
		if strings.EqualFold(systemLang, lang) {
			logger.Info("Using system language: %s", lang)
			if err := locales.LoadTranslations(lang); err != nil {
				logger.Error("Failed to load system language translations for %s: %v. Falling back to English.", lang, err)
				// Fall through to English if loading fails
			} else {
				logger.Info("Saving system language '%s' to configuration.", lang)
				globalConfig.Language = lang
				if err := configMgr.SaveGlobalConfig(globalConfig); err != nil {
					logger.Error("Failed to save system language '%s' to config: %v", lang, err)
				}
				return lang
			}
		}
	}
	logger.Info("System language '%s' is not supported or detection failed. Falling back to English.", systemLang)

	// 3. Fallback to English if system language is not supported or previous steps failed
	logger.Info("Using fallback language: en")
	if err := locales.LoadTranslations("en"); err != nil {
		// This is critical, as English is the ultimate fallback.
		logger.Error("CRITICAL: Failed to load fallback translations for 'en': %v", err)
		// Even if loading 'en' fails, we still return 'en' as the code,
		// and the UI will likely show translation keys instead of text.
	}

	logger.Info("Saving fallback language 'en' to configuration.")
	globalConfig.Language = "en"
	if err := configMgr.SaveGlobalConfig(globalConfig); err != nil {
		logger.Error("Failed to save fallback language 'en' to config: %v", err)
	}
	return "en"
}

// GetAvailableLanguages returns a list of available languages
func GetAvailableLanguages() []LanguageItem {
	langs := locales.GetAvailableLanguages()
	var items []LanguageItem

	for _, code := range langs {
		name := locales.Translate("settings.lang." + code)
		if strings.HasPrefix(name, "settings.lang.") {
			name = code
		}
		items = append(items, LanguageItem{Code: code, Name: name})
	}
	return items
}
