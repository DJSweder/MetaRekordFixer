// modules/formatupdater.go

// Package modules provides functionality for different modules in the MetaRekordFixer application.
// Each module handles a specific task related to DJ database management and music file operations.

// This module is used for changing the format of music files (e.g. replacing MP3 with FLAC) while maintaining all original track information.
// To identify these tracks, it is necessary to prepare them in advance in a playlist.

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

// FormatUpdaterModule is a module that handles updating track format in database.
// It allows users to select a playlist and a folder with new audio files, then updates
// the file paths and formats in the database to match the new files.
type FormatUpdaterModule struct {
	// ModuleBase provides common module functionality like error handling and UI components
	*common.ModuleBase
	dbMgr                *common.DBManager
	playlistSelect       *widget.Select
	folderEntry          *widget.Entry
	folderSelectionField fyne.CanvasObject
	submitBtn            *widget.Button
	playlists            []common.PlaylistItem
	pendingPlaylistID    string // Temporary storage for playlist ID
}

// NewFormatUpdaterModule creates a new instance of FormatUpdaterModule.
// It initializes the module with the provided window, configuration manager, database manager,
// and error handler, sets up the UI components, and loads any saved configuration.
//
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for saving/loading module settings
//   - dbMgr: Database manager for accessing the DJ database
//   - errorHandler: Error handler for displaying and logging errors
//
// Returns:
//   - A fully initialized FormatUpdaterModule instance
func NewFormatUpdaterModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *FormatUpdaterModule {
	m := &FormatUpdaterModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadCfg()

	return m
}

// GetName returns the localized name of this module.
// This implements the Module interface method.
func (m *FormatUpdaterModule) GetName() string {
	return locales.Translate("formatupdater.mod.name")
}

// GetConfigName returns the module's configuration key.
// This key is used to store and retrieve module-specific configuration.
func (m *FormatUpdaterModule) GetConfigName() string {
	return common.ModuleKeyFormatUpdater
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method and provides the visual representation
// of this module in the UI.
func (m *FormatUpdaterModule) GetIcon() fyne.Resource {
	return theme.SearchReplaceIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI
// containing the playlist selector, folder selection field, and submit button.
func (m *FormatUpdaterModule) GetModuleContent() fyne.CanvasObject {
	// Create form with playlist selector and folder selection field
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("formatupdater.label.replaced"), Widget: m.playlistSelect},
			{Text: locales.Translate("formatupdater.label.newfiles"), Widget: m.folderSelectionField},
		},
	}

	// Create content container with form
	contentContainer := container.NewVBox(
		form,
	)

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("formatupdater.label.info")),
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
// It checks database requirements, loads playlists, and creates the complete module layout
// with status messages container. If database checks fail, it disables the module controls.
func (m *FormatUpdaterModule) GetContent() fyne.CanvasObject {
	// Check database requirements
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "PathToDatabaseCheck",
			Severity:    common.SeverityWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("common.err.dbpath")), context)
		common.DisableModuleControls(m.playlistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}

	// Load playlists
	if err := m.loadPlaylists(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "LoadDataFromDatabase",
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

// LoadCfg loads typed configuration and updates UI elements
func (m *FormatUpdaterModule) LoadCfg() {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load typed config from ConfigManager
	config, err := m.ConfigMgr.GetModuleCfg(common.ModuleKeyFormatUpdater, m.GetConfigName())
	if err != nil {
		// This should not happen with the updated GetModuleCfg(), but handle gracefully
		return
	}

	// Cast to FormatUpdater specific config
	if cfg, ok := config.(common.FormatUpdaterCfg); ok {
		// Update UI elements with loaded values
		m.folderEntry.SetText(cfg.Folder.Value)
		m.pendingPlaylistID = cfg.PlaylistID.Value

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
}

// SaveCfg saves current UI state to typed configuration
func (m *FormatUpdaterModule) SaveCfg() {
	if m.IsLoadingConfig {
		return // Safeguard: no save if config is being loaded
	}

	// Get default configuration with all field definitions
	cfg := common.GetDefaultFormatUpdaterCfg()

	// Update only the values from current UI state
	cfg.Folder.Value = m.folderEntry.Text
	cfg.PlaylistID.Value = m.pendingPlaylistID

	// Save typed config via ConfigManager
	m.ConfigMgr.SaveModuleCfg(common.ModuleKeyFormatUpdater, m.GetConfigName(), cfg)
}

// initializeUI sets up the user interface components.
// It creates and configures all UI elements including the playlist selector,
// folder selection field, and submit button, and sets up their event handlers.
func (m *FormatUpdaterModule) initializeUI() {
	// Initialize folder selection field first, then extract entry for event handling

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
		m.SaveCfg()
	}), "common.select.plsplacehldrinact")

	// Create a folder selection field using the standardized function.
	// The folder selection field consists of a button and a text entry.
	// When the user clicks the button, open a file dialog for the user to choose a folder.
	// When the user chooses a folder, set the text entry to the path of the chosen folder
	// and save the config.
	m.folderSelectionField = common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		nil,
		func(path string) {
			m.SaveCfg()
		},
	)
	// Extract the entry widget from the container for direct access
	if container, ok := m.folderSelectionField.(*fyne.Container); ok && len(container.Objects) > 0 {
		if entry, ok := container.Objects[0].(*widget.Entry); ok {
			m.folderEntry = entry
		}
	}

	// Create a disabled submit button using the standardized function.
	// The submit button is disabled to prevent the user from starting the module
	// before the module is fully loaded.
	// When the user clicks the submit button, start the module.
	m.submitBtn = common.CreateDisabledSubmitButton(locales.Translate("formatupdater.button.libupd"), func() {
		go m.Start()
	},
	)
}

