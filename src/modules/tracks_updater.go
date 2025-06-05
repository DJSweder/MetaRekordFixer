package modules

import (
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TracksUpdaterModule is a module that handles updating track format in database.
type TracksUpdaterModule struct {
	// ModuleBase is the base struct for all modules, which contains the module's window, error handler, and configuration manager.
	*common.ModuleBase
	dbMgr             *common.DBManager
	playlistSelect    *widget.Select
	folderEntry       *widget.Entry
	folderSelect      *widget.Button
	submitBtn         *widget.Button
	playlists         []common.PlaylistItem
	pendingPlaylistID string // Temporary storage for playlist ID
}

// NewTracksUpdater creates a new instance of TracksUpdaterModule.
func NewTracksUpdaterModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *TracksUpdaterModule {
	m := &TracksUpdaterModule{
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
func (m *TracksUpdaterModule) GetName() string {
	return locales.Translate("updater.mod.name")
}

// GetConfigName returns the module's configuration key.
func (m *TracksUpdaterModule) GetConfigName() string {
	return "updater"
}

// GetIcon returns the module's icon resource.
func (m *TracksUpdaterModule) GetIcon() fyne.Resource {
	return theme.SearchReplaceIcon()
}

// GetModuleContent returns the module's specific content without status messages
// This implements the method from ModuleBase to provide the module-specific UI
func (m *TracksUpdaterModule) GetModuleContent() fyne.CanvasObject {
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
func (m *TracksUpdaterModule) GetContent() fyne.CanvasObject {
	// Check database requirements
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Validation",
			Severity:    common.SeverityWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("common.err.nodbpath")), context)
		common.DisableModuleControls(m.playlistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}

	// Load playlists
	if err := m.loadPlaylists(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Access",
			Severity:    common.SeverityWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
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
func (m *TracksUpdaterModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Check if Fields are not initialized
	if cfg.Fields == nil {
		cfg = common.NewModuleConfig()

		// Set default values with their definitions
		cfg.SetWithDefinitionAndActions("folder", "", "folder", true, "exists", []string{"start"})
		cfg.SetWithDefinitionAndActions("playlist_id", "", "playlist", true, "filled", []string{"start"})

		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	}

	// Load folder path
	m.folderEntry.SetText(cfg.Get("folder", ""))

	// Load playlist ID
	m.pendingPlaylistID = cfg.Get("playlist_id", "")

	// Load playlist selection if playlists are already loaded
	if m.pendingPlaylistID != "" && len(m.playlists) > 0 {
		for _, playlist := range m.playlists {
			if playlist.ID == m.pendingPlaylistID {
				m.playlistSelect.SetSelected(playlist.Path)
				break
			}
		}
	}
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
func (m *TracksUpdaterModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig()
	}

	cfg := common.NewModuleConfig()

	// Save folder path
	cfg.SetWithDefinitionAndActions("folder", m.folderEntry.Text, "folder", true, "exists", []string{"start"})

	// Save playlist ID
	if m.pendingPlaylistID != "" {
		cfg.SetWithDefinitionAndActions("playlist_id", m.pendingPlaylistID, "playlist", true, "filled", []string{"start"})
	}

	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components.
func (m *TracksUpdaterModule) initializeUI() {
	// Create a text entry for the user to input the folder path.
	// When the user changes the text in the entry, save the config.
	m.folderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Create a disabled select widget for the user to choose a playlist.
	// When the user chooses a playlist, save the config.
	// The select widget is disabled to prevent the user from changing the playlist
	// before the module is fully loaded.
	m.playlistSelect = common.CreatePlaylistSelect(m.CreateSelectionChangeHandler(func() {
		// Find the playlist ID for the selected name
		for _, p := range m.playlists {
			if p.Path == m.playlistSelect.Selected {
				m.pendingPlaylistID = p.ID
				break
			}
		}
		m.SaveConfig()
	}), "common.select.plsplacehldrinact")

	// Create a folder selection field using the standardized function.
	// The folder selection field consists of a button and a text entry.
	// When the user clicks the button, open a file dialog for the user to choose a folder.
	// When the user chooses a folder, set the text entry to the path of the chosen folder
	// and save the config.
	folderSelectionField := common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
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
	m.submitBtn = common.CreateDisabledSubmitButton(locales.Translate("updater.button.libupd"), func() {
		go m.Start()
	},
	)
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

func (m *TracksUpdaterModule) loadPlaylists() error {
	err := m.dbMgr.Connect()
	if err != nil {
		return fmt.Errorf("%s %w", locales.Translate("common.err.dbconnect"), err)
	}
	defer m.dbMgr.Finalize()

	// Get playlists from database
	playlists, err := m.dbMgr.GetPlaylists()
	if err != nil {
		return err
	}

	// Store playlists for later use
	m.playlists = playlists

	// Create options list for selectors
	playlistPaths := make([]string, len(playlists))
	for i, playlist := range playlists {
		playlistPaths[i] = playlist.Path
	}

	// Update select widget options
	m.playlistSelect.Options = playlistPaths

	// Set active state with appropriate placeholder
	var selectedValue string

	// Find selected value from pending ID if exists
	if m.pendingPlaylistID != "" {
		for _, playlist := range m.playlists {
			if playlist.ID == m.pendingPlaylistID {
				selectedValue = playlist.Path
				break
			}
		}
	}

	// Set active state with found value (or empty if no pending ID)
	common.SetPlaylistSelectState(m.playlistSelect, true, selectedValue)

	return nil
}

// Start performs the necessary steps before starting the main process
// It saves the configuration, validates the inputs, informs the user, displays a dialog with a progress bar
// and starts the main process.
// Input validation also includes a test of the connection to the database and creating a backup of it.
func (m *TracksUpdaterModule) Start() {

	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate("start"); err != nil {
		return
	}

	// Show the progress dialog
	m.ShowProgressDialog(locales.Translate("updater.dialog.header"))

	// Start processing in a goroutine
	go m.processUpdate()
}

func (m *TracksUpdaterModule) processUpdate() {
	// Track the number of updated files.
	updateCount := 0
	// Validate playlist selection
	if m.playlistSelect.Selected == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Playlist selection validation",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("updater.err.noplaylist")), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}
	defer func() {
		// Catch any panics or errors and show an error message.
		if r := recover(); r != nil {
			m.CloseProgressDialog()
			context := &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Update Process",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
			m.AddErrorMessage(locales.Translate("common.status.failed"))

		}
	}()

	// Check if the operation was cancelled.
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, 0)
		common.UpdateButtonToCompleted(m.submitBtn)
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
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Playlist Selection",
			Severity:    common.SeverityWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("updater.err.noplaylist")), context)
		m.CloseProgressDialog()
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
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Get Playlist Tracks",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.dbquery"), err), context)
		m.CloseProgressDialog()
		return
	}
	defer rows.Close()

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
			context := &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Database Scan",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("updater.err.dbscan"), err), context)
			m.CloseProgressDialog()
			return
		}
		tracks = append(tracks, t)
	}

	// Report playlist track count
	m.UpdateProgressStatus(0.5, fmt.Sprintf(locales.Translate("updater.tracks.playlistcount"), len(tracks)))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.tracks.playlistcount"), len(tracks)))

	// Check if operation was cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, 0)
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Get all files in target folder
	m.UpdateProgressStatus(0.6, locales.Translate("updater.tracks.scanfolder"))
	files, err := filepath.Glob(filepath.Join(m.folderEntry.Text, "*.*"))
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Scan Folder",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("updater.err.glob"), err), context)
		m.CloseProgressDialog()
		return
	}

	// Inform about number of files in folder
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.tracks.countinfolder"), len(files)))

	// Check if operation was cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, 0)
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Process file matching and updates
	matchingFiles := 0
	nonMatchingFiles := 0
	mismatchedFiles := make([]string, 0)
	updateTracks := make([]struct {
		TrackID     string
		NewPath     string
		NewFileName string
		NewFileType int
	}, 0)

	// Match files and prepare updates
	m.UpdateProgressStatus(0.7, locales.Translate("updater.status.matching"))
	for _, track := range tracks {
		baseName := strings.TrimSuffix(track.FileName, filepath.Ext(track.FileName))
		newFiles, err := filepath.Glob(filepath.Join(m.folderEntry.Text, baseName+".*"))
		if err != nil || len(newFiles) == 0 {
			nonMatchingFiles++
			mismatchedFiles = append(mismatchedFiles, track.FileName)
			continue
		}

		newPath := newFiles[0]
		newExt := strings.ToLower(filepath.Ext(newPath))
		newFileType := getFileType(newExt)
		if newFileType == 0 {
			nonMatchingFiles++
			mismatchedFiles = append(mismatchedFiles, track.FileName)
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

	// Report non-matching files
	if nonMatchingFiles > 0 {
		m.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.tracks.badfilenamescount"), nonMatchingFiles))

		// Display list of non-matching files as warning
		fileListStr := ""
		if len(mismatchedFiles) > 5 {
			fileListStr = fmt.Sprintf("%s %s",
				strings.Join(mismatchedFiles[:5], ", "),
				fmt.Sprintf(locales.Translate("updater.tracks.morefiles"), len(mismatchedFiles)-5))
		} else {
			fileListStr = strings.Join(mismatchedFiles, ", ")
		}
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("updater.tracks.badfileslist"), fileListStr))
	}

	// Check if operation was cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, len(updateTracks))
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Update tracks in database
	m.UpdateProgressStatus(0.8, locales.Translate("updater.tracks.starting"))
	for _, updateTrack := range updateTracks {
		if err := m.dbMgr.Execute(`
			UPDATE djmdContent
			SET 
				FolderPath = ?,
				FileNameL = ?,
				FileType = ?
			WHERE ID = ?
		`, updateTrack.NewPath, updateTrack.NewFileName, updateTrack.NewFileType, updateTrack.TrackID); err != nil {
			context := &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Update Track",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.dbupdate"), err), context)
			m.CloseProgressDialog()
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
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("updater.status.completed"), updateCount))

	// Mark the progress dialog as completed
	m.CompleteProgressDialog()

	// Update submit button to show completion
	common.UpdateButtonToCompleted(m.submitBtn)
}
