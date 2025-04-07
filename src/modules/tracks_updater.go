package modules

import (
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// TracksUpdater handles updating track file paths and formats in the database
// Implements the standard Module interface
// TracksUpdater is a module that handles updating track file paths and formats in the database.
// It implements the standard Module interface.
type TracksUpdater struct {
	// ModuleBase is the base struct for all modules, which contains the module's window, error handler, and
	// configuration manager.
	*common.ModuleBase
	dbMgr             *common.DBManager
	playlistSelect    *widget.Select
	folderEntry       *widget.Entry
	folderSelect      *widget.Button
	submitBtn         *widget.Button
	playlists         []common.PlaylistItem
	pendingPlaylistID string // Temporary storage for playlist ID
}

// NewTracksUpdater creates a new instance of TracksUpdater.
func NewTracksUpdater(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *TracksUpdater {
	m := &TracksUpdater{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Initialize variables before initializeUI
	m.folderEntry = widget.NewEntry()

	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	return m
}

// GetName returns the localized name of this module.
func (m *TracksUpdater) GetName() string {
	return locales.Translate("updater.mod.name")
}

// GetConfigName returns the module's configuration key.
func (m *TracksUpdater) GetConfigName() string {
	return "tracks_updater"
}

// GetIcon returns the module's icon resource.
func (m *TracksUpdater) GetIcon() fyne.Resource {
	return theme.SearchReplaceIcon()
}

// GetModuleContent returns the module's specific content without status messages
// This implements the method from ModuleBase to provide the module-specific UI
func (m *TracksUpdater) GetModuleContent() fyne.CanvasObject {
	// Create form with playlist selector and folder selection field
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("updater.label.replaced"), Widget: m.playlistSelect},
			{Text: locales.Translate("updater.label.newfiles"), Widget: container.NewBorder(nil, nil, nil, m.folderSelect, m.folderEntry)},
		},
	}

	// Create content container with form
	contentContainer := container.NewVBox(
		form,
	)

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("updater.label.info")),
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

// GetContent returns the module's main UI content and initializes database connection.
func (m *TracksUpdater) GetContent() fyne.CanvasObject {
	// Check database requirements
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Validation",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.nodbpath")), context)
		common.DisableModuleControls(m.playlistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
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
		common.DisableModuleControls(m.playlistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}
	defer m.dbMgr.Finalize()

	// Load playlists
	if err := m.loadPlaylists(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Access",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.playlistload")), context)
		common.DisableModuleControls(m.playlistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}

	// Enable interactive components if all checks passed
	m.playlistSelect.Enable()
	m.submitBtn.Enable()

	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig applies the configuration to the UI components.
func (m *TracksUpdater) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Check if Extra field is initialized
	if cfg.Extra == nil {
		return
	}

	if folder, ok := cfg.Extra["folder"]; ok && folder != "" {
		m.folderEntry.SetText(folder)
	}

	if playlistID, ok := cfg.Extra["playlist_id"]; ok && playlistID != "" {
		m.pendingPlaylistID = playlistID // Save temporary PlaylistID for later use
	}
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
func (m *TracksUpdater) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	cfg := common.NewModuleConfig()

	// Save folder path using NormalizePath which now handles empty strings correctly
	cfg.Set("folder", common.NormalizePath(m.folderEntry.Text))

	// Save playlist ID
	if m.playlistSelect.Selected != "" {
		for _, playlist := range m.playlists {
			if playlist.Path == m.playlistSelect.Selected {
				cfg.Set("playlist_id", playlist.ID)
				break
			}
		}
	}

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components.
func (m *TracksUpdater) initializeUI() {
	// Create a text entry for the user to input the folder path.
	// When the user changes the text in the entry, save the config.
	m.folderEntry = widget.NewEntry()
	m.folderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Create a disabled select widget for the user to choose a playlist.
	// When the user chooses a playlist, save the config.
	// The select widget is disabled to prevent the user from changing the playlist
	// before the module is fully loaded.
	m.playlistSelect = common.CreateDisabledSelect([]string{}, m.CreateSelectionChangeHandler(func() {
		m.SaveConfig()
	}), "common.select.plsplacehldrinact")

	// Create a folder selection field using the standardized function.
	// The folder selection field consists of a button and a text entry.
	// When the user clicks the button, open a file dialog for the user to choose a folder.
	// When the user chooses a folder, set the text entry to the path of the chosen folder
	// and save the config.
	folderSelectionField := common.CreateFolderSelectionField(
		locales.Translate("updater.folder.newfiles"),
		m.folderEntry,
		func(path string) {
			m.folderEntry.SetText(path)
			m.SaveConfig()
		},
	)

	// Store the button reference for backward compatibility
	m.folderSelect = folderSelectionField.(*fyne.Container).Objects[1].(*widget.Button)

	// Create a disabled submit button using the standardized function.
	// The submit button is disabled to prevent the user from starting the module
	// before the module is fully loaded.
	// When the user clicks the submit button, start the module.
	m.submitBtn = common.CreateDisabledSubmitButton(
		locales.Translate("updater.button.libupd"),
		func() {
			go m.Start()
		},
	)
}