// getFileType translates a file extension into a numeric identifier used in the database.
// This identifier is stored in the FileType field of the djmdContent table.
//
// Parameters:
//   - ext: The file extension including the dot (e.g., ".mp3")
//
// Returns:
//   - An integer representing the file type in the database format
func getFileType(ext string) int {
	switch strings.ToLower(ext) {
	case common.ExtensionMP3:
		return 1
	case common.ExtensionM4A:
		return 4
	case common.ExtensionFLAC:
		return 5
	case common.ExtensionWAV:
		return 11
	case common.ExtensionAIFF:
		return 12
	default:
		return 0
	}
}

// loadPlaylists loads playlist items from the database and updates the playlist selector.
// It connects to the database, retrieves all playlists, and updates the UI component
// with the playlist paths. It also restores any previously selected playlist.
//
// Returns:
//   - An error if database connection or playlist retrieval fails, nil otherwise
func (m *FormatUpdaterModule) loadPlaylists() error {
	err := m.dbMgr.Connect()
	if err != nil {
		return err // DBMgr.Connect() is expected to return a localized error.
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

// Start performs the necessary steps before starting the main process.
// It validates the inputs, displays a progress dialog, and starts the update process.
// Input validation includes checking the database connection and creating a backup.
//
// The actual update process runs in a separate goroutine to keep the UI responsive.
func (m *FormatUpdaterModule) Start() {

	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate("start"); err != nil {
		return
	}

	// Show the progress dialog
	m.ShowProgressDialog(locales.Translate("formatupdater.dialog.header"))

	// Start processing in a goroutine
	go m.processUpdate()
}

// processUpdate performs the actual track update process.
// It retrieves tracks from the selected playlist, finds matching files in the target folder,
// and updates the file paths and formats in the database.
//
// The process includes:
// 1. Validating the playlist selection
// 2. Loading tracks from the selected playlist
// 3. Scanning the target folder for matching files
// 4. Matching files by base name (without extension)
// 5. Updating track records in the database
// 6. Reporting progress and results
//
// The process can be cancelled at any time by the user.
func (m *FormatUpdaterModule) processUpdate() {
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
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("formatupdater.err.noplaylist")), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}
	defer func() {
		// Catch any panics or errors and show an error message.
		if r := recover(); r != nil {
			m.CloseProgressDialog()
			context := &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "UpdateProcess",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
			m.AddErrorMessage(locales.Translate("common.err.statusfinal"))

		}
	}()

	// Check if the operation was cancelled.
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, 0)
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Get the selected playlist.
	m.StartProcessing(locales.Translate("common.status.playlistload"))
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
			Operation:   "PlaylistSelection",
			Severity:    common.SeverityWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("formatupdater.err.noplaylist")), context)
		m.CloseProgressDialog()
		return
	}

	// Get the tracks from the playlist.
	rows, err := m.dbMgr.Query(`
		SELECT c.ID, c.FileNameL
		FROM djmdContent c
		JOIN djmdSongPlaylist sp ON c.ID = sp.ContentID
		WHERE sp.PlaylistID = ?
	`, selectedPlaylist)
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "GetPlaylistTracks",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context) // This error is not wrapped, because DBMgr provides localized message for error dialog.
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
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
				Operation:   "DatabaseScan",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(err, context) // This error is not wrapped, because DBMgr provides localized message for error dialog.
			m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
			m.CloseProgressDialog()
			return
		}
		tracks = append(tracks, t)
	}

	// Report playlist track count
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatupdater.tracks.playlistcount"), len(tracks)))

	// Check if operation was cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, 0)
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Get all files in target folder
	files, err := common.ListFilesWithExtensions(m.folderEntry.Text, nil, false)
	if err != nil {
		m.CloseProgressDialog()
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "ScanFolder",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.err.noreadaccess"), err), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Inform about number of files in folder
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatupdater.tracks.countinfolder"), len(files)))

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
		m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatupdater.tracks.badfilenamescount"), nonMatchingFiles))

		// Display list of non-matching files as warning
		fileListStr := ""
		if len(mismatchedFiles) > 5 {
			fileListStr = fmt.Sprintf("%s %s",
				strings.Join(mismatchedFiles[:5], ", "),
				fmt.Sprintf(locales.Translate("formatupdater.tracks.morefiles"), len(mismatchedFiles)-5))
		} else {
			fileListStr = strings.Join(mismatchedFiles, ", ")
		}
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatupdater.tracks.badfileslist"), fileListStr))
	}

	// Check if operation was cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("updater.status.stopped", updateCount, len(updateTracks))
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Update tracks in database
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
			m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.err.dbupdate"), err), context)
			m.CloseProgressDialog()
			return
		}

		updateCount++
		m.UpdateProcessingProgress(updateCount-1, len(updateTracks), fmt.Sprintf(locales.Translate("formatupdater.status.progress"), updateCount, len(updateTracks)))

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("updater.status.stopped", updateCount, len(updateTracks))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}
	}

	// Update progress and status
	m.CompleteProcessing(fmt.Sprintf(locales.Translate("formatupdater.status.completed"), updateCount))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatupdater.status.completed"), updateCount))

	// Mark the progress dialog as completed
	m.CompleteProgressDialog()

	// Update submit button to show completion
	common.UpdateButtonToCompleted(m.submitBtn)
}
