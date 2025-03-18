// main.go

package main

import (
	"log"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"MetaRekordFixer/assets"
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"MetaRekordFixer/modules"
	"MetaRekordFixer/theme"
	"MetaRekordFixer/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RekordboxTools is the main application structure.
type RekordboxTools struct {
	app          fyne.App
	mainWindow   fyne.Window
	configMgr    *common.ConfigManager
	dbManager    *common.DBManager
	modules      []common.Module
	logger       *log.Logger
	errorHandler *common.ErrorHandler
}

// NewRekordboxTools initializes the main application with proper logging, theme, and window setup.
func NewRekordboxTools() *RekordboxTools {
	// Create and set up our Fyne application
	fyneApp := app.NewWithID("com.example.metarekordfixer")
	fyneApp.SetIcon(assets.ResourceAppLogo)
	fyneApp.Settings().SetTheme(theme.NewCustomTheme()) // Use our custom dark theme

	// Create the main window
	mainWindow := fyneApp.NewWindow(locales.Translate("main.app.title"))
	mainWindow.Resize(fyne.NewSize(1000, 700))

	// Check if the configuration file exists and create it with default values if not
	configPath := getConfigPath()
	configMgr, err := common.NewConfigManager(configPath)
	if err != nil {
		// Create missing directories if they don't exist
		_ = common.EnsureDirectoryExists(common.JoinPaths(os.Getenv("APPDATA"), "MetaRekordFixer"))
		// Create the configuration file with default values
		err = common.CreateConfigFile(configPath)
		if err != nil {
			log.Fatalf("Failed to create configuration file: %v", err)
		}
		configMgr, err = common.NewConfigManager(configPath)
		if err != nil {
			log.Fatalf("Failed to initialize ConfigManager: %v", err)
		}
	}

	// Initialize dbManager only when needed, not during startup
	var dbManager *common.DBManager

	// Initialize logger and error handler
	logger := log.New(os.Stdout, "APP: ", log.LstdFlags)
	errorHandler := common.NewErrorHandler(logger)

	return &RekordboxTools{
		app:          fyneApp,
		mainWindow:   mainWindow,
		configMgr:    configMgr,
		dbManager:    dbManager,
		logger:       logger,
		errorHandler: errorHandler,
	}
}

// Run starts the application, initializes modules, builds the GUI, and runs the main event loop.
func (rt *RekordboxTools) Run() {
	rt.initializeModules()
	rt.createGUI()
	rt.mainWindow.ShowAndRun()
	// Ensure database connections are properly closed
	if rt.dbManager != nil {
		rt.dbManager.Finalize()
	}
}

// initializeModules loads and initializes all the application modules.
func (rt *RekordboxTools) initializeModules() {
	// Every module is initialized with the main window, config manager, error handler, and db manager
	// except MusicConverterModule which doesn't work with database
	rt.modules = []common.Module{
		modules.NewMetadataSyncModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler),
		modules.NewDateSyncModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler),
		modules.NewHotCueSyncModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler),
		modules.NewMusicConverterModule(rt.mainWindow, rt.configMgr, rt.errorHandler),
		modules.NewTracksUpdater(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler),
	}
}

// getDBManager returns the dbManager instance, initializing it if necessary.
func (rt *RekordboxTools) getDBManager() *common.DBManager {
	if rt.dbManager == nil {
		// Create a new DBManager instance without connecting to the database
		dbManager, err := common.NewDBManager(rt.configMgr.GetGlobalConfig().DatabasePath, nil, nil)
		if err != nil {
			rt.logger.Fatalf("Failed to initialize DBManager: %v", err)
		}
		rt.dbManager = dbManager
	}
	return rt.dbManager
}

// createGUI sets up the UI with tabs for each module.
func (rt *RekordboxTools) createGUI() {
	tabs := container.NewAppTabs()
	for _, module := range rt.modules {
		tabs.Append(container.NewTabItemWithIcon(
			module.GetName(),
			module.GetIcon(),
			module.GetContent(),
		))
	}

	tabs.SetTabLocation(container.TabLocationTop)

	menuBar := rt.createMenuBar()
	content := container.NewVBox(menuBar, tabs)
	rt.mainWindow.SetContent(content)
}

