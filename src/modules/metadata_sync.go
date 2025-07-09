// Package modules provides functionality for different modules in the MetaRekordFixer application.
// Each module handles a specific task related to DJ database management and music file operations.
package modules

import (
	"errors"
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
// metadata between MP3 and FLAC files in a specified folder, ensuring consistent metadata
// across different formats of the same tracks.
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
// database manager, and error handler, sets up the UI components, and loads
// any saved configuration.
//
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for saving/loading module settings
//   - dbMgr: Database manager for accessing the DJ database
//   - errorHandler: Error handler for displaying and logging errors
//
// Returns:
//   - A fully initialized MetadataSyncModule instance
func NewMetadataSyncModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *MetadataSyncModule {
	m := &MetadataSyncModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	return m
}

// GetName returns the localized name of this module.
// This implements the Module interface method.
func (m *MetadataSyncModule) GetName() string {
	return locales.Translate("metsync.mod.name")
}

// GetConfigName returns the configuration key for this module.
// This key is used to store and retrieve module-specific configuration.
func (m *MetadataSyncModule) GetConfigName() string {
	return "metsync"
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method and provides the visual representation
// of this module in the UI.
func (m *MetadataSyncModule) GetIcon() fyne.Resource {
	return theme.HomeIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI
// containing the folder selection field, recursive checkbox, and submit button.
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
// This method returns the complete module layout with status messages container.
// 
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
// It sets the source folder path and recursive checkbox state based on saved configuration.
//
// Parameters:
//   - cfg: The module configuration to load
func (m *MetadataSyncModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Check if configuration is nil or Fields are not initialized
	if cfg.Fields == nil {
		cfg = common.NewModuleConfig()

		// Set default values with their definitions
		cfg.SetWithDefinitionAndActions("source_folder", "", "folder", true, "exists", []string{"start"})
		cfg.SetBoolWithDefinition("recursive", false, false, "none")

		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	}

	// Load source folder path
	m.sourceFolderEntry.SetText(cfg.Get("source_folder", ""))

	// Load recursive flag with default value false
	m.recursiveCheck.SetChecked(cfg.GetBool("recursive", false))
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
// It normalizes paths and saves boolean values for recursive processing.
// This method is called whenever UI components change to persist user preferences.
//
// Returns:
//   - A ModuleConfig containing all current UI settings
func (m *MetadataSyncModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	// Build fresh config
	cfg := m.ConfigMgr.GetModuleConfig(m.GetConfigName())

	// Save source folder path using NormalizePath
	cfg.SetWithDefinitionAndActions("source_folder",
		common.NormalizePath(m.sourceFolderEntry.Text),
		"folder",
		true,
		"exists", []string{"start"})

	// Save recursive flag
	cfg.SetBoolWithDefinition("recursive", m.recursiveCheck.Checked, false, "none")

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components.
// It creates and configures the entry fields, checkboxes, and buttons,
// and sets up their event handlers to respond to user interactions.
func (m *MetadataSyncModule) initializeUI() {
	// Initialize entry fields
	m.sourceFolderEntry = widget.NewEntry()
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Initialize folder selection button using standardized function
	folderSelectionField := common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
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
		go m.Start()
	},
	)
}

// Start performs the necessary steps before starting the main process.
// It saves the configuration, validates the inputs, informs the user, displays a dialog with a progress bar
// and starts the main process.
// Input validation also includes a test of the connection to the database and creating a backup of it.
// Additionally, it validates that the source folder contains MP3 files before proceeding.
//
// This method is called when the user clicks the submit button and runs the validation
// before launching the actual synchronization process in a goroutine.
func (m *MetadataSyncModule) Start() {

	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate("start"); err != nil {
		return
	}

	// Validate source folder contains MP3 files (specific to this module)
	sourcePath := common.NormalizePath(m.sourceFolderEntry.Text)
	mp3Files, err := common.ListFilesWithExtensions(sourcePath, []string{".mp3"}, m.recursiveCheck.Checked)
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "Find MP3 Files",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.err.noreadaccess"), err), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	if len(mp3Files) == 0 {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "Validate MP3 Files Exist",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("common.err.nofiles")), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Show progress dialog with cancel support
	m.ShowProgressDialog(locales.Translate("metsync.dialog.header"))

	// Start processing in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.CloseProgressDialog()
				context := &common.ErrorContext{
					Module:      m.GetName(),
					Operation:   "Metadata Sync",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
				m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
			}
		}()

		// Check if cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("common.status.stopped", 0, 0)
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Process metadata synchronization
		m.processMetadataSync(sourcePath)
	}()
}

// processMetadataSync handles the actual metadata synchronization process.
// It reads MP3 files from the database, updates corresponding FLAC files with matching metadata,
// and manages the progress dialog and status updates throughout the process.
//
// The method performs the following steps:
// 1. Queries the database for MP3 files in the specified folder
// 2. For each MP3 file, finds the corresponding FLAC file by name
// 3. Updates the FLAC file's metadata to match the MP3 file's metadata
// 4. Updates progress and handles cancellation throughout the process
//
// Parameters:
//   - sourcePath: The folder path to process for metadata synchronization
func (m *MetadataSyncModule) processMetadataSync(sourcePath string) {
	// Normalize paths
	sourcePath = common.NormalizePath(sourcePath)

	// Get the last folder name from the path
	lastFolderName := filepath.Base(sourcePath)

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
		AND c1.FolderPath LIKE '%/' || ? || '/%' OR c1.FolderPath LIKE '%/' || ? || ''
	`, lastFolderName, lastFolderName)

	if err != nil {
		m.CloseProgressDialog()
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "Database Query",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context) // This error is not wrapped, because DBMgr provides localized message for error dialog.
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
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
			m.CloseProgressDialog()
			context := &common.ErrorContext{
				Module:      m.GetName(),
				Operation:   "Read Database Records",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(err, context) // This error is not wrapped, because DBMgr provides localized message for error dialog.
			m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
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
	m.UpdateProgressStatus(0.3, locales.Translate("common.status.updating"))

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
			m.CloseProgressDialog()
			context := &common.ErrorContext{
				Module:      m.GetName(),
				Operation:   "Update FLAC Metadata",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(err, context) // This error is not wrapped, because DBMgr provides localized message for error dialog.
			m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
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
}
