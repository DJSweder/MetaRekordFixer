// main.go

package main

import (
	"os"

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
	modules      []*moduleInfo
	logger       *common.Logger
	errorHandler *common.ErrorHandler
	tabContainer *container.AppTabs
}
// moduleInfo holds information about a module.
type moduleInfo struct {
	module   common.Module
	tabItem  *container.TabItem
	isLoaded bool
	createFn func() common.Module
}

// NewRekordboxTools initializes the main application with proper logging, theme, and window setup.
func NewRekordboxTools() *RekordboxTools {
	// Initialize logger first
	logPath := common.JoinPaths(os.Getenv("APPDATA"), "MetaRekordFixer", "log", "metarekordfixer.log")
	logMaxSizeMB := 10
	logMaxAgeDays := 7
	logger, err := common.NewLogger(logPath, logMaxSizeMB, logMaxAgeDays)
	if err != nil {
		os.Exit(1)
	}

	// Create and set up our Fyne application
	fyneApp := app.NewWithID("com.example.metarekordfixer")
	fyneApp.SetIcon(assets.ResourceAppLogo)
	fyneApp.Settings().SetTheme(theme.NewCustomTheme())

	// Setup configuration
	configPath := getConfigPath()
	if err := common.EnsureDirectoryExists(common.JoinPaths(os.Getenv("APPDATA"), "MetaRekordFixer")); err != nil {
		logger.Error("Failed to create config directory: %v", err)
		os.Exit(1)
	}

	configMgr, err := common.NewConfigManager(configPath)
	if err != nil {
		if err := common.CreateConfigFile(configPath); err != nil {
			logger.Error("Failed to create config file: %v", err)
			os.Exit(1)
		}
		configMgr, err = common.NewConfigManager(configPath)
		if err != nil {
			logger.Error("Failed to initialize config manager: %v", err)
			os.Exit(1)
		}
	}

	// Initialize localization
	common.DetectAndSetLanguage(configMgr, logger)

	// Create the main window with localized title
	mainWindow := fyneApp.NewWindow(locales.Translate("main.app.title"))
	mainWindow.Resize(fyne.NewSize(1000, 700))

	// Log application startup
	logger.Info("%s", locales.Translate("main.log.appstart"))

	// Initialize error handler
	errorHandler := common.NewErrorHandler(logger, mainWindow)

	return &RekordboxTools{
		app:          fyneApp,
		mainWindow:   mainWindow,
		configMgr:    configMgr,
		dbManager:    nil, // Will be initialized on demand
		logger:       logger,
		errorHandler: errorHandler,
	}
}

// Run starts the application, initializes modules, builds the GUI, and runs the main event loop.
func (rt *RekordboxTools) Run() {
	rt.initModules()
	rt.createMainContent()
	rt.mainWindow.ShowAndRun()
	// Ensure database connections are properly closed
	if rt.dbManager != nil {
		if err := rt.dbManager.Finalize(); err != nil {
			rt.logger.Error("Error finalizing database: %v", err)
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
		// Create a new DBManager instance without connecting to the database
		dbManager, err := common.NewDBManager(rt.configMgr.GetGlobalConfig().DatabasePath, rt.logger, rt.errorHandler)
		if err != nil {
			rt.logger.Error("Failed to initialize DBManager: %v", err)
		}
		rt.dbManager = dbManager
	}
	return rt.dbManager
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

// main is the entry point. It ensures config and language, then starts the RekordboxTools app.
func main() {
	rt := NewRekordboxTools()
	rt.Run()
}
