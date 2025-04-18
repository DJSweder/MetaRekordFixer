// Package modules provides functionality for different modules in the MetaRekordFixer application.
package modules

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
)

// MetadataSyncModule handles metadata synchronization between different file formats.
// It implements the standard Module interface and provides functionality for synchronizing
// metadata between files in a specified folder.
type MetadataSyncModule struct {
	// ModuleBase is the base struct for all modules, which contains the module's window,
	// error handler, and configuration manager.
	*common.ModuleBase
	// dbMgr handles database operations
	dbMgr *common.DBManager
	// sourceFolderEntry is the entry field for source folder path
	sourceFolderEntry *widget.Entry
	// folderSelect is the folder selection button
	folderSelect *widget.Button
	// recursiveCheck determines if the sync should process subfolders
	recursiveCheck *widget.Check
	// submitBtn triggers the synchronization process
	submitBtn *widget.Button
}

// NewMetadataSyncModule creates a new instance of MetadataSyncModule.
// It initializes the module with the provided window, configuration manager,
// database manager, and error handler.
func NewMetadataSyncModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *MetadataSyncModule {
	m := &MetadataSyncModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Set database requirements
	m.SetDatabaseRequirements(true, false)

	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	return m
}

// GetName returns the localized name of this module.
func (m *MetadataSyncModule) GetName() string {
	return locales.Translate("metsync.mod.name")
}

// GetConfigName returns the configuration key for this module.
func (m *MetadataSyncModule) GetConfigName() string {
	return "metadata_sync"
}

// GetIcon returns the module's icon resource.
func (m *MetadataSyncModule) GetIcon() fyne.Resource {
	return theme.HomeIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI.
func (m *MetadataSyncModule) GetModuleContent() fyne.CanvasObject {
	// Create form with folder selection field
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("metsync.label.source"), Widget: container.NewBorder(nil, nil, nil, m.folderSelect, m.sourceFolderEntry)},
		},
	}

	// Create content container with form and additional widgets
	contentContainer := container.NewVBox(
		form,
		m.recursiveCheck,
	)

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("metsync.label.info")),
		widget.NewSeparator(),
		contentContainer,
	)

	// Add submit button with right alignment if provided
	if m.submitBtn != nil {
		buttonBox := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), m.submitBtn)
		moduleContent.Add(buttonBox)
	}

	return moduleContent
}

