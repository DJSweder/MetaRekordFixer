// main.go

package main

import (
	"fmt"
	"os"
	"runtime/debug"

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

// NRT11
// RekordboxTools is the main application structure.
type RekordboxTools struct {
	app             fyne.App
	mainWindow      fyne.Window
	configMgr       *common.ConfigManager
	dbManager       *common.DBManager
	modules         []*moduleInfo
	logger          *common.Logger
	errorHandler    *common.ErrorHandler
	tabContainer    *container.AppTabs
	configInitError error // Store any error that occurs during config initialization (Phase 1 Refactor)
}

// moduleInfo holds information about a module.
type moduleInfo struct {
	module   common.Module
	tabItem  *container.TabItem
	isLoaded bool
	createFn func() common.Module
}

// NewRekordboxTools initializes the main application with proper logging, theme, and window setup.
// ID_M02_CALL_NEWREKORDBOXTOOLS: Volání funkce NewRekordboxTools, která obsahuje většinu inicializační logiky
func NewRekordboxTools() *RekordboxTools {
	// ID_NRT01_INIT_LOGGER: Inicializace logovacího systému s fallbackem
	// Initialize logger first
	var logger *common.Logger
	var err error
	logMaxSizeMB := 10
	logMaxAgeDays := 7
	
	// Získání cesty k APPDATA pro pozdější použití
	appData := os.Getenv("APPDATA")
	
	// 1. Nejprve zkontrolovat root adresář pro log soubor
	rootLogPath := "metarekordfixer.log"
	if common.FileExists(rootLogPath) {
		// Log soubor existuje v root adresáři, použijeme ho
		logger, err = common.NewLogger(rootLogPath, logMaxSizeMB, logMaxAgeDays)
		if err == nil {
			// Úspěšně inicializován logger v root adresáři
			fmt.Printf("Using existing log file in root directory\n")
		}
	}

	// 2. Pokud není v root, zkontrolovat APPDATA
	if logger == nil && appData != "" {
		appDataLogPath := common.JoinPaths(appData, "MetaRekordFixer", "log", "metarekordfixer.log")
		
		// Kontrola, zda log soubor existuje v APPDATA
		if common.FileExists(appDataLogPath) {
			logger, err = common.NewLogger(appDataLogPath, logMaxSizeMB, logMaxAgeDays)
			if err == nil {
				// Úspěšně inicializován logger v APPDATA
				fmt.Printf("Using existing log file in APPDATA\n")
			}
		} else {
			// Pokus o vytvoření log souboru v APPDATA
			if err := common.EnsureDirectoryExists(common.JoinPaths(appData, "MetaRekordFixer", "log")); err == nil {
				logger, err = common.NewLogger(appDataLogPath, logMaxSizeMB, logMaxAgeDays)
				if err == nil {
					// Úspěšně vytvořen a inicializován logger v APPDATA
					fmt.Printf("Created new log file in APPDATA\n")
				}
			}
		}
	}

	// 3. Pokud stále nemáme logger, fallback do root adresáře
	if logger == nil {
		logger, err = common.NewLogger(rootLogPath, logMaxSizeMB, logMaxAgeDays)
		if err != nil {
			// Kritická chyba - nelze vytvořit logger ani v root adresáři
			fmt.Printf("CRITICAL ERROR: Failed to initialize logger in any location: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created new log file in root directory as fallback\n")
	}
	// ID_NRT02_INIT_FYNE_APP: Vytvoření instance Fyne aplikace
	// ID_NRT03_SET_APP_ICON: Nastavení ikony aplikace
	// ID_NRT04_SET_APP_THEME: Nastavení vizuálního tématu aplikace
	// Create and set up our Fyne application
	fyneApp := app.NewWithID("com.example.metarekordfixer")
	fyneApp.SetIcon(assets.ResourceAppLogo)
	fyneApp.Settings().SetTheme(theme.NewCustomTheme())

	// ID_NRT09_EARLY_RT_STRUCT: Vytvoření hlavní struktury RekordboxTools s loggerem a fyneApp
	// Phase 1 Refactor: Create RekordboxTools instance early with logger and fyneApp.
	// Other managers (configMgr, dbManager, errorHandler) will be initialized later.
	rt := &RekordboxTools{
		app:    fyneApp,
		logger: logger,
	}

	// ID_NRT05_INIT_CONFIG_MGR_WITH_FALLBACK: Inicializace ConfigManager s fallbackem
	var configMgr *common.ConfigManager
	var configInitError error

	// 1. Nejprve zkontrolovat root adresář pro konfigurační soubor
	rootConfigPath := "settings.conf"
	if common.FileExists(rootConfigPath) {
		// Konfigurační soubor existuje v root adresáři, použijeme ho
		configMgr, configInitError = common.NewConfigManager(rootConfigPath)
		if configInitError == nil {
			// Úspěšně inicializován ConfigManager v root adresáři
			rt.logger.Info("Using existing config file in root directory")
		}
	}

	// 2. Pokud není v root, zkontrolovat APPDATA
	if configMgr == nil && appData != "" {
		appDataConfigPath := common.JoinPaths(appData, "MetaRekordFixer", "settings.conf")
		
		// Kontrola, zda konfigurační soubor existuje v APPDATA
		if common.FileExists(appDataConfigPath) {
			configMgr, configInitError = common.NewConfigManager(appDataConfigPath)
			if configInitError == nil {
				// Úspěšně inicializován ConfigManager v APPDATA
				rt.logger.Info("Using existing config file in APPDATA")
			}
		} else {
			// Pokus o vytvoření konfiguračního souboru v APPDATA
			if err := common.EnsureDirectoryExists(common.JoinPaths(appData, "MetaRekordFixer")); err == nil {
				if createErr := common.CreateConfigFile(appDataConfigPath); createErr == nil {
					rt.logger.Info("Created new config file in APPDATA. Attempting to load it.")
					configMgr, configInitError = common.NewConfigManager(appDataConfigPath)
					if configInitError == nil {
						rt.logger.Info("Successfully loaded newly created config from APPDATA")
					} else {
						rt.logger.Warning("Failed to load newly created config from APPDATA: %v", configInitError)
						configMgr = nil // Ensure we fallback
					}
				} else {
					rt.logger.Warning("Failed to create config file in APPDATA: %v", createErr)
				}
			} else {
				rt.logger.Warning("Failed to ensure config directory exists in APPDATA: %v", err)
			}
		}
	}

	// 3. Pokud stále nemáme konfiguraci, fallback do root adresáře
	if configMgr == nil {
		rt.logger.Info("Using local path for configuration as fallback")
		if createErr := common.CreateConfigFile(rootConfigPath); createErr == nil {
			rt.logger.Info("Created new config file in root directory. Attempting to load it.")
			configMgr, configInitError = common.NewConfigManager(rootConfigPath)
			if configInitError != nil {
				rt.logger.Error("Failed to load newly created config from root directory: %v", configInitError)
			} else {
				rt.logger.Info("Successfully loaded newly created config from root directory")
			}
		} else {
			rt.logger.Error("CRITICAL: Failed to create config file in root directory: %v", createErr)
			configInitError = fmt.Errorf("failed to create config file in root directory: %w", createErr)
		}
	}

	rt.configMgr = configMgr
	rt.configInitError = configInitError

	// ID_NRT10_DETECT_SET_LANGUAGE: Detekce a nastavení jazyka aplikace
	// Initialize localization
	if rt.configMgr != nil {
		common.DetectAndSetLanguage(rt.configMgr, rt.logger)
	} else {
		rt.logger.Warning("ConfigManager is not available. Skipping language detection from config. Default language will be used.")
		// Attempt to set a default language (e.g., English) or rely on Fyne's default if common.DetectAndSetLanguage cannot be called with nil configMgr.
		// For now, we assume DetectAndSetLanguage handles nil configMgr gracefully or we'll adjust it in Phase 2.
		// If DetectAndSetLanguage requires a configMgr, we might need a separate call here for a default.
		// For simplicity in Phase 1, we'll just log and rely on Phase 2 to make DetectAndSetLanguage robust.
	}

	// ID_NRT11_CREATE_MAIN_WINDOW: Vytvoření hlavního okna aplikace
	// Create the main window with localized title
	mainWindow := fyneApp.NewWindow(locales.Translate("main.app.title"))
	mainWindow.Resize(fyne.NewSize(1000, 700))

	// Log application startup - moved after rt.errorHandler is set for consistency
	// logger.Info("%s", locales.Translate("main.log.appstart"))

	// ID_NRT12_INIT_ERROR_HANDLER: Inicializace ErrorHandler s přiřazeným oknem
	// Initialize error handler and assign to rt instance
	rt.errorHandler = common.NewErrorHandler(rt.logger, mainWindow) // Use rt.logger
	rt.mainWindow = mainWindow                                      // Assign mainWindow to rt instance

	// Log application startup now that all essential rt components (logger, window, errorHandler) are set.
	rt.logger.Info("%s", locales.Translate("main.log.appstart"))

	// rt.dbManager is already nil by default (from early struct init) or will be set by getDBManager if needed.
	// rt.configMgr is already set.
	// rt.app and rt.logger were set at the beginning.
	return rt
}

// ID_M03_SHOW_MAIN_WINDOW: Zobrazení hlavního okna uživateli (v `main`)
// Run starts the application, initializes modules, builds the GUI, and runs the main event loop.
func (rt *RekordboxTools) Run() {
	// ID_M01_SETUP_PANIC_RECOVERY: Nastavení záchranného handleru pro neočekávané chyby
	// Setup panic recovery for the main application loop
	defer func() {
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack())
			if rt.errorHandler != nil {
				rt.errorHandler.ShowPanicError(r, stackTrace)
			} else if rt.logger != nil {
				// Fallback if errorHandler is somehow nil
				rt.logger.Error("PANIC RECOVERED (ErrorHandler not available): %v\n%s", r, stackTrace)
			}
		}
	}()
	// ID_RUN01_INIT_MODULES: Inicializace definic modulů
	rt.initModules()
	// ID_RUN02_CREATE_MAIN_CONTENT: Vytvoření hlavního obsahu okna
	rt.createMainContent()

	// ID_RUN03_SHOW_WINDOW: Zobrazení hlavního okna
	// Show the main window
	rt.mainWindow.Show()

	// ID_NEW_M03A_SHOW_CONFIG_ERROR_DIALOG: Kontrola inicializačních chyb a zobrazení dialogu
	// Phase 3 Refactor: Check for initialization errors after showing the window
	if rt.configInitError != nil {
		if rt.errorHandler != nil {
			rt.logger.Info("Displaying initialization error dialog for: %v", rt.configInitError) // Log before showing dialog
			rt.errorHandler.ShowInitializationErrorDialog(rt.configInitError)
		} else if rt.logger != nil {
			// Fallback if errorHandler is somehow nil (should not happen in normal operation)
			rt.logger.Error("Initialization error occurred but ErrorHandler is not available to show dialog: %v", rt.configInitError)
		}
	}

	// ID_M04_RUN_FYNE_APP: Spuštění hlavní smyčky Fyne aplikace
	// Run the application event loop
	rt.app.Run() // This blocks until the app exits

	// ID_RUN06_CLEANUP: Úklid zdrojů při ukončení aplikace
	// Ensure database connections are properly closed
	if rt.dbManager != nil {
		if err := rt.dbManager.Finalize(); err != nil {
			rt.logger.Error("%s: %v", locales.Translate("common.err.dbclosing"), err)
		}
	}
}

