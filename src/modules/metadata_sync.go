// modules/metadata_sync.go

package modules

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MetadataSyncModule handles metadata synchronization between different file formats.
type MetadataSyncModule struct {
	*common.ModuleBase // Embedded pointer to shared base
	dbMgr              *common.DBManager
	entrySourceFolder  *widget.Entry
	checkRecursive     *widget.Check
	btnSync            *widget.Button
}

// NewMetadataSyncModule creates a new instance of MetadataSyncModule.
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

// GetConfigName returns the module's configuration key.
func (m *MetadataSyncModule) GetConfigName() string {
	return "metadata_sync"
}

// GetIcon returns the module's icon resource.
func (m *MetadataSyncModule) GetIcon() fyne.Resource {
	return theme.HomeIcon()
}

// GetModuleContent returns the module's specific content without status messages
// This implements the method from ModuleBase to provide the module-specific UI
func (m *MetadataSyncModule) GetModuleContent() fyne.CanvasObject {
	// Create folder selection field using standardized function
	folderSelectionField := common.CreateFolderSelectionField(
		locales.Translate("metsync.label.source"),
		m.entrySourceFolder,
		func(path string) {
			// Normalize path, save config
			m.entrySourceFolder.SetText(common.NormalizePath(path))
			m.SaveConfig()
		},
	)

	// Create form with folder selection field
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("metsync.label.source"), Widget: container.NewBorder(nil, nil, nil, folderSelectionField.(*fyne.Container).Objects[1].(*widget.Button), m.entrySourceFolder)},
		},
	}

	// Create additional widgets array
	additionalWidgets := []fyne.CanvasObject{
		m.checkRecursive,
	}

	// Create content container with form and additional widgets
	contentContainer := container.NewVBox(
		form,
	)

	// Add additional widgets to content container
	for _, widget := range additionalWidgets {
		contentContainer.Add(widget)
	}

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		widget.NewLabel(locales.Translate("metsync.label.info")),
		widget.NewSeparator(),
		contentContainer,
	)

	// Add submit button with right alignment if provided
	if m.btnSync != nil {
		buttonBox := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), m.btnSync)
		moduleContent.Add(buttonBox)
	}

	return moduleContent
}

// GetContent returns the module's main UI content.
func (m *MetadataSyncModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig applies the configuration to the UI components.
func (m *MetadataSyncModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	if common.IsNilConfig(cfg) {
		// Initialize with default values and save them
		cfg = common.NewModuleConfig()
		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
		return
	}

	src := cfg.Get("source_folder", "")
	if src != "" {
		m.entrySourceFolder.SetText(src)
	}

	m.checkRecursive.SetChecked(cfg.GetBool("recursive", false))
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
func (m *MetadataSyncModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	// Build fresh config
	cfg := common.NewModuleConfig()

	// Use NormalizePath which now handles empty strings correctly
	cfg.Set("source_folder", common.NormalizePath(m.entrySourceFolder.Text))
	cfg.SetBool("recursive", m.checkRecursive.Checked)

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components.
func (m *MetadataSyncModule) initializeUI() {
	// Initialize entry fields
	m.entrySourceFolder = widget.NewEntry()
	m.checkRecursive = widget.NewCheck(locales.Translate("metsync.chkbox.recursive"), nil)
	m.checkRecursive.OnChanged = m.CreateBoolChangeHandler(func() {
		m.SaveConfig()
	})

	// Create sync button
	m.btnSync = common.CreateSubmitButtonWithIcon(
		locales.Translate("metsync.button.sync"),
		theme.MediaPlayIcon(),
		func() {
			if err := m.Start(); err != nil {
				m.ErrorHandler.ShowStandardError(err, nil)
			}
		},
	)

	// Disable sync button if database is not available
	if m.dbMgr == nil {
		m.btnSync.Disable()
		m.AddErrorMessage(locales.Translate("common.err.dbnotset"))
	}
}

// Start initiates the metadata synchronization process
func (m *MetadataSyncModule) Start() error {
	// Save configuration before starting
	m.SaveConfig()

	// Clear previous status messages
	m.ClearStatusMessages()

	// Validate source folder
	if m.entrySourceFolder.Text == "" {
		m.AddErrorMessage(locales.Translate("metsync.err.nosource"))
		return fmt.Errorf("source folder not selected")
	}

	// Create progress dialog
	progress := common.NewProgressDialog(
		m.Window,
		locales.Translate("metsync.progress.title"),
		locales.Translate("metsync.progress.msg"),
		func() {
			// Cancel handler - currently empty as we don't support cancellation
		},
	)
	defer progress.Hide()

	// Create error context for database operations
	context := &common.ErrorContext{
		Module:      m.GetConfigName(),
		Operation:   "Metadata Sync",
		Severity:    common.ErrorCritical,
		Recoverable: false,
	}

	// Create database backup
	m.UpdateProgressStatus(0.1, locales.Translate("common.db.backup"))
	if err := m.dbMgr.BackupDatabase(); err != nil {
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.backupdb"), err), context)
		return err
	}
	m.ModuleBase.AddInfoMessage(locales.Translate("common.db.backupdone"))

	// Connect to database
	m.UpdateProgressStatus(0.2, locales.Translate("common.db.conn"))
	if err := m.dbMgr.Connect(); err != nil {
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.dbconn"), err), context)
		return err
	}
	defer m.dbMgr.Finalize()

	// Execute metadata sync
	if err := m.syncMetadata(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Metadata Sync",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		return err
	}

	return nil
}

// syncMetadata executes the metadata synchronization process.
func (m *MetadataSyncModule) syncMetadata() error {
	// Disable the button and set icon after completion
	m.btnSync.Disable()
	defer func() {
		m.btnSync.Enable()
		m.btnSync.SetIcon(theme.ConfirmIcon())
	}()

	// Basic validation
	if m.entrySourceFolder.Text == "" {
		return errors.New(locales.Translate("metsync.err.emptypaths"))
	}

	// Clear previous status messages
	m.ClearStatusMessages()

	// Add real status message about starting the synchronization
	m.AddInfoMessage(locales.Translate("common.status.start"))

	// Normalize paths
	sourcePath := common.NormalizePath(m.entrySourceFolder.Text)

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
		return err
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
			return err
		}

		mp3Files = append(mp3Files, mp3File)
	}

	// Check if we found any MP3 files in the database
	totalDbFiles := len(mp3Files)
	if totalDbFiles == 0 {
		return errors.New(locales.Translate("common.err.noentryfound"))
	}

	// Add status message about number of files found
	m.AddInfoMessage(fmt.Sprint(locales.Translate("common.status.filesfound"), totalDbFiles))

	// Process each MP3 file and update corresponding FLAC files
	m.UpdateProgressStatus(0.2, locales.Translate("common.status.updating"))

	// Add status message about starting the update process
	m.AddInfoMessage(locales.Translate("common.status.updating"))

	for i, mp3File := range mp3Files {
		// Update progress
		progress := 0.2 + (float64(i+1) / float64(totalDbFiles) * 0.8)
		m.UpdateProgressStatus(progress, fmt.Sprint(locales.Translate("metsync.status.process"), i+1, "/", totalDbFiles))

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
			return err
		}

		// Small delay to prevent database overload
		time.Sleep(10 * time.Millisecond)
	}

	// Mark done and update progress
	m.UpdateProgressStatus(1.0, fmt.Sprint(locales.Translate("common.status.completed"), totalDbFiles))

	// Add final status message about completion
	m.AddInfoMessage(fmt.Sprint(locales.Translate("common.status.completed"), totalDbFiles))

	m.CompleteProgressDialog()
	return nil
}
