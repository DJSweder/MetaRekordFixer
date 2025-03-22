// modules/metadata_sync.go

package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	return m
}

// GetName returns the localized name of this module.
func (m *MetadataSyncModule) GetName() string {
	return locales.Translate("metsync.name.title")
}

// GetConfigName returns the module's configuration key.
func (m *MetadataSyncModule) GetConfigName() string {
	return "metadata_sync"
}

// GetIcon returns the module's icon resource.
func (m *MetadataSyncModule) GetIcon() fyne.Resource {
	return theme.MediaReplayIcon()
}

// GetContent returns the module's main UI content.
func (m *MetadataSyncModule) GetContent() fyne.CanvasObject {
	return m.Content
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

// initializeUI sets up the user interface of the module.
func (m *MetadataSyncModule) initializeUI() {
	// Source folder entry - creating it first so we can use it in the folder selection field
	m.entrySourceFolder = widget.NewEntry()
	// Standard create-change handler
	m.entrySourceFolder.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Recursive check
	m.checkRecursive = widget.NewCheck(locales.Translate("metsync.chkbox.recursive"),
		m.CreateBoolChangeHandler(func() {
			m.SaveConfig()
		}),
	)

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

	// Create submit button using standardized function
	m.btnSync = common.CreateSubmitButton(
		locales.Translate("metsync.button.sync"),
		func() {
			m.syncMetadata()
		},
	)

	// Create final layout using standardized module layout
	m.Content = common.CreateStandardModuleLayout(
		locales.Translate("metsync.label.info"),
		contentContainer,
		m.btnSync,
	)
}

// syncMetadata executes the metadata synchronization process.
func (m *MetadataSyncModule) syncMetadata() {
	// Save configuration before starting the process
	m.SaveConfig()

	// Disable the button and set icon after completion
	m.btnSync.Disable()
	defer func() {
		m.btnSync.Enable()
		m.btnSync.SetIcon(theme.ConfirmIcon())
	}()

	// Basic validation
	if m.entrySourceFolder.Text == "" {
		m.ErrorHandler.HandleError(errors.New(locales.Translate("metsync.err.emptypaths")), common.NewErrorContext(m.GetConfigName(), "Empty Paths"), m.Window, m.Status)
		return
	}

	// Show a progress dialog
	m.ShowProgressDialog(locales.Translate("metsync.dialog.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("metsync.err.panic"), r), common.NewErrorContext(m.GetConfigName(), "Panic"), m.Window, m.Status)
			}
		}()

		// Example progress
		m.UpdateProgressStatus(0.0, locales.Translate("metsync.status.start"))

		// Normalize paths
		sourcePath := common.NormalizePath(m.entrySourceFolder.Text)

		// Create a backup of the database
		err := m.dbMgr.BackupDatabase()
		if err != nil {
			m.CloseProgressDialog()
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Database Backup")
			context.Severity = common.ErrorWarning
			m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Connect to the database
		err = m.dbMgr.Connect()
		if err != nil {
			m.CloseProgressDialog()
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Database Connection")
			context.Severity = common.ErrorWarning
			m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
			return
		}

		// Ensure database connection is properly closed when done
		defer func() {
			if err := m.dbMgr.Close(); err != nil {
				m.ErrorHandler.HandleError(fmt.Errorf("Error finalizing database: %v", err),
					common.NewErrorContext(m.GetConfigName(), "Database Close"), m.Window, m.Status)
			}
		}()

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Update progress
		m.UpdateProgressStatus(0.1, locales.Translate("metsync.status.reading"))

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
			m.CloseProgressDialog()
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Database Query")
			context.Severity = common.ErrorWarning
			m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
			return
		}
		defer rows.Close()

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Prepare a slice to hold MP3 file information
		var mp3Files []struct {
			FileName    string
			AlbumID     sql.NullString
			ArtistID    sql.NullString
			OrgArtistID sql.NullString
			ReleaseDate sql.NullString
			Subtitle    sql.NullString
		}

		// Read all MP3 records from database
		for rows.Next() {
			var mp3File struct {
				FileName    string
				AlbumID     sql.NullString
				ArtistID    sql.NullString
				OrgArtistID sql.NullString
				ReleaseDate sql.NullString
				Subtitle    sql.NullString
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
				// Create error context with module name and operation
				context := common.NewErrorContext(m.GetConfigName(), "Rows Scan")
				context.Severity = common.ErrorWarning
				m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
				return
			}

			mp3Files = append(mp3Files, mp3File)
		}

		// Check if we found any MP3 files in the database
		totalDbFiles := len(mp3Files)
		if totalDbFiles == 0 {
			m.CloseProgressDialog()
			m.ErrorHandler.HandleError(errors.New(locales.Translate("metsync.err.nodbfiles")), common.NewErrorContext(m.GetConfigName(), "No DB Files"), m.Window, m.Status)
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Process each MP3 file and update corresponding FLAC files
		m.UpdateProgressStatus(0.2, locales.Translate("metsync.status.updating"))
		for i, mp3File := range mp3Files {
			// Check if operation was cancelled
			if m.IsCancelled() {
				m.CloseProgressDialog()
				return
			}

			// Update progress
			progress := 0.2 + (float64(i+1) / float64(totalDbFiles) * 0.8)
			m.UpdateProgressStatus(progress, fmt.Sprintf("%s: %d/%d", locales.Translate("metsync.status.process"), i+1, totalDbFiles))

			// Generate FLAC filename from MP3 filename
			flacFileName := strings.TrimSuffix(mp3File.FileName, filepath.Ext(mp3File.FileName)) + ".flac"

			// Update FLAC metadata in database
			err = m.dbMgr.Execute(`
				UPDATE djmdContent
				SET AlbumID = CAST(? AS INTEGER),
					ArtistID = CAST(? AS INTEGER),
					OrgArtistID = CAST(? AS INTEGER),
					ReleaseDate = ?,
					Subtitle = ?
				WHERE FileNameL = ?
			`,
				mp3File.AlbumID.String,
				mp3File.ArtistID.String,
				mp3File.OrgArtistID.String,
				mp3File.ReleaseDate,
				mp3File.Subtitle,
				flacFileName,
			)

			if err != nil {
				m.CloseProgressDialog()
				// Create error context with module name and operation
				context := common.NewErrorContext(m.GetConfigName(), "FLAC Update")
				context.Severity = common.ErrorWarning
				m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
				return
			}

			// Small delay to prevent database overload
			time.Sleep(10 * time.Millisecond)
		}

		// Mark done and update progress
		m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("metsync.status.completed"), totalDbFiles))

		m.CompleteProgressDialog()
	}()
}
