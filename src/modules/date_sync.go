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
	excludeFoldersCheck    *widget.Check
	customDateFoldersEntry []*widget.Entry
	excludedFoldersEntry   []*widget.Entry
	datePickerEntry        *widget.Entry
	foldersContainer       *fyne.Container
	customDateContainer    *fyne.Container
	datePickerContainer    *fyne.Container
	standardUpdateBtn      *widget.Button
	customDateUpdateBtn    *widget.Button
	calendarBtn            *widget.Button
}

// CustomCalendar implements a custom calendar widget for date selection
type CustomCalendar struct {
	widget.BaseWidget
	yearSelect   *widget.Select
	monthSelect  *widget.Select
	daysGrid     *fyne.Container
	onSelected   func(time.Time)
	currentYear  int
	currentMonth time.Month
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
	return "date_sync"
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

	if common.IsNilConfig(cfg) {
		// Initialize with default values and save them
		cfg = common.NewModuleConfig()
		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
		return
	}

	// Clear existing entries
	m.foldersContainer.Objects = nil
	m.customDateContainer.Objects = nil
	m.excludedFoldersEntry = nil
	m.customDateFoldersEntry = nil

	// Load excluded folders - use the dedicated checkbox state field
	excludeFoldersEnabled := cfg.GetBool("exclude_folders_enabled", false)
	m.excludeFoldersCheck.SetChecked(excludeFoldersEnabled)

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
	// Also ensure that the last entry is empty for better UX
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
	// Also ensure that the last entry is empty for better UX
	if len(m.customDateFoldersEntry) == 0 || m.customDateFoldersEntry[len(m.customDateFoldersEntry)-1].Text != "" {
		m.addCustomDateFolderEntry()
	}

	// Load custom date
	customDate := cfg.Get("custom_date", "")
	if customDate != "" {
		m.datePickerEntry.SetText(customDate)
	}
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
func (m *DateSyncModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	// Build fresh config
	cfg := common.NewModuleConfig()

	// Store checkbox states
	cfg.SetBool("exclude_folders_enabled", m.excludeFoldersCheck.Checked)

	// Save excluded folders
	var excludedFoldersEntry []string
	for _, entry := range m.excludedFoldersEntry {
		if entry.Text != "" {
			excludedFoldersEntry = append(excludedFoldersEntry, entry.Text)
		}
	}
	cfg.Set("excluded_folders", strings.Join(excludedFoldersEntry, "|"))

	// Save custom date folders
	var customDateFoldersEntry []string
	for _, entry := range m.customDateFoldersEntry {
		if entry.Text != "" {
			customDateFoldersEntry = append(customDateFoldersEntry, entry.Text)
		}
	}
	cfg.Set("custom_date_folders", strings.Join(customDateFoldersEntry, "|"))

	// Save custom date
	cfg.Set("custom_date", m.datePickerEntry.Text)

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
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

// CreateRenderer implements the fyne.Widget interface
func (c *CustomCalendar) CreateRenderer() fyne.WidgetRenderer {
	header := container.NewHBox(c.monthSelect, c.yearSelect)
	content := container.NewVBox(header, c.daysGrid)
	return widget.NewSimpleRenderer(content)
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
		m.SaveConfig()
	})

	// Create calendar button
	m.calendarBtn = widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
		calendar := NewCustomCalendar(func(selectedDate time.Time) {
			m.datePickerEntry.SetText(selectedDate.Format("2006-01-02"))
			m.SaveConfig()
		})
		dialog.ShowCustom(locales.Translate("datesync.dialog.calendar"), locales.Translate("common.button.close"), calendar, m.Window)
	})

	// Create standard update button
	m.standardUpdateBtn = common.CreateSubmitButton(
		locales.Translate("datesync.button.startupdate"),
		func() {
			go m.standardUpdate()
		},
	)

	// Create custom date update button
	m.customDateUpdateBtn = common.CreateSubmitButton(
		locales.Translate("datesync.button.startcustomupdate"),
		func() {
			go m.customUpdate()
		},
	)

	// Add initial folder entries
	m.addExcludedFolderEntry()
	m.addCustomDateFolderEntry()
}

// standardUpdate performs the standard date synchronization
func (m *DateSyncModule) standardUpdate() {
	// Save configuration before starting
	m.SaveConfig()

	// Show progress dialog
	m.ShowProgressDialog(locales.Translate("datesync.dialog.header"))
	defer m.CloseProgressDialog()

	// Check database path
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Validation",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.nodbpath")), context)
		return
	}

	// Create database backup
	if err := m.dbMgr.BackupDatabase(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Backup",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		return
	}

	// Try to connect to database
	if err := m.dbMgr.Connect(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Connection",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.connectdb")), context)
		return
	}

	// Execute standard date sync
	if err := m.synchronizeDates(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Date Sync",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		return
	}
}