// Start handles the module's main functionality

func (m *TracksUpdater) GetStatusMessagesContainer() *common.StatusMessagesContainer {
	return m.ModuleBase.GetStatusMessagesContainer()
}

func (m *TracksUpdater) AddInfoMessage(message string) {
	m.ModuleBase.AddInfoMessage(message)
}

func (m *TracksUpdater) AddErrorMessage(message string) {
	m.ModuleBase.AddErrorMessage(message)
}

func (m *TracksUpdater) ClearStatusMessages() {
	m.ModuleBase.ClearStatusMessages()
}

// getFileType is used to translate the file type according to its extension into an identifier that is stored in updated records in djmdContent in the database
func getFileType(ext string) int {
	switch strings.ToLower(ext) {
	case ".mp3":
		return 1
	case ".m4a":
		return 4
	case ".flac":
		return 5
	case ".wav":
		return 11
	case ".aiff":
		return 12
	default:
		return 0
	}
}

func (m *TracksUpdater) loadPlaylists() error {
	err := m.dbMgr.Connect()
	if err != nil {
		return fmt.Errorf("%s %w", locales.Translate("common.err.connectdb"), err)
	}
	defer m.dbMgr.Finalize()

	rows, err := m.dbMgr.Query(`
        SELECT p1.ID, p1.Name, p1.ParentID,
        CASE
            WHEN p2.Name IS NOT NULL THEN p2.Name || ' > ' || p1.Name
            ELSE p1.Name
        END as Path
        FROM djmdPlaylist p1
        LEFT JOIN djmdPlaylist p2 ON p1.ParentID = p2.ID
        ORDER BY
            CASE WHEN p2.ID IS NULL THEN p1.Seq ELSE p2.Seq END,
            CASE WHEN p2.ID IS NULL THEN 0 ELSE p1.Seq + 1 END
    `)
	if err != nil {
		return fmt.Errorf("%s %w", locales.Translate("common.err.dbread"), err)
	}
	defer rows.Close()

	m.playlists = make([]common.PlaylistItem, 0)
	var playlistPaths []string
	for rows.Next() {
		var p common.PlaylistItem
		if err := rows.Scan(&p.ID, &p.Name, &p.ParentID, &p.Path); err != nil {
			return fmt.Errorf("%s %w", locales.Translate("common.err.dbread"), err)
		}
		m.playlists = append(m.playlists, p)
		playlistPaths = append(playlistPaths, p.Path)
	}

	m.playlistSelect.Options = playlistPaths
	m.playlistSelect.Enable()
	m.playlistSelect.PlaceHolder = locales.Translate("common.select.plsplaceholder")

	// Apply pending PlaylistID after successful loading
	if m.pendingPlaylistID != "" {
		for i, playlist := range m.playlists {
			if playlist.ID == m.pendingPlaylistID {
				if i < len(m.playlistSelect.Options) {
					m.playlistSelect.SetSelected(m.playlistSelect.Options[i])
				}
				break
			}
		}
		m.pendingPlaylistID = "" // Reset pending PlaylistID
	} else if len(m.playlistSelect.Options) > 0 {
		m.playlistSelect.SetSelectedIndex(0)
	}

	return nil
}
func (m *TracksUpdater) Start() {
	// Save the configuration before starting the process so that the most recent playlist and folder path are used.
	m.SaveConfig()

	// Clear any previous status messages.
	m.ClearStatusMessages()

	// Get the selected playlist.
	if m.playlistSelect.Selected == "" {
		m.AddErrorMessage(locales.Translate("updater.err.noplaylist"))
		return
	}

	// Get the folder path.
	folderEntry := m.folderEntry.Text
	if folderEntry == "" {
		m.AddErrorMessage(locales.Translate("updater.err.nofolder"))
		return
	}

	// Show the progress dialog.
	m.ShowProgressDialog(locales.Translate("updater.dialog.header"))

	// Start processing in a goroutine.
	go func() {
		// Track the number of updated files.
		updateCount := 0

		defer func() {
			// Catch any panics or errors and show an error message.
			if r := recover(); r != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Update Process",
					Severity:    common.ErrorCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
			}
		}()

		// Backup the database.
		m.UpdateProgressStatus(0.1, locales.Translate("common.db.backupcreate"))
		err := m.dbMgr.BackupDatabase()
		if err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.ShowError(fmt.Errorf(locales.Translate("common.err.backupdb"), err))
			return
		}

		// Inform about the successful backup.
		m.ModuleBase.AddInfoMessage(locales.Translate("common.db.backupdone"))

		// Check if the operation was cancelled.
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len([]int{}))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Connect to the database.
		m.UpdateProgressStatus(0.2, locales.Translate("common.db.conn"))
		err = m.dbMgr.Connect()
		if err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.ShowError(fmt.Errorf(locales.Translate("common.err.connectdb"), err))
			return
		}

		// Get the selected playlist.
		m.UpdateProgressStatus(0.3, locales.Translate("updater.tracks.getplaylist"))
		selectedPlaylist := ""
		for _, p := range m.playlists {
			if p.Path == m.playlistSelect.Selected {
				selectedPlaylist = p.ID
				break
			}
		}
		if selectedPlaylist == "" {
			m.CloseProgressDialog()
			m.ModuleBase.AddErrorMessage(locales.Translate("updater.err.noplaylist"))
			return
		}

		// Check if the operation was cancelled.
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len([]int{}))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Get the tracks from the playlist.
		m.UpdateProgressStatus(0.4, locales.Translate("updater.status.gettracks"))
		rows, err := m.dbMgr.Query(`
		SELECT c.ID, c.FileNameL
		FROM djmdContent c
		JOIN djmdSongPlaylist sp ON c.ID = sp.ContentID
		WHERE sp.PlaylistID = ?
	`, selectedPlaylist)
		if err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.ShowError(fmt.Errorf(locales.Translate("common.err.dbquery"), err))
			return
		}
		defer rows.Close()

		// Check if the operation was cancelled.
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len([]int{}))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		var tracks []struct {
			ID       string
			FileName string
		}
		for rows.Next() {
			var t struct {
				ID       string
				FileName string
			}
			if err := rows.Scan(&t.ID, &t.FileName); err != nil {
				m.CloseProgressDialog()
				m.ErrorHandler.ShowError(fmt.Errorf(locales.Translate("updater.err.dbscan"), err))
				return
			}
			tracks = append(tracks, t)
		}

		// Report playlist track count
		m.UpdateProgressStatus(0.5, fmt.Sprintf(locales.Translate("updater.tracks.playlistcount"), len(tracks)))
		m.ModuleBase.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.tracks.playlistcount"), len(tracks)))

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len(tracks))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Get all files in target folder
		m.UpdateProgressStatus(0.6, locales.Translate("updater.tracks.scanfolder"))
		files, err := filepath.Glob(filepath.Join(m.folderEntry.Text, "*.*"))
		if err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.ShowError(fmt.Errorf(locales.Translate("updater.err.glob"), err))
			return
		}

		// Inform about number of files in folder
		m.ModuleBase.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.tracks.countinfolder"), len(files)))

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len(tracks))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Count matching files and non-matching files
		m.UpdateProgressStatus(0.7, locales.Translate("updater.status.matching"))
		matchingFiles := 0
		nonMatchingFiles := 0
		mismatchedFiles := make([]string, 0) // Add slice for mismatched filenames
		updateTracks := make([]struct {
			TrackID     string
			NewPath     string
			NewFileName string
			NewFileType int
		}, 0)

		for _, track := range tracks {
			baseName := strings.TrimSuffix(track.FileName, filepath.Ext(track.FileName))
			newFiles, err := filepath.Glob(filepath.Join(m.folderEntry.Text, baseName+".*"))
			if err != nil || len(newFiles) == 0 {
				nonMatchingFiles++
				mismatchedFiles = append(mismatchedFiles, track.FileName) // Store mismatched filename
				continue
			}

			newPath := newFiles[0]
			newExt := strings.ToLower(filepath.Ext(newPath))
			newFileType := getFileType(newExt)
			if newFileType == 0 {
				nonMatchingFiles++
				mismatchedFiles = append(mismatchedFiles, track.FileName) // Store mismatched filename
				continue
			}

			matchingFiles++
			updateTracks = append(updateTracks, struct {
				TrackID     string
				NewPath     string
				NewFileName string
				NewFileType int
			}{
				TrackID:     track.ID,
				NewPath:     common.ToDbPath(newPath, false),
				NewFileName: filepath.Base(newPath),
				NewFileType: newFileType,
			})
		}

		// Inform about non-matching files
		if nonMatchingFiles > 0 {
			m.ModuleBase.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.tracks.badfilenamescount"), nonMatchingFiles))

			// Display list of non-matching files as warning
			fileListStr := ""
			if len(mismatchedFiles) > 5 {
				// Display only first 5 files and information about remaining files
				fileListStr = strings.Join(mismatchedFiles[:5], ", ")
				fileListStr += fmt.Sprintf(" %s", fmt.Sprintf(locales.Translate("updater.tracks.morefiles"), len(mismatchedFiles)-5))
			} else {
				fileListStr = strings.Join(mismatchedFiles, ", ")
			}
			m.ModuleBase.AddWarningMessage(fmt.Sprintf(locales.Translate("updater.tracks.badfileslist"), fileListStr))
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len(updateTracks))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Process tracks
		m.UpdateProgressStatus(0.0, locales.Translate("updater.tracks.starting"))
		for _, updateTrack := range updateTracks {
			err = m.dbMgr.Execute(`
		UPDATE djmdContent
		SET 
			FolderPath = ?,
			FileNameL = ?,
			FileType = ?
		WHERE ID = ?
	`, updateTrack.NewPath, updateTrack.NewFileName, updateTrack.NewFileType, updateTrack.TrackID)

			if err != nil {
				m.CloseProgressDialog()
				m.ErrorHandler.ShowError(fmt.Errorf(locales.Translate("common.err.dbupdate"), err))
				return
			}

			updateCount++
			progress := float64(updateCount) / float64(len(updateTracks))
			m.UpdateProgressStatus(progress, fmt.Sprintf(locales.Translate("updater.status.progress"), updateCount, len(updateTracks)))

			// Check if operation was cancelled
			if m.IsCancelled() {
				m.HandleProcessCancellation("updater.status.stopped", updateCount, len(updateTracks))
				common.UpdateButtonToCompleted(m.submitBtn)
				return
			}
		}

		// Update progress and status
		m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("updater.status.completed"), updateCount))
		m.ModuleBase.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.status.completed"), updateCount))

		// Mark the progress dialog as completed
		m.CompleteProgressDialog()

		// Update submit button to show completion using standardized function
		common.UpdateButtonToCompleted(m.submitBtn)
	}()
}