// GetContent returns the module's main UI content.
// Note: This module intentionally does not implement database availability checks
// as it operates without an active database connection until backup is created.
// The module also does not implement enable/disable logic for its controls
// as it needs to remain functional even when database is not available.
func (m *MetadataSyncModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig applies the configuration to the UI components.
// If the configuration is nil, it creates a new one with default values.
func (m *MetadataSyncModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Check if configuration is nil or Extra field is not initialized
	if cfg.Extra == nil {
		// Initialize with default values and save them
		cfg = common.NewModuleConfig()
		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
		return
	}

	// Load source folder path
	if folder, ok := cfg.Extra["source_folder"]; ok && folder != "" {
		m.sourceFolderEntry.SetText(folder)
	}

	// Load recursive flag with default value false
	m.recursiveCheck.SetChecked(cfg.GetBool("recursive", false))
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
// It normalizes paths and saves boolean values for recursive processing.
func (m *MetadataSyncModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	// Build fresh config
	cfg := common.NewModuleConfig()

	// Save source folder path using NormalizePath
	cfg.Set("source_folder", common.NormalizePath(m.sourceFolderEntry.Text))

	// Save recursive flag
	cfg.SetBool("recursive", m.recursiveCheck.Checked)

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components.
// It creates and configures the entry fields, checkboxes, and buttons,
// and sets up their event handlers.
func (m *MetadataSyncModule) initializeUI() {
	// Initialize entry fields
	m.sourceFolderEntry = widget.NewEntry()
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Initialize folder selection button using standardized function
	folderSelectionField := common.CreateFolderSelectionField(
		locales.Translate("metsync.label.source"),
		m.sourceFolderEntry,
		func(path string) {
			m.sourceFolderEntry.SetText(common.NormalizePath(path))
			m.SaveConfig()
		},
	)
	m.folderSelect = folderSelectionField.(*fyne.Container).Objects[1].(*widget.Button)

	// Initialize recursive checkbox using standardized function
	m.recursiveCheck = common.CreateCheckbox(locales.Translate("metsync.chkbox.recursive"), func(checked bool) {
		m.SaveConfig()
	})

	// Initialize sync button
	m.submitBtn = common.CreateSubmitButton(locales.Translate("metsync.button.sync"), func() {
		go func() {
			if err := m.Start(); err != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Metadata Sync",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				}
				m.ErrorHandler.ShowStandardError(err, context)
			} else {
				common.UpdateButtonToCompleted(m.submitBtn)
			}
		}()
	})
}

// GetStatusMessagesContainer returns the status messages container.
func (m *MetadataSyncModule) GetStatusMessagesContainer() *common.StatusMessagesContainer {
	return m.ModuleBase.GetStatusMessagesContainer()
}

// AddInfoMessage adds an information message to the status messages container.
func (m *MetadataSyncModule) AddInfoMessage(message string) {
	m.ModuleBase.AddInfoMessage(message)
}

// AddErrorMessage adds an error message to the status messages container.
func (m *MetadataSyncModule) AddErrorMessage(message string) {
	m.ModuleBase.AddErrorMessage(message)
}

// ClearStatusMessages clears all status messages.
func (m *MetadataSyncModule) ClearStatusMessages() {
	m.ModuleBase.ClearStatusMessages()
}

// Start initiates the metadata synchronization process.
// It validates the input, clears previous status messages,
// and executes the synchronization.
func (m *MetadataSyncModule) Start() error {
	// Disable the button during processing
	m.submitBtn.Disable()
	defer func() {
		m.submitBtn.Enable()
		m.submitBtn.SetIcon(theme.ConfirmIcon())
	}()

	// Save configuration before starting
	m.SaveConfig()

	// Clear previous status messages
	m.ClearStatusMessages()

	// Validate source folder
	if m.sourceFolderEntry.Text == "" {
		m.AddErrorMessage(locales.Translate("metsync.err.nosource"))
		return fmt.Errorf("source folder not selected")
	}

	// Show progress dialog with cancel support
	m.ShowProgressDialog(locales.Translate("metsync.dialog.header"))

	// Start processing in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Metadata Sync",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
			}
		}()

		// Add initial status message
		m.AddInfoMessage(locales.Translate("common.status.start"))

		// Create database backup
		m.UpdateProgressStatus(0.1, fmt.Sprintf(locales.Translate("common.db.backupcreate")))
		if err := m.dbMgr.BackupDatabase(); err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.backupdb"), err), &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Database Backup",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			})
			return
		}
		m.ModuleBase.AddInfoMessage(locales.Translate("common.db.backupdone"))

		// Check if cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("metsync.status.stopped", 0, 0)
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Connect to database
		m.UpdateProgressStatus(0.2, fmt.Sprintf(locales.Translate("common.db.conn")))
		if err := m.dbMgr.Connect(); err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.dbconn"), err), &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Database Connection",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			})
			return
		}
		defer m.dbMgr.Finalize()

		// Check if cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("metsync.status.stopped", 0, 0)
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Normalize paths
		sourcePath := common.NormalizePath(m.sourceFolderEntry.Text)

		// Prepare a slice to hold MP3 file information
		var mp3Files []struct {
			FileName    string
			AlbumID     common.NullString
			ArtistID    common.NullString
			OrgArtistID common.NullString
			ReleaseDate common.NullString
			Subtitle    common.NullString
		}

		// Query to get MP3 files from database
		rows, err := m.dbMgr.Query(`
			SELECT 
				c1.FileNameL,
				c1.AlbumID,
				c1.ArtistID,
				c1.OrgArtistID,
				c1.ReleaseDate,
				c1.Subtitle
			FROM djmdContent c1
			WHERE c1.FileNameL LIKE '%.mp3'
			AND c1.FolderPath LIKE ? || '%'
		`, common.ToDbPath(sourcePath, true))

		if err != nil {
			m.ErrorHandler.ShowStandardError(err, &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Database Query",
				Severity:    common.SeverityWarning,
				Recoverable: true,
			})
			return
		}
		defer rows.Close()

		// Read all MP3 records from database
		for rows.Next() {
			var mp3File struct {
				FileName    string
				AlbumID     common.NullString
				ArtistID    common.NullString
				OrgArtistID common.NullString
				ReleaseDate common.NullString
				Subtitle    common.NullString
			}

			err := rows.Scan(
				&mp3File.FileName,
				&mp3File.AlbumID,
				&mp3File.ArtistID,
				&mp3File.OrgArtistID,
				&mp3File.ReleaseDate,
				&mp3File.Subtitle,
			)

			if err != nil {
				m.ErrorHandler.ShowStandardError(err, &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Read Database Records",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				})
				return
			}

			mp3Files = append(mp3Files, mp3File)
		}

		// Check if we found any MP3 files in the database
		totalDbFiles := len(mp3Files)
		if totalDbFiles == 0 {
			// Add error message to status
			m.AddErrorMessage(locales.Translate("common.err.noentryfound"))

			// Update progress and complete dialog
			m.UpdateProgressStatus(1.0, locales.Translate("common.err.noentryfound"))
			m.CompleteProgressDialog()
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Add status message about number of files found
		m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.filesfound"), totalDbFiles))

		// Process each MP3 file and update corresponding FLAC files
		m.UpdateProgressStatus(0.3, fmt.Sprintf(locales.Translate("common.status.updating")))

		// Add status message about starting the update process
		m.AddInfoMessage(locales.Translate("common.status.updating"))

		for i, mp3File := range mp3Files {
			// Update progress
			progress := 0.3 + (float64(i+1) / float64(totalDbFiles) * 0.7)
			m.UpdateProgressStatus(progress, fmt.Sprintf(locales.Translate("common.status.progress"), i+1, totalDbFiles))

			// Check if cancelled
			if m.IsCancelled() {
				m.HandleProcessCancellation("common.status.stopped", i, totalDbFiles)
				common.UpdateButtonToCompleted(m.submitBtn)
				return
			}

			// Generate FLAC filename from MP3 filename
			flacFileName := strings.TrimSuffix(mp3File.FileName, filepath.Ext(mp3File.FileName)) + ".flac"

			// Update the FLAC file with the metadata from the MP3 file
			err = m.dbMgr.Execute(`
					UPDATE djmdContent
					SET AlbumID = CAST(? AS INTEGER),
						ArtistID = CAST(? AS INTEGER),
						OrgArtistID = CAST(? AS INTEGER),
						ReleaseDate = ?,
						Subtitle = ?
					WHERE FileNameL = ?
				`,
				mp3File.AlbumID.ValueOrNil(),
				mp3File.ArtistID.ValueOrNil(),
				mp3File.OrgArtistID.ValueOrNil(),
				mp3File.ReleaseDate.ValueOrNil(),
				mp3File.Subtitle.ValueOrNil(),
				flacFileName,
			)

			if err != nil {
				m.ErrorHandler.ShowStandardError(err, &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Update FLAC Metadata",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				})
				return
			}

			// Small delay to prevent database overload
			time.Sleep(10 * time.Millisecond)
		}

		// Update progress to completion
		m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("common.status.completed"), totalDbFiles))
		m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.completed"), totalDbFiles))

		// Mark the progress dialog as completed and update button
		m.CompleteProgressDialog()
		common.UpdateButtonToCompleted(m.submitBtn)
	}()

	return nil
}