// customUpdate performs the custom date synchronization
func (m *DateSyncModule) customUpdate() {
	// Save configuration before starting
	m.SaveConfig()

	// Show progress dialog
	m.ShowProgressDialog(locales.Translate("datesync.dialog.header"))
	defer m.CloseProgressDialog()

	// Check database path
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Validation",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.nodbpath")), context)
		return
	}

	// Create database backup
	if err := m.dbMgr.BackupDatabase(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Backup",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		return
	}

	// Try to connect to database
	if err := m.dbMgr.Connect(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Connection",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.connectdb")), context)
		return
	}

	// Get custom date folders
	var customDateFoldersEntry []string
	for _, entry := range m.customDateFoldersEntry {
		if entry.Text != "" {
			customDateFoldersEntry = append(customDateFoldersEntry, entry.Text)
		}
	}

	// Parse custom date
	customDate, err := time.Parse("2006-01-02", m.datePickerEntry.Text)
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Date Parse",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("datesync.err.invaliddate")), context)
		return
	}

	// Execute custom date sync
	if err := m.setCustomDates(customDateFoldersEntry, customDate); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Custom Date Sync",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		return
	}
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
			locales.Translate("datesync.label.excluded"),
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
			locales.Translate("datesync.label.customdate"),
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

// addExcludedFolderEntry adds a new entry for excluded folder selection
func (m *DateSyncModule) addExcludedFolderEntry() {
	if len(m.excludedFoldersEntry) >= 6 {
		return
	}

	// Initialize entry field for folder selection
	entry := widget.NewEntry()
	folderField := common.CreateFolderSelectionFieldWithDelete(
		locales.Translate("datesync.label.excluded"),
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

// addCustomDateFolderEntry adds a new entry for custom date folder selection
func (m *DateSyncModule) addCustomDateFolderEntry() {
	if len(m.customDateFoldersEntry) >= 6 {
		return
	}

	// Initialize entry field for custom date folder selection
	entry := widget.NewEntry()
	customDateFolderField := common.CreateFolderSelectionFieldWithDelete(
		locales.Translate("datesync.label.customdate"),
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
			locales.Translate("datesync.label.excluded"),
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
			locales.Translate("datesync.label.customdate"),
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

// synchronizeDates updates the dates in the Rekordbox database
func (m *DateSyncModule) synchronizeDates() error {
	// Clear previous status messages
	m.ClearStatusMessages()

	// Add status message about starting the process
	m.AddInfoMessage(locales.Translate("common.status.start"))

	// Get excluded folders if enabled
	var excludedFoldersEntry []string
	if m.excludeFoldersCheck.Checked {
		for _, entry := range m.excludedFoldersEntry {
			if entry.Text != "" {
				excludedFoldersEntry = append(excludedFoldersEntry, entry.Text)
			}
		}
	}

	// Build the WHERE clause for excluded folders
	var excludeClause string
	if len(excludedFoldersEntry) > 0 {
		excludeConditions := make([]string, len(excludedFoldersEntry))
		for i, folder := range excludedFoldersEntry {
			excludeConditions[i] = fmt.Sprintf("FolderPath NOT LIKE '%s%%'", common.ToDbPath(folder, true))
		}
		excludeClause = " AND " + strings.Join(excludeConditions, " AND ")
	}

	// Query to update dates
	updateQuery := fmt.Sprintf(`
		UPDATE djmdContent
		SET DateCreated = DateModified
		WHERE FileType = 1%s
	`, excludeClause)

	// Execute the update
	if err := m.dbMgr.Execute(updateQuery); err != nil {
		m.AddErrorMessage(fmt.Sprintf(locales.Translate("datesync.err.dateupd"), err))
		return err
	}

	// Add success message
	m.AddInfoMessage(locales.Translate("common.status.completed"))
	m.CompleteProgressDialog()
	return nil
}

// setCustomDates sets custom dates for tracks in selected folders
func (m *DateSyncModule) setCustomDates(customDateFoldersEntry []string, customDate time.Time) error {
	// Clear previous status messages
	m.ClearStatusMessages()

	// Add status message about starting the process
	m.AddInfoMessage(locales.Translate("common.status.start"))

	// Check if any folders are selected
	if len(customDateFoldersEntry) == 0 {
		return fmt.Errorf(locales.Translate("datesync.err.nofolders"))
	}

	// Build the WHERE clause for custom date folders
	var conditions []string
	for _, folder := range customDateFoldersEntry {
		conditions = append(conditions, fmt.Sprintf("FolderPath LIKE '%s%%'", common.ToDbPath(folder, true)))
	}
	whereClause := strings.Join(conditions, " OR ")

	// Query to update dates
	updateQuery := fmt.Sprintf(`
		UPDATE djmdContent
		SET StockDate = ?
		WHERE FileType = 1 AND (%s)
	`, whereClause)

	// Execute the update
	if err := m.dbMgr.Execute(updateQuery, customDate.Format("2006-01-02")); err != nil {
		m.AddErrorMessage(fmt.Sprintf(locales.Translate("datesync.err.dateupd"), err))
		return err
	}

	// Add success message
	m.AddInfoMessage(locales.Translate("common.status.completed"))
	m.CompleteProgressDialog()
	return nil
}

// GetStatusMessagesContainer returns the status messages container for this module
func (m *DateSyncModule) GetStatusMessagesContainer() *common.StatusMessagesContainer {
	return m.ModuleBase.GetStatusMessagesContainer()
}

// AddInfoMessage adds an informational message to the status container
func (m *DateSyncModule) AddInfoMessage(message string) {
	m.ModuleBase.AddInfoMessage(message)
}

// AddErrorMessage adds an error message to the status container
func (m *DateSyncModule) AddErrorMessage(message string) {
	m.ModuleBase.AddErrorMessage(message)
}

// ClearStatusMessages clears all status messages
func (m *DateSyncModule) ClearStatusMessages() {
	m.ModuleBase.ClearStatusMessages()
}
