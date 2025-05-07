// common/language.go

package common

import (
	"MetaRekordFixer/locales"
	"strings"
	"syscall"
	"unsafe"
)

// DetectAndSetLanguage sets the application language based on the following priorities:
func DetectAndSetLanguage(configMgr *ConfigManager, logger *Logger) string {
	// 1. Try loading from configuration
	globalConfig := configMgr.GetGlobalConfig()
	configLang := strings.ToLower(globalConfig.Language)

	// 2. Check if the language from the configuration is supported
	supportedLangs := locales.GetAvailableLanguages()

	// Use language if it's in the configuration and supported
	if configLang != "" {
		for _, lang := range supportedLangs {
			if configLang == lang {
				logger.Info(locales.Translate("Loaded language from configuration: %s"), configLang)
				return configLang
			}
		}
	}

	// 3. Try to detect the system language
	systemLang := getSystemLanguage()
	if len(systemLang) >= 2 {
		systemLang = systemLang[:2] // Use only the first two characters (e.g., "en", "cs", "de")
	}

	// If the system language is supported, use it and save it to the configuration
	if systemLang != "" {
		for _, lang := range supportedLangs {
			if systemLang == lang {
				logger.Info(locales.Translate("Loaded language from system: %s"), systemLang)
				globalConfig.Language = systemLang
				configMgr.SaveGlobalConfig(globalConfig)
				return systemLang
			}
		}
	}

	// 4. Fallback to default language (English)
	logger.Info(locales.Translate("Set default language - English"))
	globalConfig.Language = "en"
	configMgr.SaveGlobalConfig(globalConfig)
	return "en"
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
