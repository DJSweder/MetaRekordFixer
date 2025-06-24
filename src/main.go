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

// NewRekordboxTools initializes the main application.
// It sets up logging, configuration, error handling, and the main window.
// Any critical errors during initialization are stored and displayed after the UI is ready.
func NewRekordboxTools() *RekordboxTools {
	// Phase 1: Initialize Logger
	logPath, err := common.LocateOrCreatePath("metarekordfixer.log", "log")
	if err != nil {
		// This is a critical failure, as we cannot log anything without a logger.
		// We capture the error in early log buffer and exit.
		common.CaptureEarlyLog(common.SeverityCritical, "Could not determine or create path for log file: %v", err)
		os.Exit(1)
	}
	logger, err := common.NewLogger(logPath, 10, 7) // 10MB max size, 7 days max age
	if err != nil {
		common.CaptureEarlyLog(common.SeverityCritical, "Could not initialize logger at '%s': %v", logPath, err)
		os.Exit(1)
	}
	logger.Info("Logger initialized successfully at: %s", logPath)

	// Phase 2: Initialize Core Application Components
	// Create and set up our Fyne application
	fyneApp := app.NewWithID("com.example.metarekordfixer")
	fyneApp.SetIcon(assets.ResourceAppLogo)
	fyneApp.Settings().SetTheme(theme.NewCustomTheme())

	// Create the main application struct early with the logger and fyneApp.
	rt := &RekordboxTools{
		app:    fyneApp,
		logger: logger,
	}

	// Phase 3: Initialize Configuration Manager
	configPath, configInitError := common.LocateOrCreatePath("settings.conf", "") // Empty subDir for config at MetaRekordFixer/settings.conf
	if configInitError != nil {
		rt.configInitError = fmt.Errorf("failed to determine path for config file: %w", configInitError)
		logger.Error(rt.configInitError.Error())
		// We proceed without a config manager, the error will be shown to the user in Run().
	} else {
		configMgr, err := common.NewConfigManager(configPath)
		if err != nil {
			rt.configInitError = fmt.Errorf("failed to initialize config manager at '%s': %w", configPath, err)
			logger.Error(rt.configInitError.Error())
		} else {
			rt.configMgr = configMgr
			// Flush any early logs captured during initialization (after ConfigManager is initialized)
			common.FlushEarlyLogs(logger)
			logger.Info("Configuration initialized successfully at: %s", configPath)
		}
	}

	// Phase 4: Initialize Localization
	if rt.configMgr != nil {
		common.DetectAndSetLanguage(rt.configMgr, rt.logger)
	} else {
		rt.logger.Warning("ConfigManager not available. Skipping language detection. Default language will be used.")
	}

	// Phase 5: Create Main Window but do not show it yet
	mainWindow := fyneApp.NewWindow(locales.Translate("main.app.title"))
	mainWindow.Resize(fyne.NewSize(1000, 700))
	rt.mainWindow = mainWindow

	// Phase 6: Initialize ErrorHandler and log application start
	rt.errorHandler = common.NewErrorHandler(rt.logger, rt.mainWindow)
	rt.logger.Info("%s", locales.Translate("main.log.appstart"))

	// rt.dbManager is already nil by default (from early struct init) or will be set by getDBManager if needed.
	// rt.configMgr is already set.
	// rt.app and rt.logger were set at the beginning.
	return rt
}

// Run starts the application, initializes modules, builds the GUI, and runs the main event loop.
func (rt *RekordboxTools) Run() {
	// Setup panic recovery for the main application loop.
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

	// Initialize modules and create the main window content.
	rt.initModules()
	rt.createMainContent()

	// Show the main window.
	rt.mainWindow.Show()

	// Handle any errors that occurred during initialization, now that the window is visible.
	if rt.configInitError != nil {
		rt.logger.Info("Initialization error occurred: %v", rt.configInitError)
		// We don't show a dialog for initialization errors anymore, just log them
	}

	// Run the application event loop.
	rt.app.Run() // This blocks until the app exits.

	// Cleanup on exit.
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

// getDBManager returns the dbManager instance, initializing it if necessary.
func (rt *RekordboxTools) getDBManager() *common.DBManager {
	if rt.dbManager == nil {
		// DBManager initialization is non-fatal and handles nil configMgr.
		if rt.configMgr == nil {
			rt.logger.Warning("DBManager: Configuration manager is not available. Cannot get database path.")
			return nil
		}

		dbPath := rt.configMgr.GetGlobalConfig().DatabasePath
		dbManagerInstance, err := common.NewDBManager(dbPath, rt.logger, rt.errorHandler)
		if err != nil {
			rt.logger.Error("DBManager: Failed to initialize for path '%s': %v", dbPath, err)
		} else {
			rt.dbManager = dbManagerInstance
			rt.logger.Info("DBManager: Initialized for path: %s", dbPath)
		}
	}
	return rt.dbManager
}

// main is the entry point. It initializes and runs the RekordboxTools application.
func main() {
	rt := NewRekordboxTools()
	rt.Run()
}
