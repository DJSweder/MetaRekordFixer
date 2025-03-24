// modules/date_sync.go

package modules

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
)

// DateSyncModule implements a module for synchronizing dates in the Rekordbox database
type DateSyncModule struct {
	*common.ModuleBase
	dbMgr               *common.DBManager
	excludeFoldersCheck *widget.Check
	customDateCheck     *widget.Check
	excludedFolders     []*widget.Entry
	customDateFolders   []*widget.Entry
	datePicker          *widget.Entry
	foldersContainer    *fyne.Container
	customDateContainer *fyne.Container
	standardUpdateBtn   *widget.Button
	customDateUpdateBtn *widget.Button
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

// GetContent returns the module's UI content
func (m *DateSyncModule) GetContent() fyne.CanvasObject {
	return m.Content
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
	m.excludedFolders = nil
	m.foldersContainer.Objects = nil
	m.customDateFolders = nil
	m.customDateContainer.Objects = nil

	// Load excluded folders - use the dedicated checkbox state field
	excludeFoldersEnabled := cfg.GetBool("exclude_folders_enabled", false)
	m.excludeFoldersCheck.SetChecked(excludeFoldersEnabled)

	excludedFolders := cfg.Get("excluded_folders", "")

	// First add all non-empty folders in sequence
	if excludedFolders != "" {
		folders := []string{}
		// Filter out empty folders and collect valid ones
		for _, folder := range strings.Split(excludedFolders, "|") {
			if folder != "" {
				folders = append(folders, folder)
			}
		}
		// Add entries for valid folders in sequence
		for _, folder := range folders {
			m.addFolderEntryForConfig(folder, true)
		}
	}

	// Add one empty entry if we haven't reached the limit
	if len(m.excludedFolders) < 6 {
		m.addFolderEntryForConfig("", true)
	}

	// Load custom date folders - use the dedicated checkbox state field
	customDateEnabled := cfg.GetBool("custom_date_enabled", false)
	m.customDateCheck.SetChecked(customDateEnabled)

	customDateFolders := cfg.Get("custom_date_folders", "")

	// First add all non-empty folders in sequence
	if customDateFolders != "" {
		folders := []string{}
		// Filter out empty folders and collect valid ones
		for _, folder := range strings.Split(customDateFolders, "|") {
			if folder != "" {
				folders = append(folders, folder)
			}
		}
		// Add entries for valid folders in sequence
		for _, folder := range folders {
			m.addFolderEntryForConfig(folder, false)
		}
	}

	// Add one empty entry if we haven't reached the limit
	if len(m.customDateFolders) < 6 {
		m.addFolderEntryForConfig("", false)
	}

	// Load custom date
	customDate := cfg.Get("custom_date", "")
	if customDate != "" {
		m.datePicker.SetText(customDate)
	}

	m.foldersContainer.Refresh()
	m.customDateContainer.Refresh()
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
	cfg.SetBool("custom_date_enabled", m.customDateCheck.Checked)

	// Save excluded folders
	var excludedFolders []string
	for _, entry := range m.excludedFolders {
		if entry.Text != "" {
			excludedFolders = append(excludedFolders, entry.Text)
		}
	}
	if len(excludedFolders) > 0 {
		cfg.Set("excluded_folders", strings.Join(excludedFolders, "|"))
	}

	// Save custom date folders
	var customDateFolders []string
	for _, entry := range m.customDateFolders {
		if entry.Text != "" {
			customDateFolders = append(customDateFolders, entry.Text)
		}
	}
	if len(customDateFolders) > 0 {
		cfg.Set("custom_date_folders", strings.Join(customDateFolders, "|"))
	}

	// Save custom date only if it's set
	if m.datePicker.Text != "" && m.datePicker.Text != "YYYY-MM-DD" {
		cfg.Set("custom_date", m.datePicker.Text)
	}

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
		dayBtn := widget.NewButton(fmt.Sprintf("%d", day), func() {
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
	// Initialize UI components for configuration
	m.excludeFoldersCheck = widget.NewCheck(locales.Translate("datesync.chkbox.exception"),
		m.CreateBoolChangeHandler(func() { m.SaveConfig() }))

	m.customDateCheck = widget.NewCheck(locales.Translate("datesync.chkbox.dateset"),
		m.CreateBoolChangeHandler(func() { m.SaveConfig() }))

	m.datePicker = widget.NewEntry()
	m.datePicker.SetPlaceHolder("YYYY-MM-DD")
	m.datePicker.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	m.foldersContainer = container.NewVBox()
	m.customDateContainer = container.NewVBox()

	// Create main info label using standard component
	infoLabel := common.CreateDescriptionLabel(locales.Translate("datesync.mod.descr"))

	// Action buttons using standard components
	m.standardUpdateBtn = common.CreateSubmitButtonWithIcon(
		locales.Translate("datesync.date.dbupd"),
		nil,
		func() {
			go m.synchronizeDates()
		},
	)

	m.customDateUpdateBtn = common.CreateSubmitButtonWithIcon(
		locales.Translate("datesync.date.foldersupd"),
		nil,
		func() {
			customDateFolders := []string{}
			for _, entry := range m.customDateFolders {
				if entry.Text != "" {
					customDateFolders = append(customDateFolders, entry.Text)
				}
			}
			customDate, err := time.Parse("2006-01-02", m.datePicker.Text)
			if err != nil {
				m.ShowError(fmt.Errorf("%s", locales.Translate("datesync.err.invalidcustomdate")))
				return
			}
			go m.setCustomDates(customDateFolders, customDate)
		},
	)

	// Calendar date selection
	calendarBtn := widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
		// Declaration of the dialog variable
		var calendarDialog dialog.Dialog

		// Create a new custom calendar
		calendar := NewCustomCalendar(func(date time.Time) {
			m.datePicker.SetText(date.Format("2006-01-02"))
			m.SaveConfig()
			// Dialog bude zavřen automaticky po výběru data
			calendarDialog.Hide()
		})

		// Create the calendar content
		calendarContent := container.NewVBox(
			widget.NewLabel(locales.Translate("datesync.calendar.label")),
			calendar,
		)

		// Calendar dialog initialization
		calendarDialog = dialog.NewCustomWithoutButtons(
			locales.Translate("datesync.dialog.header"),
			calendarContent,
			m.Window,
		)

		// Settings for the calendar dialog
		calendarDialog.Resize(fyne.NewSize(350, 400))
		calendarDialog.Show()
	})

	// Assemble the layout
	m.Content = container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		m.excludeFoldersCheck,
		m.foldersContainer,
		container.NewHBox(layout.NewSpacer(), m.standardUpdateBtn),
		widget.NewSeparator(),
		m.customDateCheck,
		container.NewBorder(nil, nil, nil, calendarBtn, m.datePicker),
		m.customDateContainer,
		container.NewHBox(layout.NewSpacer(), m.customDateUpdateBtn),
		m.Status,
	)
}