// initModules prepares module definitions without initializing them
func (rt *RekordboxTools) initModules() {
	rt.modules = []*moduleInfo{
		{
			createFn: func() common.Module {
				m := modules.NewMetadataSyncModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler)
				m.SetDatabaseRequirements(true, false)
				return m
			},
		},
		{
			createFn: func() common.Module {
				m := modules.NewHotCueSyncModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler)
				m.SetDatabaseRequirements(true, true)
				return m
			},
		},
		{
			createFn: func() common.Module {
				m := modules.NewDateSyncModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler)
				m.SetDatabaseRequirements(true, false)
				return m
			},
		},
		{
			createFn: func() common.Module {
				m := modules.NewTracksUpdaterModule(rt.mainWindow, rt.configMgr, rt.getDBManager(), rt.errorHandler)
				m.SetDatabaseRequirements(true, true)
				return m
			},
		},
		{
			createFn: func() common.Module {
				m := modules.NewMusicConverterModule(rt.mainWindow, rt.configMgr, rt.errorHandler)
				m.SetDatabaseRequirements(false, false)
				return m
			},
		},
	}
}

// createModuleTabItem creates a tab item for a module
func (rt *RekordboxTools) createModuleTabItem(info *moduleInfo) *container.TabItem {
	if !info.isLoaded {
		// Create temporary module just to get name and icon
		tempModule := info.createFn()
		dbReqs := tempModule.GetDatabaseRequirements()

		if !dbReqs.NeedsDatabase {
			// Module doesn't need database, create it immediately
			info.module = tempModule
			info.isLoaded = true
		} else {
			// For modules that need database, create placeholder content
			placeholder := container.NewVBox()
			// Return tab with placeholder, real content will be loaded on selection
			return container.NewTabItem(tempModule.GetName(), placeholder)
		}
	}

	return container.NewTabItem(info.module.GetName(), info.module.GetContent())
}

