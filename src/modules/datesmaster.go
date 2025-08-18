// modules/datesmaster.go

// Package modules provides functionality for different modules in the MetaRekordFixer application.
// This file contains the DatesMasterModule implementation for synchronizing dates in the Rekordbox database.

// This module changes *StockDate* (date added) and *DateCreated* (date created) for tracks in the Rekordbox database in 2 ways:
// 1. Copies values of release date fields with the option to exclude songs in folders (maximum 6 folders)
// 2. Sets custom date for tracks in specific folders (maximum 6 folders)

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

// DatesMasterModule implements a module for synchronizing dates in the Rekordbox database.
// It provides functionality to set standard dates based on release dates or custom dates for specific folders.
type DatesMasterModule struct {
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
		locales.Translate("datesmaster.month.jan"),
		locales.Translate("datesmaster.month.feb"),
		locales.Translate("datesmaster.month.mar"),
		locales.Translate("datesmaster.month.apr"),
		locales.Translate("datesmaster.month.may"),
		locales.Translate("datesmaster.month.jun"),
		locales.Translate("datesmaster.month.jul"),
		locales.Translate("datesmaster.month.aug"),
		locales.Translate("datesmaster.month.sep"),
		locales.Translate("datesmaster.month.okt"),
		locales.Translate("datesmaster.month.nov"),
		locales.Translate("datesmaster.month.dec"),
	}

	c.yearSelect = widget.NewSelect(years, func(s string) {
		year := 0
		fmt.Sscanf(s, "%d", &year)
		c.currentYear = year
		c.updateDays()
	})
	c.monthSelect = widget.NewSelect(months, func(s string) {
		months := map[string]time.Month{
			locales.Translate("datesmaster.month.jan"): time.January,
			locales.Translate("datesmaster.month.feb"): time.February,
			locales.Translate("datesmaster.month.mar"): time.March,
			locales.Translate("datesmaster.month.apr"): time.April,
			locales.Translate("datesmaster.month.may"): time.May,
			locales.Translate("datesmaster.month.jun"): time.June,
			locales.Translate("datesmaster.month.jul"): time.July,
			locales.Translate("datesmaster.month.aug"): time.August,
			locales.Translate("datesmaster.month.sep"): time.September,
			locales.Translate("datesmaster.month.okt"): time.October,
			locales.Translate("datesmaster.month.nov"): time.November,
			locales.Translate("datesmaster.month.dec"): time.December,
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
		locales.Translate("datesmaster.day.mon"),
		locales.Translate("datesmaster.day.tue"),
		locales.Translate("datesmaster.day.wed"),
		locales.Translate("datesmaster.day.thu"),
		locales.Translate("datesmaster.day.fri"),
		locales.Translate("datesmaster.day.sat"),
		locales.Translate("datesmaster.day.sun"),
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

// NewDatesMasterModule creates a new instance of DatesMasterModule.
// It initializes the UI components and loads the configuration.
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for module settings
//   - dbMgr: Database manager for database operations
//   - errorHandler: Error handler for error management
//
// Returns a new DatesMasterModule instance.
func NewDatesMasterModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *DatesMasterModule {
	m := &DatesMasterModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadCfg()

	return m
}

// GetName returns the localized name of the module.
// This implements the Module interface method.
func (m *DatesMasterModule) GetName() string {
	return locales.Translate("datesmaster.mod.name")
}

// GetConfigName returns the configuration identifier for the module.
// This implements the Module interface method and is used for configuration storage.
func (m *DatesMasterModule) GetConfigName() string {
	return "datesmaster"
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method.
func (m *DatesMasterModule) GetIcon() fyne.Resource {
	return theme.StorageIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI.
// Returns a canvas object containing the module's UI components.
func (m *DatesMasterModule) GetModuleContent() fyne.CanvasObject {
	// Left section - excluded folders
	leftHeader := widget.NewLabel(locales.Translate("datesmaster.label.leftpanel"))
	leftHeader.TextStyle = fyne.TextStyle{Bold: true}

	leftSection := container.NewVBox(
		leftHeader,
		m.excludeFoldersCheck,
		m.foldersContainer,
		container.NewHBox(layout.NewSpacer(), m.standardUpdateBtn),
	)

	// Right section - custom date folders
	rightHeader := widget.NewLabel(locales.Translate("datesmaster.label.rightpanel"))
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
		common.CreateDescriptionLabel(locales.Translate("datesmaster.label.info")),
		widget.NewSeparator(),
		contentContainer,
	)

	return moduleContent
}

// GetContent returns the module's main UI content including status messages.
// This implements the Module interface method.
// Returns a canvas object containing the complete module layout.
func (m *DatesMasterModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

func (m *DatesMasterModule) LoadCfg() {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load typed config from ConfigManager
	config, err := m.ConfigMgr.GetModuleCfg("datesmaster", m.GetConfigName())
	if err != nil {
		// This should not happen with the updated GetModuleCfg(), but handle gracefully
		return
	}

	// Cast to DatesMaster specific config
	if cfg, ok := config.(common.DatesMasterCfg); ok {
		// Update UI elements with loaded values
		m.excludeFoldersCheck.SetChecked(cfg.ExcludeFoldersEnabled.Value == "true")
		m.datePickerEntry.SetText(cfg.CustomDate.Value)
		
		// Parse excluded folders
		excludedFolderPaths := []string{}
		if cfg.ExcludedFolders.Value != "" {
			excludedFolderPaths = strings.Split(cfg.ExcludedFolders.Value, "|")
		}
		
		// Create excluded folders list
		m.foldersContainer, m.excludedFoldersEntry = common.CreateDynamicEntryList(
			m.Window,
			excludedFolderPaths,
			maxFolderEntries,
			func(entries []*widget.Entry) {
				m.excludedFoldersEntry = entries
				m.SaveCfg()
			},
		)
		
		// Parse custom date folders
		customFolderPaths := []string{}
		if cfg.CustomDateFolders.Value != "" {
			customFolderPaths = strings.Split(cfg.CustomDateFolders.Value, "|")
		}
		
		// Create custom date folders list
		m.customDateContainer, m.customDateFoldersEntry = common.CreateDynamicEntryList(
			m.Window,
			customFolderPaths,
			maxFolderEntries,
			func(entries []*widget.Entry) {
				m.customDateFoldersEntry = entries
				m.SaveCfg()
			},
		)
	}
}

// SaveCfg saves current UI state to typed configuration
func (m *DatesMasterModule) SaveCfg() {
	if m.IsLoadingConfig {
		return // Safeguard: no save if config is being loaded
	}

	// Collect excluded folders
	var excludedFoldersEntry []string
	for _, entry := range m.excludedFoldersEntry {
		if entry.Text != "" {
			excludedFoldersEntry = append(excludedFoldersEntry, entry.Text)
		}
	}

	// Collect custom date folders
	var customDateFoldersEntry []string
	for _, entry := range m.customDateFoldersEntry {
		if entry.Text != "" {
			customDateFoldersEntry = append(customDateFoldersEntry, entry.Text)
		}
	}

	// Get default configuration with all field definitions
	cfg := common.GetDefaultDatesMasterCfg()
	
	// Update only the values from current UI state
	cfg.CustomDate.Value = m.datePickerEntry.Text
	cfg.CustomDateFolders.Value = strings.Join(customDateFoldersEntry, "|")
	cfg.ExcludeFoldersEnabled.Value = fmt.Sprintf("%t", m.excludeFoldersCheck.Checked)
	cfg.ExcludedFolders.Value = strings.Join(excludedFoldersEntry, "|")

	// Save typed config via ConfigManager
	m.ConfigMgr.SaveModuleCfg("datesmaster", m.GetConfigName(), cfg)
}

// initializeUI sets up the user interface components for the module.
// It creates all UI elements, sets up event handlers, and initializes containers.
// This method is called during module creation.
func (m *DatesMasterModule) initializeUI() {
	// Create excluded folders checkbox
	m.excludeFoldersCheck = widget.NewCheck(locales.Translate("datesmaster.chkbox.exception"),
		m.CreateBoolChangeHandler(func() {
			m.SaveCfg()
		}),
	)

	// Create folders container
	m.foldersContainer = container.NewVBox()

	// Create custom date container
	m.customDateContainer = container.NewVBox()

	// Create date picker
	m.datePickerEntry = widget.NewEntry()
	m.datePickerEntry.SetPlaceHolder(locales.Translate("datesmaster.date.placeholder"))
	m.datePickerEntry.OnChanged = m.CreateChangeHandler(func() {

		// Limit input to 10 characters (YYYY-MM-DD)
		if len(m.datePickerEntry.Text) > 10 {
			m.datePickerEntry.SetText(m.datePickerEntry.Text[:10])
		}

		m.SaveCfg()
	})

	// Create calendar button
	m.calendarBtn = widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
		// Create dialog with calendar that will close automatically after date selection
		calendar := NewCustomCalendar(func(selectedDate time.Time) {
			m.datePickerEntry.SetText(selectedDate.Format("2006-01-02"))
			m.SaveCfg()
		})
		dlg := dialog.NewCustomWithoutButtons(locales.Translate("datesmaster.datepicker.header"), calendar, m.Window)
		calendar.onSelected = func(selectedDate time.Time) {
			m.datePickerEntry.SetText(selectedDate.Format("2006-01-02"))
			m.SaveCfg()
			dlg.Hide()
		}

		dlg.Show()

	})

	// Create standard update button
	m.standardUpdateBtn = common.CreateSubmitButton(locales.Translate("datesmaster.button.startupdate"), func() {
		m.Start("standard")
	},
	)

	// Create custom date update button
	m.customDateUpdateBtn = common.CreateSubmitButton(locales.Translate("datesmaster.button.startcustomupdate"), func() {
		m.Start("custom")
	},
	)

	// Initialize dynamic entry lists
	m.foldersContainer, m.excludedFoldersEntry = common.CreateDynamicEntryList(
		m.Window,
		[]string{},
		maxFolderEntries,
		func(entries []*widget.Entry) {
			m.excludedFoldersEntry = entries
			m.SaveCfg()
		},
	)

	m.customDateContainer, m.customDateFoldersEntry = common.CreateDynamicEntryList(
		m.Window,
		[]string{},
		maxFolderEntries,
		func(entries []*widget.Entry) {
			m.customDateFoldersEntry = entries
			m.SaveCfg()
		},
	)
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
func (m *DatesMasterModule) Start(mode string) {
	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate(mode); err != nil {
		return
	}

	// Show progress dialog with cancel support
	m.ShowProgressDialog(locales.Translate("datesmaster.dialog.header"))

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
func (m *DatesMasterModule) processStandardUpdate() {

	// Execute standard date sync
	m.StartProcessing(locales.Translate("common.status.updating"))
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

	// Update progress and complete dialog with final count
	m.CompleteProcessing(fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
	m.CompleteProgressDialog()

	// Update button to show completion
	common.UpdateButtonToCompleted(m.standardUpdateBtn)
}

// processCustomUpdate performs the custom date synchronization.
// It collects custom date folders, calls setCustomDates to perform the database update,
// handles errors and cancellation, and updates the UI with results.
// This method runs in a separate goroutine.
func (m *DatesMasterModule) processCustomUpdate() {

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
	m.StartProcessing(locales.Translate("common.status.updating"))
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

	// Update progress and complete dialog with final count
	m.CompleteProcessing(fmt.Sprintf(locales.Translate("common.status.completed"), updatedCount))
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
func (m *DatesMasterModule) setStandardDates() (int, error) {
	// Ensure database resources are properly released
	defer m.dbMgr.Finalize()

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
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesmaster.err.dbitemscount"), err)
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
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesmaster.err.dbupdate"), err)
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
func (m *DatesMasterModule) setCustomDates(customDateFoldersEntry []string, customDate time.Time) (int, error) {
	// Ensure database resources are properly released
	defer m.dbMgr.Finalize()

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
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesmaster.err.dbitemscount"), err)
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
		return 0, fmt.Errorf("%s: %w", locales.Translate("datesmaster.err.dbupdate"), err)
	}

	return totalCount, nil
}
