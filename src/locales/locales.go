package locales

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

//go:embed en/translations.json
//go:embed cs/translations.json
//go:embed de/translations.json
var translationsFS embed.FS

// translations stores the loaded translations in memory
var translations map[string]string

// translationsMutex ensures thread safety when working with translations
var translationsMutex sync.RWMutex

// LoadTranslations loads the translation file for the specified language.
// This function is thread-safe due to the use of translationsMutex.
// It reads the JSON translation file and stores the translations in memory.
// Returns an error if the file cannot be loaded or parsed.
func LoadTranslations(lang string) error {
	translationsMutex.Lock()
	defer translationsMutex.Unlock()
	// Log language being loaded
	// log.Printf("Loading translations for language: %s", lang)

	// Construct the path to the translation file
	filePath := lang + "/translations.json"
	// log.Printf("Translation file path: %s", filePath)

	// Read the file content
	data, err := translationsFS.ReadFile(filePath)
	if err != nil {
		log.Printf("Error loading translation file: %v", err)
		return fmt.Errorf("failed to load translation file: %v", err)
	}

	// Parse the JSON content into the translations map
	err = json.Unmarshal(data, &translations)
	if err != nil {
		log.Printf("Error parsing translation file: %v", err)
		return fmt.Errorf("failed to parse translation file: %v", err)
	}

	// Log loaded translations for debugging
	// log.Printf("Loaded translations: %v", translations)

	return nil
}

// Translate returns the translated string for the given key.
// This function is thread-safe due to the use of translationsMutex.
// If the translation is not found, returns the key itself.
func Translate(key string) string {
	translationsMutex.RLock()
	defer translationsMutex.RUnlock()
	if translation, ok := translations[key]; ok {
		return translation
	}
	return key
}

// GetAvailableLanguages returns a list of all available languages
// from the embedded filesystem. Returns ["en"] as fallback on error.
func GetAvailableLanguages() []string {
	var langs []string
	entries, err := translationsFS.ReadDir(".")
	if err != nil {
		return []string{"en"} // Fallback to English on error
	}
	for _, entry := range entries {
		if entry.IsDir() {
			langs = append(langs, entry.Name())
		}
	}
	return langs
}