// createMainContent creates the main window content with tabs
func (rt *RekordboxTools) createMainContent() fyne.CanvasObject {
	rt.tabContainer = container.NewAppTabs()

	// First create all tab items
	for _, info := range rt.modules {
		info.tabItem = rt.createModuleTabItem(info)
		rt.tabContainer.Append(info.tabItem)
	}

	// Then select the first tab (metadata_sync) and ensure it's loaded
	if len(rt.tabContainer.Items) > 0 {
		firstTab := rt.tabContainer.Items[0]
		rt.tabContainer.Select(firstTab)

		// Find and load the first module
		for _, info := range rt.modules {
			if info.tabItem == firstTab && !info.isLoaded {
				info.module = info.createFn()
				info.isLoaded = true
				firstTab.Content = info.module.GetContent()
				break
			}
		}
	}

	// Handle tab changes to load modules on demand
	rt.tabContainer.OnSelected = func(tab *container.TabItem) {
		// Find the corresponding module info
		for _, info := range rt.modules {
			if info.tabItem == tab && !info.isLoaded {
				// Create the module
				info.module = info.createFn()
				info.isLoaded = true

				// Update tab content
				tab.Content = info.module.GetContent()
				rt.tabContainer.Refresh()
				break
			}
		}
	}

	rt.tabContainer.SetTabLocation(container.TabLocationTop)

	menuBar := rt.createMenuBar()
	content := container.NewVBox(menuBar, rt.tabContainer)
	rt.mainWindow.SetContent(content)
	return content
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

// ID_M04_LAZY_DB_MANAGER: Lazy inicializace DBManager při prvním použití
// getDBManager returns the dbManager instance, initializing it if necessary.
func (rt *RekordboxTools) getDBManager() *common.DBManager {
	if rt.dbManager == nil {
		// Phase 1 Refactor: Make DBManager initialization non-fatal and handle nil configMgr.
		if rt.configMgr == nil {
			rt.logger.Warning("DBManager: Configuration manager is not available. Cannot get database path.")
			return nil // rt.dbManager remains nil
		}

		dbPath := rt.configMgr.GetGlobalConfig().DatabasePath
		//		if dbPath == "" {
		//			rt.logger.Warning("DBManager: Database path is not configured. DBManager will not be initialized.")
		//			return nil // rt.dbManager remains nil
		//		}

		dbManagerInstance, err := common.NewDBManager(dbPath, rt.logger, rt.errorHandler)
		if err != nil {
			rt.logger.Error("DBManager: Failed to initialize DBManager for path '%s': %v", dbPath, err)
			// rt.dbManager remains nil
		} else {
			rt.dbManager = dbManagerInstance
			rt.logger.Info("DBManager: Initialized for path: %s", dbPath)
		}
	}
	return rt.dbManager
}

// Funkce getConfigPath byla odstraněna a nahrazena přímou implementací algoritmu
// v NewRekordboxTools(), který nejprve kontroluje root adresář, pak APPDATA
// a nakonec fallback do root adresáře.

// main is the entry point. It ensures config and language, then starts the RekordboxTools app.
func main() {
	// ID_M02_CALL_NEWREKORDBOXTOOLS: Volání funkce NewRekordboxTools
	rt := NewRekordboxTools()
	// ID_M03_CALL_RT_RUN: Volání metody Run
	rt.Run()
}
