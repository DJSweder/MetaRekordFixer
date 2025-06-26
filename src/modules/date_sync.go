// modules/date_sync.go

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

// DateSyncModule implements a module for synchronizing dates in the Rekordbox database
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

// CustomCalendar implements a custom calendar widget for date selection
type CustomCalendar struct {
	widget.BaseWidget
	currentYear  int
	currentMonth time.Month
	daysGrid     *fyne.Container
	monthSelect  *widget.Select
	onSelected   func(time.Time)
	yearSelect   *widget.Select
}

// NewCustomCalendar creates a new custom calendar widget
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

// CreateRenderer implements the fyne.Widget interface
func (c *CustomCalendar) CreateRenderer() fyne.WidgetRenderer {
	header := container.NewHBox(c.monthSelect, c.yearSelect)
	content := container.NewVBox(header, c.daysGrid)
	return widget.NewSimpleRenderer(content)
}

// updateDays updates the day grid in the calendar
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

// GetName returns the localized name of the module
func (m *DateSyncModule) GetName() string {
	return locales.Translate("datesync.mod.name")
}

// GetConfigName returns the configuration identifier for the module
func (m *DateSyncModule) GetConfigName() string {
	return "datesync"
}

// GetIcon returns the module's icon
func (m *DateSyncModule) GetIcon() fyne.Resource {
	return theme.StorageIcon()
}

// GetModuleContent returns the module's specific content without status messages
// This implements the method from ModuleBase to provide the module-specific UI
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

// GetContent returns the module's main UI content.
func (m *DateSyncModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig loads module configuration
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
		m.addExcludedFolderEntry()
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

	// Ensure there's at least one empty entry for custom date folders
	if len(m.customDateFoldersEntry) == 0 || m.customDateFoldersEntry[len(m.customDateFoldersEntry)-1].Text != "" {
		m.addCustomDateFolderEntry()
	}

	// Load custom date
	m.datePickerEntry.SetText(cfg.Get("custom_date", ""))
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
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

// initializeUI sets up the user interface components
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
	m.addExcludedFolderEntry()
	m.addCustomDateFolderEntry()
}

// addCustomDateFolderEntry adds a new entry for custom date folder selection
func (m *DateSyncModule) addCustomDateFolderEntry() {
	if len(m.customDateFoldersEntry) >= 6 {
		return
	}

	// Initialize entry field for custom date folder selection
	entry := widget.NewEntry()
	customDateFolderField := common.CreateFolderSelectionFieldWithDelete(
		locales.Translate("common.entry.placeholderpath"),
		entry,
		func(path string) {
			entry.SetText(path)
			// Add new field if this is the last non-empty one and we haven't reached the limit
			if entry.Text != "" && len(m.customDateFoldersEntry) < 6 && entry == m.customDateFoldersEntry[len(m.customDateFoldersEntry)-1] {
				m.addCustomDateFolderEntry()
			}
			m.SaveConfig()
		},
		func() {
			m.removeCustomDateFolderEntry(entry)
		},
	)

	m.customDateFoldersEntry = append(m.customDateFoldersEntry, entry)
	m.customDateContainer.Add(customDateFolderField)
}

// addExcludedFolderEntry adds a new entry for excluded folder selection
func (m *DateSyncModule) addExcludedFolderEntry() {
	if len(m.excludedFoldersEntry) >= 6 {
		return
	}

	// Initialize entry field for folder selection
	entry := widget.NewEntry()
	folderField := common.CreateFolderSelectionFieldWithDelete(
		locales.Translate("common.entry.placeholderpath"),
		entry,
		func(path string) {
			entry.SetText(path)
			// Add new field if this is the last non-empty one and we haven't reached the limit
			if entry.Text != "" && len(m.excludedFoldersEntry) < 6 && entry == m.excludedFoldersEntry[len(m.excludedFoldersEntry)-1] {
				m.addExcludedFolderEntry()
			}
			m.SaveConfig()
		},
		func() {
			m.removeExcludedFolderEntry(entry)
		},
	)

	m.excludedFoldersEntry = append(m.excludedFoldersEntry, entry)
	m.foldersContainer.Add(folderField)
}

