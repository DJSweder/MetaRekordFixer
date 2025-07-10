// Package modules provides functionality for different modules in the MetaRekordFixer application.
// This file contains the DateSyncModule implementation for synchronizing dates in the Rekordbox database.

package modules

import (
	"fmt"
	"strings"
	"time"

	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FolderEntryType defines the type of folder entry list used in the dynamic UI components.
// It distinguishes between custom date folders and excluded folders.
type FolderEntryType int

const (
	// CustomDateFolder represents a custom date folder in the dynamic entry list
	CustomDateFolder FolderEntryType = iota
	// ExcludedFolder represents an excluded folder in the dynamic entry list
	ExcludedFolder

	// maxFolderEntries represents the maximum number of folder entries allowed in each list
	maxFolderEntries = 6
)

// DateSyncModule implements a module for synchronizing dates in the Rekordbox database.
// It provides functionality to set standard dates based on release dates or custom dates for specific folders.
type DateSyncModule struct {
	*common.ModuleBase
	dbMgr                  *common.DBManager
	calendarBtn            *widget.Button
	customDateContainer    *fyne.Container
	customDateFoldersEntry []*widget.Entry
	customDateUpdateBtn    *widget.Button
	datePickerContainer    *fyne.Container
	datePickerEntry        *widget.Entry
	excludeFoldersCheck    *widget.Check
	excludedFoldersEntry   []*widget.Entry
	foldersContainer       *fyne.Container
	standardUpdateBtn      *widget.Button
}

// CustomCalendar implements a custom calendar widget for date selection.
// It provides a user-friendly interface for selecting dates with month and year navigation.
type CustomCalendar struct {
	widget.BaseWidget
	currentYear  int
	currentMonth time.Month
	daysGrid     *fyne.Container
	monthSelect  *widget.Select
	onSelected   func(time.Time)
	yearSelect   *widget.Select
}

// NewCustomCalendar creates a new custom calendar widget with the specified callback function.
// The callback function is called when a date is selected.
// Returns a new CustomCalendar instance initialized with the current date.
func NewCustomCalendar(callback func(time.Time)) *CustomCalendar {
	c := &CustomCalendar{
		onSelected: callback,
		daysGrid:   container.New(layout.NewGridLayout(7)),
	}

	c.ExtendBaseWidget(c)
	now := time.Now()
	c.currentYear = now.Year()
	c.currentMonth = now.Month()

	years := make([]string, 51)
	for i := 0; i < 51; i++ {
		years[i] = fmt.Sprintf("%d", now.Year()-25+i)
	}

	months := []string{
		locales.Translate("datesync.month.jan"),
		locales.Translate("datesync.month.feb"),
		locales.Translate("datesync.month.mar"),
		locales.Translate("datesync.month.apr"),
		locales.Translate("datesync.month.may"),
		locales.Translate("datesync.month.jun"),
		locales.Translate("datesync.month.jul"),
		locales.Translate("datesync.month.aug"),
		locales.Translate("datesync.month.sep"),
		locales.Translate("datesync.month.okt"),
		locales.Translate("datesync.month.nov"),
		locales.Translate("datesync.month.dec"),
	}

	c.yearSelect = widget.NewSelect(years, func(s string) {
		year := 0
		fmt.Sscanf(s, "%d", &year)
		c.currentYear = year
		c.updateDays()
	})
	c.monthSelect = widget.NewSelect(months, func(s string) {
		months := map[string]time.Month{
			locales.Translate("datesync.month.jan"): time.January,
			locales.Translate("datesync.month.feb"): time.February,
			locales.Translate("datesync.month.mar"): time.March,
			locales.Translate("datesync.month.apr"): time.April,
			locales.Translate("datesync.month.may"): time.May,
			locales.Translate("datesync.month.jun"): time.June,
			locales.Translate("datesync.month.jul"): time.July,
			locales.Translate("datesync.month.aug"): time.August,
			locales.Translate("datesync.month.sep"): time.September,
			locales.Translate("datesync.month.okt"): time.October,
			locales.Translate("datesync.month.nov"): time.November,
			locales.Translate("datesync.month.dec"): time.December,
		}

		c.currentMonth = months[s]
		c.updateDays()
	})

	c.yearSelect.SetSelected(fmt.Sprintf("%d", now.Year()))
	c.monthSelect.SetSelected(months[now.Month()-1])
	c.updateDays()
	return c
}

// CreateRenderer implements the fyne.Widget interface.
// It creates and returns a widget renderer for the custom calendar.
func (c *CustomCalendar) CreateRenderer() fyne.WidgetRenderer {
	header := container.NewHBox(c.monthSelect, c.yearSelect)
	content := container.NewVBox(header, c.daysGrid)
	return widget.NewSimpleRenderer(content)
}

// updateDays updates the day grid in the calendar based on the current year and month.
// It creates day buttons for each day in the month and handles proper layout with weekday alignment.
func (c *CustomCalendar) updateDays() {
	if c.daysGrid == nil {
		return
	}

	c.daysGrid.Objects = []fyne.CanvasObject{}

	days := []string{
		locales.Translate("datesync.day.mon"),
		locales.Translate("datesync.day.tue"),
		locales.Translate("datesync.day.wed"),
		locales.Translate("datesync.day.thu"),
		locales.Translate("datesync.day.fri"),
		locales.Translate("datesync.day.sat"),
		locales.Translate("datesync.day.sun"),
	}

	for _, day := range days {
		c.daysGrid.Add(widget.NewLabel(day))
	}

	firstDay := time.Date(c.currentYear, c.currentMonth, 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)
	weekday := int(firstDay.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	for i := 1; i < weekday; i++ {
		c.daysGrid.Add(widget.NewLabel(""))
	}

	for day := 1; day <= lastDay.Day(); day++ {
		currentDay := day
		dayBtn := common.CreateCalendarDayButton(day, func() {
			date := time.Date(c.currentYear, c.currentMonth, currentDay, 0, 0, 0, 0, time.Local)
			if c.onSelected != nil {
				c.onSelected(date)
			}
		})
		c.daysGrid.Add(dayBtn)
	}

	c.Refresh()
}

// NewDateSyncModule creates a new instance of DateSyncModule.
// It initializes the UI components and loads the configuration.
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for module settings
//   - dbMgr: Database manager for database operations
//   - errorHandler: Error handler for error management
//
// Returns a new DateSyncModule instance.
func NewDateSyncModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *DateSyncModule {
	m := &DateSyncModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	return m
}

// GetName returns the localized name of the module.
// This implements the Module interface method.
func (m *DateSyncModule) GetName() string {
	return locales.Translate("datesync.mod.name")
}

// GetConfigName returns the configuration identifier for the module.
// This implements the Module interface method and is used for configuration storage.
func (m *DateSyncModule) GetConfigName() string {
	return "datesync"
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method.
func (m *DateSyncModule) GetIcon() fyne.Resource {
	return theme.StorageIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI.
// Returns a canvas object containing the module's UI components.
func (m *DateSyncModule) GetModuleContent() fyne.CanvasObject {
	// Left section - excluded folders
	leftHeader := widget.NewLabel(locales.Translate("datesync.label.leftpanel"))
	leftHeader.TextStyle = fyne.TextStyle{Bold: true}

	leftSection := container.NewVBox(
		leftHeader,
		m.excludeFoldersCheck,
		m.foldersContainer,
		container.NewHBox(layout.NewSpacer(), m.standardUpdateBtn),
	)

	// Right section - custom date folders
	rightHeader := widget.NewLabel(locales.Translate("datesync.label.rightpanel"))
	rightHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Date picker with calendar button
	m.datePickerContainer = container.NewBorder(nil, nil, nil, m.calendarBtn, m.datePickerEntry)

	rightSection := container.NewVBox(
		rightHeader,
		m.datePickerContainer,
		m.customDateContainer,
		container.NewHBox(layout.NewSpacer(), m.customDateUpdateBtn),
	)

	// Create a horizontal container with left and right sections
	horizontalLayout := container.NewHSplit(leftSection, rightSection)
	// Set a fixed position for the divider (50% of the width)
	horizontalLayout.Offset = 0.5

	// Create content container
	contentContainer := container.NewVBox(
		horizontalLayout,
	)

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("datesync.label.info")),
		widget.NewSeparator(),
		contentContainer,
	)

	return moduleContent
}

// GetContent returns the module's main UI content including status messages.
// This implements the Module interface method.
// Returns a canvas object containing the complete module layout.
func (m *DateSyncModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig loads module configuration from the provided ModuleConfig.
// It initializes UI components with the stored values and creates default configuration if needed.
// Parameter:
//   - cfg: The module configuration to load
//
// The method sets IsLoadingConfig flag during loading to prevent triggering save operations.
func (m *DateSyncModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Check if configuration is nil or Fields are not initialized
	if cfg.Fields == nil {
		cfg = common.NewModuleConfig()

		// Set default values with their definitions
		cfg.SetWithDefinitionAndActions("custom_date", "", "date", true, "valid_date", []string{"custom"})
		cfg.SetWithDefinitionAndActions("custom_date_folders", "", "folder", true, "exists", []string{"custom"})
		cfg.SetWithDefinitionAndActions("exclude_folders_enabled", "false", "checkbox", false, "none", []string{"standard"})
		cfg.SetWithDependencyAndActions("excluded_folders", "", "folder", true, "exclude_folders_enabled", "true", "filled", []string{"standard"})

		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	}

	// Clear existing entries
	m.foldersContainer.Objects = nil
	m.customDateContainer.Objects = nil
	m.excludedFoldersEntry = nil
	m.customDateFoldersEntry = nil

	// Load excluded folders checkbox state
	m.excludeFoldersCheck.SetChecked(cfg.GetBool("exclude_folders_enabled", false))

	// Load excluded folders if enabled
	excludedFoldersEntry := cfg.Get("excluded_folders", "")
	if excludedFoldersEntry != "" {
		folderPaths := strings.Split(excludedFoldersEntry, "|")
		for _, folderPath := range folderPaths {
			if folderPath != "" {
				m.addFolderEntryForConfig(folderPath, true)
			}
		}
	}

	// Ensure there's at least one empty entry for excluded folders
	if len(m.excludedFoldersEntry) == 0 || m.excludedFoldersEntry[len(m.excludedFoldersEntry)-1].Text != "" {
		m.addFolderEntry(ExcludedFolder)
	}

	// Load custom date folders
	customDateFoldersEntry := cfg.Get("custom_date_folders", "")
	if customDateFoldersEntry != "" {
		folderPaths := strings.Split(customDateFoldersEntry, "|")
		for _, folderPath := range folderPaths {
			if folderPath != "" {
				m.addFolderEntryForConfig(folderPath, false)
			}
		}
	}

	// Ensure at least one empty field is present
	if len(m.customDateFoldersEntry) == 0 {
		m.addFolderEntry(CustomDateFolder)
	}

	// Load custom date
	m.datePickerEntry.SetText(cfg.Get("custom_date", ""))
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
// It collects values from all UI components and stores them in the configuration manager.
// Returns the newly created and saved ModuleConfig.
// The method checks IsLoadingConfig flag to prevent saving during configuration loading.
func (m *DateSyncModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	// Build fresh config
	cfg := common.NewModuleConfig()

	// Store checkbox state with definition
	cfg.SetWithDefinitionAndActions("exclude_folders_enabled",
		fmt.Sprintf("%t", m.excludeFoldersCheck.Checked),
		"checkbox",
		false,
		"none", []string{"standard"})

	// Save excluded folders with dependency
	var excludedFoldersEntry []string
	for _, entry := range m.excludedFoldersEntry {
		if entry.Text != "" {
			excludedFoldersEntry = append(excludedFoldersEntry, entry.Text)
		}
	}
	cfg.SetWithDependencyAndActions("excluded_folders",
		strings.Join(excludedFoldersEntry, "|"),
		"folder",
		true,
		"exclude_folders_enabled",
		"true",
		"exists", []string{"standard"})

	// Save custom date folders with definition
	var customDateFoldersEntry []string
	for _, entry := range m.customDateFoldersEntry {
		if entry.Text != "" {
			customDateFoldersEntry = append(customDateFoldersEntry, entry.Text)
		}
	}
	cfg.SetWithDefinitionAndActions("custom_date_folders",
		strings.Join(customDateFoldersEntry, "|"),
		"folder",
		true,
		"exists", []string{"custom"})

	// Save custom date with definition
	cfg.SetWithDefinitionAndActions("custom_date",
		m.datePickerEntry.Text,
		"date",
		true,
		"valid_date", []string{"custom"})

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components for the module.
// It creates all UI elements, sets up event handlers, and initializes containers.
// This method is called during module creation.
func (m *DateSyncModule) initializeUI() {
	// Create excluded folders checkbox
	m.excludeFoldersCheck = widget.NewCheck(locales.Translate("datesync.chkbox.exception"),
		m.CreateBoolChangeHandler(func() {
			m.SaveConfig()
		}),
	)

	// Create folders container
	m.foldersContainer = container.NewVBox()

	// Create custom date container
	m.customDateContainer = container.NewVBox()

	// Create date picker
	m.datePickerEntry = widget.NewEntry()
	m.datePickerEntry.SetPlaceHolder(locales.Translate("datesync.date.placeholder"))
	m.datePickerEntry.OnChanged = m.CreateChangeHandler(func() {

		// Limit input to 10 characters (YYYY-MM-DD)
		if len(m.datePickerEntry.Text) > 10 {
			m.datePickerEntry.SetText(m.datePickerEntry.Text[:10])
		}

		m.SaveConfig()
	})

	// Create calendar button
	m.calendarBtn = widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
		// Create dialog with calendar that will close automatically after date selection
		calendar := NewCustomCalendar(func(selectedDate time.Time) {
			m.datePickerEntry.SetText(selectedDate.Format("2006-01-02"))
			m.SaveConfig()
		})
		dlg := dialog.NewCustomWithoutButtons(locales.Translate("datesync.datepicker.header"), calendar, m.Window)
		calendar.onSelected = func(selectedDate time.Time) {
			m.datePickerEntry.SetText(selectedDate.Format("2006-01-02"))
			m.SaveConfig()
			dlg.Hide()
		}

		dlg.Show()

	})

	// Create standard update button
	m.standardUpdateBtn = common.CreateSubmitButton(locales.Translate("datesync.button.startupdate"), func() {
		m.Start("standard")
	},
	)

	// Create custom date update button
	m.customDateUpdateBtn = common.CreateSubmitButton(locales.Translate("datesync.button.startcustomupdate"), func() {
		m.Start("custom")
	},
	)

	// Add initial folder entries
	m.addFolderEntry(ExcludedFolder)
	m.addFolderEntry(CustomDateFolder)
}

// addFolderEntry adds a new folder entry to the appropriate container.
// It creates a folder selection field with delete button and handles dynamic entry addition.
// Parameters:
//   - folderType: The type of folder entry to add (CustomDateFolder or ExcludedFolder)
//
// Returns the newly created entry widget that was added to the entries slice, or nil if maximum entries reached.
func (m *DateSyncModule) addFolderEntry(folderType FolderEntryType) *widget.Entry {
	// Determine which container and entries to use based on folder type
	var container *fyne.Container
	var entries []*widget.Entry
	var placeholderText string = locales.Translate("common.entry.placeholderpath")

	// Set container and entries based on folder type
	if folderType == CustomDateFolder {
		// Check if we've reached the maximum number of entries
		if len(m.customDateFoldersEntry) >= maxFolderEntries {
			return nil
		}
		container = m.customDateContainer
		entries = m.customDateFoldersEntry
	} else {
		// Check if we've reached the maximum number of entries
		if len(m.excludedFoldersEntry) >= maxFolderEntries {
			return nil
		}
		container = m.foldersContainer
		entries = m.excludedFoldersEntry
	}

	// Create a new entry
	newEntry := widget.NewEntry()
	newEntry.SetPlaceHolder(placeholderText)

	// Define folder selection callback
	folderSelectCallback := func(entryWidget *widget.Entry, path string) {
		// Set the text of the entry
		entryWidget.SetText(path)

		// Safety check for empty entry lists
		var isLastEntry bool = false

		// Update entries reference to ensure we're using current state
		if folderType == CustomDateFolder {
			entries = m.customDateFoldersEntry
		} else {
			entries = m.excludedFoldersEntry
		}

		if len(entries) > 0 {
			// Find index of the current entry
			indexOfEntry := -1
			for i, e := range entries {
				if e == entryWidget {
					indexOfEntry = i
					break
				}
			}

			// Check if it's the last entry
			if indexOfEntry != -1 && indexOfEntry == len(entries)-1 {
				isLastEntry = true
			}
		}

		// Add a new entry if this is the last non-empty one and we haven't reached the limit
		if path != "" && isLastEntry {
			if folderType == CustomDateFolder && len(m.customDateFoldersEntry) < maxFolderEntries {
				m.addFolderEntry(CustomDateFolder)
			} else if folderType == ExcludedFolder && len(m.excludedFoldersEntry) < maxFolderEntries {
				m.addFolderEntry(ExcludedFolder)
			}
		}

		m.SaveConfig()
	}

	// Define delete callback
	deleteCallback := func(entryWidget *widget.Entry) {
		m.removeFolderEntry(entryWidget, folderType)
	}

	// Create folder field with delete button
	folderField := common.CreateFolderSelectionFieldWithDelete(
		locales.Translate("common.button.browsefolder"),
		newEntry,
		func(path string) {
			// Set the text of the entry
			newEntry.SetText(path)

			// Call the folder select callback
			folderSelectCallback(newEntry, path)
		},
		func() {
			// Call the delete callback
			deleteCallback(newEntry)
		},
	)

	// Add the new entry to the appropriate container
	container.Add(folderField)

	// Update the appropriate entries slice
	if folderType == CustomDateFolder {
		m.customDateFoldersEntry = append(m.customDateFoldersEntry, newEntry)
	} else {
		m.excludedFoldersEntry = append(m.excludedFoldersEntry, newEntry)
	}

	// Refresh the container
	container.Refresh()

	return newEntry
}

// removeFolderEntry removes a folder entry from the appropriate list and updates the UI.
// It rebuilds the container after removal and ensures at least one empty entry remains.
// Parameters:
//   - entryToRemove: The entry widget to remove
//   - folderType: The type of folder entry (CustomDateFolder or ExcludedFolder)
func (m *DateSyncModule) removeFolderEntry(entryToRemove *widget.Entry, folderType FolderEntryType) {
	// Determine which container and entries to use based on folder type
	var container *fyne.Container
	var entries []*widget.Entry

	if folderType == CustomDateFolder {
		container = m.customDateContainer
		entries = m.customDateFoldersEntry
	} else {
		container = m.foldersContainer
		entries = m.excludedFoldersEntry
	}

	// Find the index of the entry to remove
	indexToRemove := -1
	for i, entry := range entries {
		if entry == entryToRemove {
			indexToRemove = i
			break
		}
	}

	// If entry not found, return
	if indexToRemove == -1 {
		return
	}

	// Remove the entry from the list
	if folderType == CustomDateFolder {
		// Bezpečná úprava seznamu s ošetřením indexů
		if indexToRemove < len(m.customDateFoldersEntry) {
			m.customDateFoldersEntry = append(m.customDateFoldersEntry[:indexToRemove], m.customDateFoldersEntry[indexToRemove+1:]...)
		}
	} else {
		// Bezpečná úprava seznamu s ošetřením indexů
		if indexToRemove < len(m.excludedFoldersEntry) {
			m.excludedFoldersEntry = append(m.excludedFoldersEntry[:indexToRemove], m.excludedFoldersEntry[indexToRemove+1:]...)
		}
	}

	// Aktualizujeme reference na aktuální seznamy
	if folderType == CustomDateFolder {
		entries = m.customDateFoldersEntry
	} else {
		entries = m.excludedFoldersEntry
	}

	// Clear and rebuild the container
	container.Objects = nil

	// Rebuild the container with all entries
	for i, entry := range entries {
		// Vytváříme kopie pro použití v closures, abychom předešli problémům
		// s odkazováním na proměnné ve smyčce
		currentEntry := entry
		isLastEntry := (i == len(entries)-1)

		folderField := common.CreateFolderSelectionFieldWithDelete(
			locales.Translate("common.button.browsefolder"),
			currentEntry,
			func(path string) {
				currentEntry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if path != "" && isLastEntry {
					if folderType == CustomDateFolder && len(m.customDateFoldersEntry) < maxFolderEntries {
						m.addFolderEntry(folderType)
					} else if folderType == ExcludedFolder && len(m.excludedFoldersEntry) < maxFolderEntries {
						m.addFolderEntry(folderType)
					}
				}
				m.SaveConfig()
			},
			func() {
				m.removeFolderEntry(currentEntry, folderType)
			},
		)
		container.Add(folderField)
	}

	// Ensure there's at least one empty entry
	if folderType == CustomDateFolder && len(m.customDateFoldersEntry) == 0 {
		m.addFolderEntry(CustomDateFolder)
	} else if folderType == ExcludedFolder && len(m.excludedFoldersEntry) == 0 {
		m.addFolderEntry(ExcludedFolder)
	}

	// Refresh the container and save config
	container.Refresh()
	m.SaveConfig()
}

// addFolderEntryForConfig adds a folder entry during config loading without triggering auto-add.
// This method is used specifically during configuration loading to prevent cascading UI updates.
// Parameters:
//   - folderPath: The folder path to set in the entry
//   - isExcluded: Whether this is an excluded folder (true) or custom date folder (false)
func (m *DateSyncModule) addFolderEntryForConfig(folderPath string, isExcluded bool) {
	// Determine folder type based on isExcluded parameter
	folderType := CustomDateFolder
	if isExcluded {
		folderType = ExcludedFolder
	}

	// Check limits based on folder type
	if folderType == CustomDateFolder {
		// Check if we've reached the maximum number of entries
		if len(m.customDateFoldersEntry) >= maxFolderEntries {
			return
		}
	} else {
		// Check if we've reached the maximum number of entries
		if len(m.excludedFoldersEntry) >= maxFolderEntries {
			return
		}
	}

	// Initialize entry field with the provided folder path
	entry := widget.NewEntry()
	entry.SetText(folderPath)

	// Create folder field with delete button using common component
	folderField := common.CreateFolderSelectionFieldWithDelete(
		locales.Translate("common.entry.placeholderpath"),
		entry,
		func(path string) {
			entry.SetText(path)
			// Add new field only if not loading config, this is last entry with value, and limit not reached
			if !m.IsLoadingConfig && entry.Text != "" {
				if folderType == CustomDateFolder {
					if len(m.customDateFoldersEntry) < maxFolderEntries &&
						entry == m.customDateFoldersEntry[len(m.customDateFoldersEntry)-1] {
						m.addFolderEntry(CustomDateFolder)
					}
				} else {
					if len(m.excludedFoldersEntry) < maxFolderEntries &&
						entry == m.excludedFoldersEntry[len(m.excludedFoldersEntry)-1] {
						m.addFolderEntry(ExcludedFolder)
					}
				}
			}
			m.SaveConfig()
		},
		func() {
			m.removeFolderEntry(entry, folderType)
		},
	)

	// Add entry to appropriate slice and container
	if folderType == CustomDateFolder {
		m.customDateFoldersEntry = append(m.customDateFoldersEntry, entry)
		m.customDateContainer.Add(folderField)
	} else {
		m.excludedFoldersEntry = append(m.excludedFoldersEntry, entry)
		m.foldersContainer.Add(folderField)
	}
}

// Start performs the necessary steps before starting the main process.
// It saves the configuration, validates the inputs, informs the user, displays a dialog with a progress bar
// and starts the main process based on specific mode.
// Parameters:
//   - mode: The operation mode, either "standard" for date synchronization over music library
//     or "custom" to set specific date for songs stored in the selected location
//
// Input validation includes testing the database connection and creating a backup.
// The actual processing is started in a goroutine to keep the UI responsive.
func (m *DateSyncModule) Start(mode string) {
	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate(mode); err != nil {
		return
	}

	// Show progress dialog with cancel support
	m.ShowProgressDialog(locales.Translate("datesync.dialog.header"))

	// Start processing in goroutine based on mode
	switch mode {
	case "standard":
		go m.processStandardUpdate()
	case "custom":
		go m.processCustomUpdate()
	}
}

// processStandardUpdate performs the standard date synchronization.
// It updates the progress dialog, calls setStandardDates to perform the database update,
// handles errors and cancellation, and updates the UI with results.
// This method runs in a separate goroutine.
func (m *DateSyncModule) processStandardUpdate() {

	// Execute standard date sync
	m.UpdateProgressStatus(0.3, locales.Translate("common.status.updating"))
	m.AddInfoMessage(locales.Translate("common.status.updating"))
	updatedCount, err := m.setStandardDates()
	if err != nil {
		m.CloseProgressDialog()
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "StandardDateUpdate",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Update progress with count
	m.UpdateProgressStatus(0.9, fmt.Sprintf(locales.Translate("common.status.progress"), updatedCount, updatedCount))

	// Update progress and complete dialog with final count
	m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
	m.CompleteProgressDialog()

	// Update button to show completion
	common.UpdateButtonToCompleted(m.standardUpdateBtn)
}

// processCustomUpdate performs the custom date synchronization.
// It collects custom date folders, calls setCustomDates to perform the database update,
// handles errors and cancellation, and updates the UI with results.
// This method runs in a separate goroutine.
func (m *DateSyncModule) processCustomUpdate() {

	// No need to parse custom date, it's already parsed in the validator
	customDate, _ := time.Parse("2006-01-02", m.datePickerEntry.Text)

	// Collect custom date folders
	var customDateFolders []string
	for _, entry := range m.customDateFoldersEntry {
		if entry.Text != "" {
			customDateFolders = append(customDateFolders, entry.Text)
		}
	}

	// Execute custom date sync
	m.UpdateProgressStatus(0.3, locales.Translate("common.status.updating"))
	m.AddInfoMessage(locales.Translate("common.status.updating"))
	updatedCount, err := m.setCustomDates(customDateFolders, customDate)
	if err != nil {
		m.CloseProgressDialog()
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "CustomDateUpdate",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Check if cancelled after database update
	if m.IsCancelled() {
		return
	}

	// Update progress with count
	m.UpdateProgressStatus(0.9, fmt.Sprintf(locales.Translate("common.status.progress"), updatedCount, updatedCount))

	// Update progress and complete dialog with final count
	m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
	m.CompleteProgressDialog()

	// Update button to show completion
	common.UpdateButtonToCompleted(m.customDateUpdateBtn)
}

// setStandardDates updates the dates in the Rekordbox database based on release dates.
// It builds a WHERE clause to exclude specified folders if needed, counts affected records,
// and executes the update query.
// Returns:
//   - int: The number of records updated
//   - error: Any error that occurred during the operation
//
// The method handles cancellation during processing.
func (m *DateSyncModule) setStandardDates() (int, error) {
	// Build WHERE clause for excluded folders
	whereClause := "WHERE ReleaseDate IS NOT NULL"
	if m.excludeFoldersCheck.Checked {
		var excludedFolders []string
		for _, entry := range m.excludedFoldersEntry {
			if entry.Text != "" {
				excludedFolders = append(excludedFolders, entry.Text)
			}
		}
		if len(excludedFolders) > 0 {
			for _, folder := range excludedFolders {
				whereClause += fmt.Sprintf(" AND FolderPath NOT LIKE '%s%%'", common.ToDbPath(folder, true))
			}
		}
	}

	// Get total number of records to be updated
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM djmdContent %s", whereClause)
	var totalCount int
	err := m.dbMgr.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesync.err.dbitemscount"), err)
	}

	// Add info message about number of records to update
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.toupdatecount"), totalCount))

	// Check if cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("common.status.stopped", 0, totalCount)
		common.UpdateButtonToCompleted(m.standardUpdateBtn)
		return 0, nil
	}

	// Update query
	updateQuery := fmt.Sprintf("UPDATE djmdContent SET StockDate = ReleaseDate, DateCreated = ReleaseDate %s", whereClause)
	err = m.dbMgr.Execute(updateQuery)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesync.err.dbupdate"), err)
	}

	return totalCount, nil
}

// setCustomDates sets custom dates for tracks in selected folders.
// It builds a WHERE clause to include only specified folders, counts affected records,
// and executes the update query with the provided custom date.
// Parameters:
//   - customDateFoldersEntry: List of folder paths to include in the update
//   - customDate: The date to set for all matching tracks
//
// Returns:
//   - int: The number of records updated
//   - error: Any error that occurred during the operation
//
// The method handles cancellation during processing.
func (m *DateSyncModule) setCustomDates(customDateFoldersEntry []string, customDate time.Time) (int, error) {

	// Build WHERE clause for selected folders
	whereClause := "WHERE"
	for i, folder := range customDateFoldersEntry {
		if i > 0 {
			whereClause += " OR"
		}
		whereClause += fmt.Sprintf(" FolderPath LIKE '%s%%'", common.ToDbPath(folder, true))
	}

	// Get total number of records to be updated
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM djmdContent %s", whereClause)
	var totalCount int
	err := m.dbMgr.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesync.err.dbitemscount"), err)
	}

	// Add info message about number of records to update
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.toupdatecount"), totalCount))

	// Check if cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("common.status.stopped", 0, totalCount)
		common.UpdateButtonToCompleted(m.customDateUpdateBtn)
		return 0, nil
	}

	// Update query
	updateQuery := fmt.Sprintf(`
		UPDATE djmdContent
		SET StockDate = ?,
			DateCreated = ?
		%s`, whereClause)

	err = m.dbMgr.Execute(updateQuery, customDate.Format("2006-01-02"), customDate.Format("2006-01-02"))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesync.err.dbupdate"), err)
	}

	return totalCount, nil
}