// addFolderEntryForConfig adds a folder entry during config loading without triggering auto-add
func (m *DateSyncModule) addFolderEntryForConfig(folderPath string, isExcluded bool) {
	if (isExcluded && len(m.excludedFolders) >= 6) || (!isExcluded && len(m.customDateFolders) >= 6) {
		return
	}

	// Initialize entry field for folder selection
	entry := widget.NewEntry()
	entry.SetText(folderPath)

	var folderField fyne.CanvasObject
	if isExcluded {
		folderField = common.CreateFolderSelectionField(
			locales.Translate("datesync.label.excluded"),
			entry,
			func(path string) {
				entry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if !m.IsLoadingConfig && entry.Text != "" && len(m.excludedFolders) < 6 && entry == m.excludedFolders[len(m.excludedFolders)-1] {
					m.addExcludedFolderEntry()
				}
				m.SaveConfig()
			},
		)
		m.excludedFolders = append(m.excludedFolders, entry)
		m.foldersContainer.Add(folderField)
	} else {
		folderField = common.CreateFolderSelectionField(
			locales.Translate("datesync.label.customdate"),
			entry,
			func(path string) {
				entry.SetText(path)
				// Add new field if this is the last non-empty one and we haven't reached the limit
				if !m.IsLoadingConfig && entry.Text != "" && len(m.customDateFolders) < 6 && entry == m.customDateFolders[len(m.customDateFolders)-1] {
					m.addCustomDateFolderEntry()
				}
				m.SaveConfig()
			},
		)
		m.customDateFolders = append(m.customDateFolders, entry)
		m.customDateContainer.Add(folderField)
	}
}

