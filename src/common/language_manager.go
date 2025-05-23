// common/language_manager.go

package common

import (
	"MetaRekordFixer/locales"
	"strings"
	"syscall"
	"unsafe"
)

type LanguageItem struct {
	Code string
	Name string
}

// DetectAndSetLanguage sets the application language based on the following priorities:
// DetectAndSetLanguage sets the application language based on the following priorities:
func DetectAndSetLanguage(configMgr *ConfigManager, logger *Logger) string {
	globalConfig := configMgr.GetGlobalConfig()
	configLang := strings.ToLower(globalConfig.Language)
	supportedLangs := locales.GetAvailableLanguages()

	logger.Info("Supported languages: %v", supportedLangs)
	logger.Info("Current configuration language: %s", configLang)

	// 1. Check if language is configured
	if configLang != "" {
		for _, lang := range supportedLangs {
			if strings.EqualFold(configLang, lang) {
				logger.Info("Loading configured language: %s", lang)
				if err := locales.LoadTranslations(lang); err != nil {
					logger.Error("Failed to load translations for %s: %v", lang, err)
				} else {
					logger.Info("Successfully loaded translations for %s", lang)
					return lang
				}
			}
		}
	}

	// 2. Try system language if no configuration exists
	systemLang := getSystemLanguage()
	if len(systemLang) >= 2 {
		systemLang = systemLang[:2]
	}
	logger.Info("Detected system language: %s", systemLang)

	for _, lang := range supportedLangs {
		if strings.EqualFold(systemLang, lang) {
			logger.Info("Using system language: %s", lang)
			if err := locales.LoadTranslations(lang); err != nil {
				logger.Error("Failed to load system language translations: %v", err)
			} else {
				globalConfig.Language = lang
				if err := configMgr.SaveGlobalConfig(globalConfig); err != nil {
					logger.Error("Failed to save language config: %v", err)
				}
				return lang
			}
		}
	}

	// 3. Fallback to English if system language is not supported
	logger.Info("Using fallback language: en")
	if err := locales.LoadTranslations("en"); err != nil {
		logger.Error("Failed to load fallback translations: %v", err)
	}
	globalConfig.Language = "en"
	if err := configMgr.SaveGlobalConfig(globalConfig); err != nil {
		logger.Error("Failed to save fallback language config: %v", err)
	}
	return "en"
}

// GetAvailableLanguages vrací seznam dostupných jazyků
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

// getSystemLanguage retrieves the system language on Windows via kernel32.dll calls.
func getSystemLanguage() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getUserDefaultLocaleName := kernel32.NewProc("GetUserDefaultLocaleName")

	// Buffer to store the locale name
	localeName := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH is 85
	getUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&localeName[0])), uintptr(len(localeName)))
	return strings.ToLower(syscall.UTF16ToString(localeName))
}