// addFolderEntryForConfig adds a folder entry during config loading without triggering auto-add
func (m *DateSyncModule) addFolderEntryForConfig(folderPath string, isExcluded bool) {
	if (isExcluded && len(m.excludedFoldersEntry) >= 6) || (!isExcluded && len(m.customDateFoldersEntry) >= 6) {
		return
	}

	// Initialize entry field for folder selection
	entry := widget.NewEntry()
	entry.SetText(folderPath)

	var folderField fyne.CanvasObject
	if isExcluded {
		folderField = common.CreateFolderSelectionFieldWithDelete(
			locales.Translate("common.entry.placeholderpath"),
			entry,
			func(path string) {
				entry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if !m.IsLoadingConfig && entry.Text != "" && len(m.excludedFoldersEntry) < 6 && entry == m.excludedFoldersEntry[len(m.excludedFoldersEntry)-1] {
					m.addExcludedFolderEntry()
				}
				m.SaveConfig()
			},
			func() {
				m.removeExcludedFolderEntry(entry)
			},
		)
		m.excludedFoldersEntry = append(m.excludedFoldersEntry, entry)
		m.foldersContainer.Add(folderField)
	} else {
		folderField = common.CreateFolderSelectionFieldWithDelete(
			locales.Translate("common.entry.placeholderpath"),
			entry,
			func(path string) {
				entry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if !m.IsLoadingConfig && entry.Text != "" && len(m.customDateFoldersEntry) < 6 && entry == m.customDateFoldersEntry[len(m.customDateFoldersEntry)-1] {
					m.addCustomDateFolderEntry()
				}
				m.SaveConfig()
			},
			func() {
				m.removeCustomDateFolderEntry(entry)
			},
		)
		m.customDateFoldersEntry = append(m.customDateFoldersEntry, entry)
		m.customDateContainer.Add(folderField)
	}
}

// removeCustomDateFolderEntry removes a folder entry from the custom date folders list
func (m *DateSyncModule) removeCustomDateFolderEntry(entryToRemove *widget.Entry) {
	// Find the index of the entry to remove
	indexToRemove := -1
	for i, entry := range m.customDateFoldersEntry {
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
	m.customDateFoldersEntry = append(m.customDateFoldersEntry[:indexToRemove], m.customDateFoldersEntry[indexToRemove+1:]...)

	// Clear and rebuild the container
	m.customDateContainer.Objects = nil
	for _, entry := range m.customDateFoldersEntry {
		folderField := common.CreateFolderSelectionFieldWithDelete(
			locales.Translate("common.entry.placeholderpath"),
			entry,
			func(path string) {
				entry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if entry.Text != "" && len(m.customDateFoldersEntry) < 6 && entry == m.customDateFoldersEntry[len(m.customDateFoldersEntry)-1] {
					m.addCustomDateFolderEntry()
				}
				m.SaveConfig()
			},
			func() {
				m.removeCustomDateFolderEntry(entry)
			},
		)
		m.customDateContainer.Add(folderField)
	}

	// Ensure there's at least one empty entry
	if len(m.customDateFoldersEntry) == 0 {
		m.addCustomDateFolderEntry()
	}

	// Refresh the container and save config
	m.customDateContainer.Refresh()
	m.SaveConfig()
}

// removeExcludedFolderEntry removes a folder entry from the excluded folders list
func (m *DateSyncModule) removeExcludedFolderEntry(entryToRemove *widget.Entry) {
	// Find the index of the entry to remove
	indexToRemove := -1
	for i, entry := range m.excludedFoldersEntry {
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
	m.excludedFoldersEntry = append(m.excludedFoldersEntry[:indexToRemove], m.excludedFoldersEntry[indexToRemove+1:]...)

	// Clear and rebuild the container
	m.foldersContainer.Objects = nil
	for _, entry := range m.excludedFoldersEntry {
		folderField := common.CreateFolderSelectionFieldWithDelete(
			locales.Translate("common.entry.placeholderpath"),
			entry,
			func(path string) {
				entry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if entry.Text != "" && len(m.excludedFoldersEntry) < 6 && entry == m.excludedFoldersEntry[len(m.excludedFoldersEntry)-1] {
					m.addExcludedFolderEntry()
				}
				m.SaveConfig()
			},
			func() {
				m.removeExcludedFolderEntry(entry)
			},
		)
		m.foldersContainer.Add(folderField)
	}

	// Ensure there's at least one empty entry
	if len(m.excludedFoldersEntry) == 0 {
		m.addExcludedFolderEntry()
	}

	// Refresh the container and save config
	m.foldersContainer.Refresh()
	m.SaveConfig()
}

// Start performs the necessary steps before starting the main process
// It saves the configuration, validates the inputs, informs the user, displays a dialog with a progress bar
// and starts the main process based on specific mode:
// - "standard" for date synchronization over music library
// - "custom" to set specific date for songs stored in the selected location
// Input validation also includes a test of the connection to the database and creating a backup of it.
func (m *DateSyncModule) Start(mode string) {

	if mode == "standard" {

		// Create and run validator
		validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
		if err := validator.Validate(mode); err != nil {
			return
		}

		// Show progress dialog with cancel support
		m.ShowProgressDialog(locales.Translate("datesync.dialog.header"))

		// Start processing in goroutine
		go m.processStandardUpdate()

	} else if mode == "custom" {

		// Create and run validator
		validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
		if err := validator.Validate(mode); err != nil {
			return
		}

		// Show progress dialog with cancel support
		m.ShowProgressDialog(locales.Translate("datesync.dialog.header"))

		// Start processing in goroutine
		go m.processCustomUpdate()
	}
}

// processStandardUpdate performs the standard date synchronization
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

// processCustomUpdate performs the custom date synchronization
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

// setStandardDates updates the dates in the Rekordbox database
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

// setCustomDates sets custom dates for tracks in selected folders
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