// addExcludedFolderEntry adds a new entry for excluded folder selection
func (m *DateSyncModule) addExcludedFolderEntry() {
	if len(m.excludedFolders) >= 6 {
		return
	}

	// Initialize entry field for folder selection
	entry := widget.NewEntry()
	folderField := common.CreateFolderSelectionField(
		locales.Translate("datesync.label.excluded"),
		entry,
		func(path string) {
			entry.SetText(path)
			// Add new field if this is the last non-empty one and we haven't reached the limit
			if entry.Text != "" && len(m.excludedFolders) < 6 && entry == m.excludedFolders[len(m.excludedFolders)-1] {
				m.addExcludedFolderEntry()
			}
			m.SaveConfig()
		},
	)

	m.excludedFolders = append(m.excludedFolders, entry)
	m.foldersContainer.Add(folderField)
}

// addCustomDateFolderEntry adds a new entry for custom date folder selection
func (m *DateSyncModule) addCustomDateFolderEntry() {
	if len(m.customDateFolders) >= 6 {
		return
	}

	// Initialize entry field for custom date folder selection
	entry := widget.NewEntry()
	customDateFolderField := common.CreateFolderSelectionField(
		locales.Translate("datesync.label.customdate"),
		entry,
		func(path string) {
			entry.SetText(path)
			// Add new field if this is the last non-empty one and we haven't reached the limit
			if entry.Text != "" && len(m.customDateFolders) < 6 && entry == m.customDateFolders[len(m.customDateFolders)-1] {
				m.addCustomDateFolderEntry()
			}
			m.SaveConfig()
		},
	)

	m.customDateFolders = append(m.customDateFolders, entry)
	m.customDateContainer.Add(customDateFolderField)
}

// synchronizeDates updates the dates in the Rekordbox database
func (m *DateSyncModule) synchronizeDates() {
	// Zakážeme tlačítko během zpracování
	m.standardUpdateBtn.Disable()

	// After completion, we restore the button and set the success icon
	defer func() {
		m.standardUpdateBtn.Enable()
		// Změníme tlačítko na verzi s ikonou odškrtnutí po dokončení procesu
		m.standardUpdateBtn.SetIcon(theme.ConfirmIcon())
	}()

	m.ShowProgressDialog(locales.Translate("datesync.dialog.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("datesync.err.panic"), r), common.NewErrorContext(m.GetConfigName(), "Panic"), m.Window, m.Status)
			}
		}()

		// Get global configuration for database path
		globalConfig := m.ConfigMgr.GetGlobalConfig()
		if globalConfig.DatabasePath == "" {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf("%s", locales.Translate("datesync.err.nodbpath")))
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		m.UpdateProgressStatus(0.1, locales.Translate("datesync.db.backup"))
		// Use the DBManager's backup function
		err := m.dbMgr.BackupDatabase()
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dbackup"), err))
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		m.UpdateProgressStatus(0.2, locales.Translate("datesync.db.conn"))
		// Make sure we're connected to the database - Pass false to avoid read-only mode
		err = m.dbMgr.EnsureConnected(false)
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dbconn"), err))
			return
		}

		// Make sure to close the database connection when we're done
		defer func() {
			if closeErr := m.dbMgr.Close(); closeErr != nil {
				m.ErrorHandler.HandleError(fmt.Errorf("Error closing database: %v", closeErr),
					common.NewErrorContext(m.GetConfigName(), "Database Close"), m.Window, m.Status)
			}
		}()

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		whereClause := "WHERE ReleaseDate IS NOT NULL"

		// Add excluded folders to WHERE clause if enabled
		if m.excludeFoldersCheck.Checked {
			var excludedPaths []string
			for _, entry := range m.excludedFolders {
				if entry.Text != "" {
					excludedPaths = append(excludedPaths, entry.Text)
				}
			}
			if len(excludedPaths) > 0 {
				whereClause += " AND ("
				for i, path := range excludedPaths {
					if i > 0 {
						whereClause += " AND "
					}
					whereClause += fmt.Sprintf("FolderPath NOT LIKE '%s%%'", common.ToDbPath(path, true))
				}
				whereClause += ")"
			}
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		m.UpdateProgressStatus(0.4, locales.Translate("datesync.db.counting"))
		// Get count of affected records
		var count int
		countQuery := "SELECT COUNT(*) FROM djmdContent " + whereClause
		row := m.dbMgr.QueryRow(countQuery)
		err = row.Scan(&count)
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.entrycount"), err))
			return
		}

		if count == 0 {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf("%s", locales.Translate("datesync.err.noentryfound")))
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		m.UpdateProgressStatus(0.6, locales.Translate("datesync.db.update"))
		// Update dates
		updateQuery := "UPDATE djmdContent SET StockDate = ReleaseDate, DateCreated = ReleaseDate " + whereClause
		err = m.dbMgr.Execute(updateQuery)
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dateupd"), err))
			return
		}

		// Update progress and status
		m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("datesync.status.completed"), count))

		// Mark the progress dialog as completed instead of closing it
		m.CompleteProgressDialog()
	}()
}