// createMenuBar creates a simple horizontal bar with Settings and Help buttons.
func (rt *RekordboxTools) createMenuBar() fyne.CanvasObject {
	settingsButton := widget.NewButton(locales.Translate("settings.win.title"), func() {
		ui.ShowSettingsWindow(rt.mainWindow, rt.configMgr)
	})
	helpButton := widget.NewButton(locales.Translate("main.menu.help"), func() {
		ui.ShowHelpWindow(rt.mainWindow)
	})

	return container.NewHBox(settingsButton, helpButton)
}

// getConfigPath returns the path to the configuration file (settings.conf) in the user's AppData,
// or uses a local fallback if APPDATA is not set.
func getConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		// Fallback to local directory if APPDATA is not available
		return "settings.conf"
	}
	return common.JoinPaths(appData, "MetaRekordFixer", "settings.conf")
}

// detectLanguage determines the application language based on config or system settings.
func detectLanguage(configMgr *common.ConfigManager) string {
	globalConfig := configMgr.GetGlobalConfig()
	configLang := strings.ToLower(globalConfig.Language)
	supportedLangs := getAvailableLanguages()

	log.Printf("Configured language: %s", configLang)
	log.Printf("Supported languages: %v", supportedLangs)

	// If user-specified language is recognized, use it.
	if configLang != "" {
		for _, lang := range supportedLangs {
			if configLang == lang.Code || configLang == strings.ToLower(lang.Name) {
				log.Printf("Using configured language: %s", lang.Code)
				return lang.Code
			}
		}
	}

	systemLang := getSystemLanguage()
	if len(systemLang) >= 2 {
		systemLang = systemLang[:2] // Use only first two letters
	}

	log.Printf("System language: %s", systemLang)

	// If system language is recognized, use it. Otherwise fallback to English.
	if systemLang != "" {
		for _, lang := range supportedLangs {
			if systemLang == lang.Code {
				log.Printf("Using system language: %s", lang.Code)
				return lang.Code
			}
		}
	}

	log.Printf("Falling back to default language: en")
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

// getAvailableLanguages scans the locales directory from the embedded FS and returns a list of available languages.
func getAvailableLanguages() []languageItem {
	langs := locales.GetAvailableLanguages()
	var langItems []languageItem
	for _, code := range langs {
		name := locales.Translate("settings.lang." + code)
		if strings.HasPrefix(name, "settings.lang.") {
			name = code
		}
		langItems = append(langItems, languageItem{Code: code, Name: name})
	}
	return langItems
}

// languageItem is a helper for language detection, storing both code and display name.
type languageItem struct {
	Code string // e.g. "cs", "en", "de"
	Name string // e.g. "Čeština", "English", "Deutsch"
}

// main is the entry point. It ensures config and language, then starts the RekordboxTools app.
func main() {
	configPath := getConfigPath()
	configMgr, err := common.NewConfigManager(configPath)
	if err != nil {
		_ = common.EnsureDirectoryExists(common.JoinPaths(os.Getenv("APPDATA"), "MetaRekordFixer"))
		// Create the configuration file with default values
		err = common.CreateConfigFile(configPath)
		if err != nil {
			log.Fatalf("Failed to create configuration file: %v", err)
		}
		configMgr, err = common.NewConfigManager(configPath)
		if err != nil {
			log.Fatalf("Failed to initialize ConfigManager: %v", err)
		}
	}

	// Load or detect language
	globalConfig := configMgr.GetGlobalConfig()
	language := globalConfig.Language
	if language == "" {
		language = detectLanguage(configMgr)
		globalConfig.Language = language
		configMgr.SaveGlobalConfig(globalConfig)
	}

	// Initialize translations
	err = locales.LoadTranslations(language)
	if err != nil {
		// Fallback to English if translation loading fails
		log.Printf("Failed to load translations for %s: %v", language, err)
		_ = locales.LoadTranslations("en")
	}

	// Start the main application
	app := NewRekordboxTools()
	app.Run()
}