// setCustomDates sets custom dates for tracks in selected folders
func (m *DateSyncModule) setCustomDates(customDateFolders []string, customDate time.Time) {
	// Disable the button during processing
	m.customDateUpdateBtn.Disable()

	// After completion, we restore the button and set the success icon
	defer func() {
		m.customDateUpdateBtn.Enable()
		// Změníme tlačítko na verzi s ikonou odškrtnutí po dokončení procesu
		m.customDateUpdateBtn.SetIcon(theme.ConfirmIcon())
	}()

	m.ShowProgressDialog(locales.Translate("datesync.diag.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("datesync.err.panic"), r), common.NewErrorContext(m.GetConfigName(), "Panic"), m.Window, m.Status)
			}
		}()

		// Get global configuration
		globalConfig := m.ConfigMgr.GetGlobalConfig()

		// Ensure we have a database manager
		if m.dbMgr == nil {
			var err error
			m.dbMgr, err = common.NewDBManager(globalConfig.DatabasePath, log.New(os.Stdout, "DateSync DB: ", log.LstdFlags), m.ErrorHandler)
			if err != nil {
				m.CloseProgressDialog()
				m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dbmanager"), err))
				return
			}
		}

		// Make sure to finalize the database connection when we're done
		defer m.dbMgr.Finalize()

		// Create backup
		m.UpdateProgressStatus(0.1, locales.Translate("datesync.db.backup"))
		// Use the DBManager's backup function
		err := m.dbMgr.BackupDatabase()
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dbackup"), err))
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Connect to database
		m.UpdateProgressStatus(0.2, locales.Translate("datesync.db.conn"))

		// Make sure we're connected to the database - Pass false to avoid read-only mode
		err = m.dbMgr.EnsureConnected(false)
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dbconn"), err))
			return
		}

		// Make sure to close the database connection when we're done
		defer func() {
			if closeErr := m.dbMgr.Close(); closeErr != nil {
				m.ErrorHandler.HandleError(fmt.Errorf("Error closing database: %v", closeErr),
					common.NewErrorContext(m.GetConfigName(), "Database Close"), m.Window, m.Status)
			}
		}()

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Build folder clauses for the WHERE condition
		folderClauses := make([]string, len(customDateFolders))
		for i, path := range customDateFolders {
			folderClauses[i] = fmt.Sprintf("FolderPath LIKE '%s%%'", common.ToDbPath(path, true))
		}

		whereClause := "WHERE " + strings.Join(folderClauses, " OR ")

		// Count affected records
		var count int
		countQuery := "SELECT COUNT(*) FROM djmdContent " + whereClause
		row := m.dbMgr.QueryRow(countQuery)
		err = row.Scan(&count)
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.entrycount"), err))
			return
		}

		if count == 0 {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf("%s", locales.Translate("datesync.err.noentryinfolders")))
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Set progress bar maximum value
		m.UpdateProgressStatus(0.3, locales.Translate("datesync.dates.updating"))

		// Format date string
		formattedDate := customDate.Format("2006-01-02")

		// Update database using parameterized query
		updateQuery := `
        UPDATE djmdContent
        SET StockDate = ?,
            DateCreated = ?
    ` + whereClause

		err = m.dbMgr.Execute(updateQuery, formattedDate, formattedDate)
		if err != nil {
			m.CloseProgressDialog()
			m.ShowError(fmt.Errorf(locales.Translate("datesync.err.dateupd"), err))
			return
		}

		// Update progress and status
		m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("datesync.status.completed"), count))

		// Mark the progress dialog as completed instead of closing it
		m.CompleteProgressDialog()
	}()
}
