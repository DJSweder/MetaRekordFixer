// modules/dataduplicator.go

// Package modules contains specialized functionality modules for the MetaRekordFixer application.
// Each module handles a specific task related to DJ database management and music file operations.

// This module copies these metadata fields from tracks to tracks:
// HOT CUE & Memory CUE points, play counts, addition/creation dates, color
// This is useful if the user maintains a music library in multiple formats.

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

// SourceType defines the type of source (folder or playlist) for synchronization operations.
type SourceType string

const (
	SourceTypeFolder   SourceType = common.ContentTypeFolder
	SourceTypePlaylist SourceType = common.ContentTypePlaylist
)

// DataDuplicatorModule handles hot cue synchronization between tracks.
// It allows copying hot cues and related metadata from source tracks to target tracks
// based on matching filenames, using either folder or playlist as source/target.
type DataDuplicatorModule struct {
	*common.ModuleBase
	dbMgr                *common.DBManager
	sourceType           *widget.Select
	targetType           *widget.Select
	sourceFolderField    fyne.CanvasObject
	targetFolderField    fyne.CanvasObject
	sourceFolderEntry    *widget.Entry
	targetFolderEntry    *widget.Entry
	sourcePlaylistSelect *widget.Select
	targetPlaylistSelect *widget.Select
	playlists            []common.PlaylistItem
	sourcePlaylistID     string
	targetPlaylistID     string
	submitBtn            *widget.Button
}

// NewDataDuplicatorModule creates a new DataDuplicatorModule instance and initializes its UI.
// It sets up all UI components, loads saved configuration, and prepares the module for use.
//
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for saving/loading module settings
//   - dbMgr: Database manager for accessing the DJ database
//   - errorHandler: Error handler for displaying and logging errors
//
// Returns:
//   - A fully initialized DataDuplicatorModule instance
func NewDataDuplicatorModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *DataDuplicatorModule {
	m := &DataDuplicatorModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Initialize UI components
	m.initializeUI()

	// Load configuration
	m.LoadCfg()

	return m
}

// GetName returns the localized name of this module.
// This implements the Module interface method.
func (m *DataDuplicatorModule) GetName() string {
	return locales.Translate("dataduplicator.mod.name")
}

// GetConfigName returns the configuration key for this module.
// This key is used to store and retrieve module-specific configuration.
func (m *DataDuplicatorModule) GetConfigName() string {
	return common.ModuleKeyDataDuplicator
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method and provides the visual representation
// of this module in the UI.
func (m *DataDuplicatorModule) GetIcon() fyne.Resource {
	return theme.ContentCopyIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI
// containing all the input fields, selectors, and buttons needed for hot cue synchronization.
func (m *DataDuplicatorModule) GetModuleContent() fyne.CanvasObject {
	// Form without submit button
	standardForm := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text: locales.Translate("dataduplicator.label.source"),
				Widget: container.NewBorder(
					nil, nil,
					m.sourceType,
					nil,
					container.NewStack(
						m.sourceFolderField,
						m.sourcePlaylistSelect,
					),
				),
			},
			{
				Text: locales.Translate("dataduplicator.label.target"),
				Widget: container.NewBorder(
					nil, nil,
					m.targetType,
					nil,
					container.NewStack(
						m.targetFolderField,
						m.targetPlaylistSelect,
					),
				),
			},
		},
	}

	// Create content container
	contentContainer := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("dataduplicator.label.info")),
		widget.NewSeparator(),
		standardForm,
	)

	// Add submit button with right alignment
	buttonBox := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), m.submitBtn)
	contentContainer.Add(buttonBox)

	// Update controls visibility
	m.updateControlsState()

	return contentContainer
}

// GetContent returns the module's main UI content and initializes database connection.
// This method performs database validation, loads playlists, and ensures all UI components
// are properly initialized and enabled/disabled based on the current state.
// It returns the complete module layout with status messages container.
func (m *DataDuplicatorModule) GetContent() fyne.CanvasObject {
	// Check database requirements
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Validation",
			Severity:    common.SeverityCritical,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("common.err.dbpath")), context)
		common.DisableModuleControls(m.sourcePlaylistSelect, m.targetPlaylistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}

	// Load playlists
	if err := m.loadPlaylists(); err != nil {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "LoadDataFromDatabase",
			Severity:    common.SeverityCritical,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		common.DisableModuleControls(m.sourcePlaylistSelect, m.targetPlaylistSelect, m.submitBtn)
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}

	// Enable interactive components if all checks passed
	if m.sourceType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypePlaylist)) {
		m.sourcePlaylistSelect.Enable()
	}
	if m.targetType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypePlaylist)) {
		m.targetPlaylistSelect.Enable()
	}
	m.submitBtn.Enable()

	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadCfg loads typed configuration and sets UI elements
func (m *DataDuplicatorModule) LoadCfg() {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load typed config from ConfigManager
	config, err := m.ConfigMgr.GetModuleCfg("dataduplicator", m.GetConfigName())
	if err != nil {
		// This should not happen with the updated GetModuleCfg(), but handle gracefully
		return
	}

	// Cast to DataDuplicator specific config
	if cfg, ok := config.(common.DataDuplicatorCfg); ok {
		// Update UI elements with loaded values
		m.sourceType.SetSelected(locales.Translate("dataduplicator.dropdown." + cfg.SourceType.Value))
		m.targetType.SetSelected(locales.Translate("dataduplicator.dropdown." + cfg.TargetType.Value))
		m.sourceFolderEntry.SetText(cfg.SourceFolder.Value)
		m.targetFolderEntry.SetText(cfg.TargetFolder.Value)
		m.sourcePlaylistID = cfg.SourcePlaylist.Value
		m.targetPlaylistID = cfg.TargetPlaylist.Value

		// Load playlist selections if playlists are loaded
		if len(m.playlists) > 0 {
			// Find and set source playlist
			for i, playlist := range m.playlists {
				if playlist.ID == m.sourcePlaylistID {
					if i < len(m.sourcePlaylistSelect.Options) {
						m.sourcePlaylistSelect.SetSelected(m.sourcePlaylistSelect.Options[i])
					}
					break
				}
			}

			// Find and set target playlist
			for i, playlist := range m.playlists {
				if playlist.ID == m.targetPlaylistID {
					if i < len(m.targetPlaylistSelect.Options) {
						m.targetPlaylistSelect.SetSelected(m.targetPlaylistSelect.Options[i])
					}
					break
				}
			}
		}
	}

	// Update UI state based on loaded configuration
	m.updateControlsState()
}

// SaveCfg saves current UI state to typed configuration
func (m *DataDuplicatorModule) SaveCfg() {
	if m.IsLoadingConfig {
		return // Safeguard: no save if config is being loaded
	}

	// Determine source type
	var sourceType SourceType
	if m.sourceType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
		sourceType = SourceTypeFolder
	} else {
		sourceType = SourceTypePlaylist
	}

	// Determine target type
	var targetType SourceType
	if m.targetType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
		targetType = SourceTypeFolder
	} else {
		targetType = SourceTypePlaylist
	}

	// Get playlist IDs if needed
	sourcePlaylistID := ""
	if sourceType == SourceTypePlaylist && m.sourcePlaylistSelect.Selected != "" {
		for _, playlist := range m.playlists {
			if playlist.Path == m.sourcePlaylistSelect.Selected {
				sourcePlaylistID = playlist.ID
				break
			}
		}
	}

	targetPlaylistID := ""
	if targetType == SourceTypePlaylist && m.targetPlaylistSelect.Selected != "" {
		for _, playlist := range m.playlists {
			if playlist.Path == m.targetPlaylistSelect.Selected {
				targetPlaylistID = playlist.ID
				break
			}
		}
	}

	// Get default configuration with all field definitions
	cfg := common.GetDefaultDataDuplicatorCfg()

	// Update only the values from current UI state
	cfg.SourceType.Value = string(sourceType)
	cfg.SourceFolder.Value = m.sourceFolderEntry.Text
	cfg.SourcePlaylist.Value = sourcePlaylistID
	cfg.TargetType.Value = string(targetType)
	cfg.TargetFolder.Value = m.targetFolderEntry.Text
	cfg.TargetPlaylist.Value = targetPlaylistID

	// Save typed config via ConfigManager
	m.ConfigMgr.SaveModuleCfg("dataduplicator", m.GetConfigName(), cfg)
}

// initializeUI sets up the user interface components.
// This method initializes all UI elements including selectors, entry fields,
// and buttons, and sets up their event handlers.
func (m *DataDuplicatorModule) initializeUI() {
	// Initialize source type selector
	m.sourceType = widget.NewSelect([]string{
		locales.Translate("dataduplicator.dropdown.folder"),
		locales.Translate("dataduplicator.dropdown.playlist"),
	}, nil)
	m.sourceType.OnChanged = m.CreateSelectionChangeHandler(func() {
		var sourceType SourceType
		if m.sourceType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
			sourceType = SourceTypeFolder
		} else {
			sourceType = SourceTypePlaylist
		}
		m.updateSourceVisibility(sourceType)
		m.SaveCfg()
	})

	// Initialize target type selector
	m.targetType = widget.NewSelect([]string{
		locales.Translate("dataduplicator.dropdown.folder"),
		locales.Translate("dataduplicator.dropdown.playlist"),
	}, nil)
	m.targetType.OnChanged = m.CreateSelectionChangeHandler(func() {
		var targetType SourceType
		if m.targetType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
			targetType = SourceTypeFolder
		} else {
			targetType = SourceTypePlaylist
		}
		m.updateTargetVisibility(targetType)
		m.SaveCfg()
	})

	// Initialize source folder field
	m.sourceFolderField = common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		nil,
		m.CreateChangeHandler(func() {
			m.SaveCfg()
		}),
	)
	// Extract the entry widget from the container for direct access
	if container, ok := m.sourceFolderField.(*fyne.Container); ok && len(container.Objects) > 0 {
		if entry, ok := container.Objects[0].(*widget.Entry); ok {
			m.sourceFolderEntry = entry
			m.sourceFolderEntry.TextStyle = fyne.TextStyle{Monospace: true}
		}
	}

	// Initialize target folder field
	m.targetFolderField = common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		nil,
		m.CreateChangeHandler(func() {
			m.SaveCfg()
		}),
	)
	// Extract the entry widget from the container for direct access
	if container, ok := m.targetFolderField.(*fyne.Container); ok && len(container.Objects) > 0 {
		if entry, ok := container.Objects[0].(*widget.Entry); ok {
			m.targetFolderEntry = entry
			m.targetFolderEntry.TextStyle = fyne.TextStyle{Monospace: true}
		}
	}

	// Initialize source playlist selector
	m.sourcePlaylistSelect = common.CreatePlaylistSelect(nil, "common.select.plsplacehldrinact")
	m.sourcePlaylistSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		// Find the playlist ID for the selected name
		for _, p := range m.playlists {
			if p.Path == m.sourcePlaylistSelect.Selected {
				m.sourcePlaylistID = p.ID
				break
			}
		}
		m.SaveCfg()
	})

	// Initialize target playlist selector
	m.targetPlaylistSelect = common.CreatePlaylistSelect(nil, "common.select.plsplacehldrinact")
	m.targetPlaylistSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		// Find the playlist ID for the selected name
		for _, p := range m.playlists {
			if p.Path == m.targetPlaylistSelect.Selected {
				m.targetPlaylistID = p.ID
				break
			}
		}
		m.SaveCfg()
	})

	// Create a standardized submit button
	m.submitBtn = common.CreateDisabledSubmitButton(locales.Translate("dataduplicator.button.start"), func() {
		go m.Start()
	},
	)
}

// copyHotCues copies hot cues from the source track to the target track.
// It retrieves hot cues from the source track using the database manager,
// and then applies them to the target track. The function handles both
// the retrieval and update of hot cue data.
//
// The method performs the following steps:
// 1. Retrieves all hot cues from the source track
// 2. For each hot cue, deletes any existing hot cue with the same Kind in the target track
// 3. Generates a new ID for each hot cue
// 4. Inserts the hot cue into the target track with updated timestamps
//
// Parameters:
//   - sourceID: The ID of the source track to copy hot cues from
//   - targetID: The ID of the target track to copy hot cues to
//
// Returns:
//   - error: Returns nil if successful, otherwise returns an error with a localized message
//     describing what went wrong (e.g., database query errors, update errors)
func (m *DataDuplicatorModule) copyHotCues(sourceID, targetID string) error {
	hotCues, err := m.dbMgr.GetTrackHotCues(sourceID)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("dataduplicator.err.querycues"), err)
	}

	// Counter for tracking the number of hot cues
	hotCueCount := 0

	// Process each hot cue
	for _, hotCue := range hotCues {
		// Increase the hot cue counter
		hotCueCount++

		// Get the Kind value from the hot cue
		kind, ok := hotCue["Kind"]
		if !ok {
			continue
		}

		// Delete existing hot cues with the same Kind value in the target track
		err = m.dbMgr.Execute(`DELETE FROM djmdCue WHERE ContentID = ? AND Kind = ?`, targetID, kind)
		if err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("dataduplicator.err.deletecue"), err)
		}

		// Generate a new ID for the hot cue in the target track
		var maxID int64
		err = m.dbMgr.QueryRow("SELECT COALESCE(MAX(CAST(ID AS INTEGER)), 0) FROM djmdCue").Scan(&maxID)
		if err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("dataduplicator.err.maxidcheck"), err)
		}
		maxID++
		newID := fmt.Sprintf("%d", maxID)

		// Get current timestamp for created_at
		currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000 +00:00")

		// SQL query preparation for inserting hot cue
		query := `
			INSERT INTO djmdCue (
				ID, ContentID, InMsec, InFrame, InMpegFrame, InMpegAbs, OutMsec, OutFrame, OutMpegFrame, 
				OutMpegAbs, Kind, Color, ColorTableIndex, ActiveLoop, Comment, BeatLoopSize, CueMicrosec, 
				InPointSeekInfo, OutPointSeekInfo, ContentUUID, UUID, rb_data_status, rb_local_data_status, 
				rb_local_deleted, rb_local_synced, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, 
				?, ?, ?, ?, ?, ?, ?, ?, 
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			)
		`

		// Parameters for the insert preparation
		params := []interface{}{
			newID, targetID,
			hotCue["InMsec"], hotCue["InFrame"], hotCue["InMpegFrame"], hotCue["InMpegAbs"],
			hotCue["OutMsec"], hotCue["OutFrame"], hotCue["OutMpegFrame"], hotCue["OutMpegAbs"],
			hotCue["Kind"], hotCue["Color"], hotCue["ColorTableIndex"], hotCue["ActiveLoop"],
			hotCue["Comment"], hotCue["BeatLoopSize"], hotCue["CueMicrosec"],
			hotCue["InPointSeekInfo"], hotCue["OutPointSeekInfo"], hotCue["ContentUUID"],
			hotCue["UUID"], hotCue["rb_data_status"], hotCue["rb_local_data_status"],
			hotCue["rb_local_deleted"], hotCue["rb_local_synced"],
			currentTime, currentTime,
		}

		// Execute the insert
		err = m.dbMgr.Execute(query, params...)
		if err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("dataduplicator.err.cueinsert"), err)
		}
	}

	m.Logger.Info(locales.Translate("dataduplicator.status.copiedcues"), hotCueCount, sourceID, targetID)
	return nil
}

// copyTrackMetadata copies specific metadata fields from source track to target track.
// Fields copied: StockDate, DateCreated, ColorID, DJPlayCount
//
// Parameters:
//   - sourceID: The ID of the source track to copy metadata from
//   - targetID: The ID of the target track to copy metadata to
//
// Returns:
//   - error: Returns nil if successful, otherwise returns an error with details about the failure
func (m *DataDuplicatorModule) copyTrackMetadata(sourceID, targetID string) error {
	// Query to get source track metadata
	query := `
		SELECT StockDate, DateCreated, ColorID, DJPlayCount
		FROM djmdContent
		WHERE ID = ?
	`

	row := m.dbMgr.QueryRow(query, sourceID)
	if row == nil {
		return fmt.Errorf("%s", locales.Translate("dataduplicator.err.querysource"))
	}

	// Scan source track metadata using our custom null types
	var stockDate common.NullString
	var dateCreated common.NullString
	var colorID common.NullInt64
	var djPlayCount common.NullInt64

	err := row.Scan(&stockDate, &dateCreated, &colorID, &djPlayCount)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("dataduplicator.err.metadatascan"), err)
	}

	// Get current timestamp for updated_at
	currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000 +00:00")

	// Update target track with source track metadata
	updateQuery := `
		UPDATE djmdContent
		SET StockDate = ?, DateCreated = ?, ColorID = ?, DJPlayCount = ?, updated_at = ?
		WHERE ID = ?
	`

	err = m.dbMgr.Execute(updateQuery,
		stockDate.ValueOrNil(),
		dateCreated.ValueOrNil(),
		colorID.ValueOrNil(),
		djPlayCount.ValueOrNil(),
		currentTime, targetID)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("dataduplicator.err.metadataupdate"), err)
	}

	m.Logger.Info(locales.Translate("dataduplicator.status.copiedmetadata"), sourceID, targetID)
	return nil
}

// getSourceTracks retrieves source tracks from the database based on the selected source type.
// It handles both folder-based and playlist-based track retrieval.
//
// Returns:
//   - []common.TrackItem: A slice of tracks retrieved from the selected source
//   - error: An error if no tracks were found or if another issue occurred
func (m *DataDuplicatorModule) getSourceTracks() ([]common.TrackItem, error) {
	var tracks []common.TrackItem

	if m.sourceType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
		tracks, _ = m.dbMgr.GetTracksBasedOnFolder(m.sourceFolderEntry.Text)
	} else {
		// Find playlist ID
		var playlistID string

		// First try to find the playlist by path
		for _, p := range m.playlists {
			if p.Path == m.sourcePlaylistSelect.Selected {
				playlistID = p.ID
				break
			}
		}

		tracks, _ = m.dbMgr.GetTracksBasedOnPlaylist(playlistID)
	}

	if len(tracks) == 0 {
		return nil, fmt.Errorf("%s", locales.Translate("dataduplicator.err.nosourcetracks"))
	}

	return tracks, nil
}

// getTargetTracks retrieves target tracks from the database based on the selected target type.
// It finds tracks in the target location (folder or playlist) that match the source track's filename
// (without extension), allowing for synchronization between different formats of the same track.
//
// Parameters:
//   - sourceTrack: The source track to find matches for
//
// Returns:
//   - A slice of matching target tracks with their IDs and filenames
//   - error: An error if retrieval failed
func (m *DataDuplicatorModule) getTargetTracks(sourceTrack common.TrackItem) ([]struct {
	ID       string
	FileName string
}, error) {
	// Extract the file name from the source track's folder path without extension
	fileName := filepath.Base(sourceTrack.FolderPath)
	relativePathWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	var targetTracks []common.TrackItem

	if m.targetType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
		targetTracks, _ = m.dbMgr.GetTracksBasedOnFolder(m.targetFolderEntry.Text)
	} else {
		// Find playlist ID
		var playlistID string

		// First try to find the playlist by path
		for _, p := range m.playlists {
			if p.Path == m.targetPlaylistSelect.Selected {
				playlistID = p.ID
				break
			}
		}

		targetTracks, _ = m.dbMgr.GetTracksBasedOnPlaylist(playlistID)
	}

	// Prepare final result slice
	var result []struct {
		ID       string
		FileName string
	}

	// Omit the source track from the destination
	for _, track := range targetTracks {
		if track.ID == sourceTrack.ID {
			continue
		}

		// Get the relative path of the target file without the extension
		targetFileName := filepath.Base(track.FolderPath)
		targetRelativePathWithoutExt := strings.TrimSuffix(targetFileName, filepath.Ext(targetFileName))

		// Compare relative paths (without extension) using case-sensitive comparison
		if targetRelativePathWithoutExt == relativePathWithoutExt {
			result = append(result, struct {
				ID       string
				FileName string
			}{
				ID:       track.ID,
				FileName: track.FileNameL,
			})
		}
	}

	if len(result) == 0 {
		m.Logger.Warning(locales.Translate("dataduplicator.err.notgttracks"), fileName)
	} else {
		m.Logger.Info(locales.Translate("dataduplicator.status.foundtargettracks"), fileName, len(result))
	}

	return result, nil
}

// loadPlaylists loads playlist items from the database and updates the playlist selectors.
// It updates the UI to show loading state, retrieves playlists from the database,
// and populates the source and target playlist selectors with the results.
//
// Returns:
//   - error: An error if playlist loading failed
func (m *DataDuplicatorModule) loadPlaylists() error {
	// Update UI to show loading state
	m.StartProcessing(locales.Translate("common.status.playlistload"))

	// Get playlists from database
	playlists, err := m.dbMgr.GetPlaylists()
	if err != nil {
		return err
	}

	// Store playlists for later use
	m.playlists = playlists

	// Create options list for selectors
	options := make([]string, len(playlists))
	for i, playlist := range playlists {
		options[i] = playlist.Path // Use Path instead of Name to show hierarchy
	}

	// Update selectors
	m.sourcePlaylistSelect.Options = options
	m.targetPlaylistSelect.Options = options

	// Prepare selected values for source and target
	var sourceSelectedValue string
	var targetSelectedValue string

	// Find source selected value
	if m.sourcePlaylistID != "" {
		for _, playlist := range m.playlists {
			if playlist.ID == m.sourcePlaylistID {
				sourceSelectedValue = playlist.Path
				break
			}
		}
	}

	// Find target selected value
	if m.targetPlaylistID != "" {
		for _, playlist := range m.playlists {
			if playlist.ID == m.targetPlaylistID {
				targetSelectedValue = playlist.Path
				break
			}
		}
	}

	// Set active state for source and target playlist selects
	common.SetPlaylistSelectState(m.sourcePlaylistSelect, true, sourceSelectedValue)
	common.SetPlaylistSelectState(m.targetPlaylistSelect, true, targetSelectedValue)

	m.Logger.Info(locales.Translate("dataduplicator.status.loadedplaylists"), len(playlists))
	return nil
}

// updateControlsState updates the visibility of UI controls based on the current source and target types.
// It ensures that only the relevant input fields are shown based on whether folder or playlist
// is selected as the source and target.
func (m *DataDuplicatorModule) updateControlsState() {
	// Get current source and target types
	var sourceType, targetType SourceType
	if m.sourceType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
		sourceType = SourceTypeFolder
	} else {
		sourceType = SourceTypePlaylist
	}

	if m.targetType.Selected == locales.Translate("dataduplicator.dropdown."+string(SourceTypeFolder)) {
		targetType = SourceTypeFolder
	} else {
		targetType = SourceTypePlaylist
	}

	// Update visibility based on selected source type
	if sourceType == SourceTypeFolder {
		m.sourceFolderField.Show()
		m.sourcePlaylistSelect.Hide()
	} else {
		m.sourceFolderField.Hide()
		m.sourcePlaylistSelect.Show()
	}

	// Update visibility based on selected target type
	if targetType == SourceTypeFolder {
		m.targetFolderField.Show()
		m.targetPlaylistSelect.Hide()
	} else {
		m.targetFolderField.Hide()
		m.targetPlaylistSelect.Show()
	}
}

// updateSourceVisibility updates the visibility of source input controls based on the selected source type.
// When switching from folder to playlist, it also reloads playlists from the database.
//
// Parameters:
//   - sourceType: The selected source type (folder or playlist)
func (m *DataDuplicatorModule) updateSourceVisibility(sourceType SourceType) {
	if sourceType == SourceTypeFolder {
		m.sourceFolderField.Show()
		m.sourcePlaylistSelect.Hide()
	} else {
		// Switch from type folder to playlist will load playlists again
		if err := m.dbMgr.Connect(); err == nil {
			if err := m.loadPlaylists(); err != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Load Playlists",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				}
				m.ErrorHandler.ShowStandardError(err, context)
			}
			m.dbMgr.Finalize()
		}
		m.sourceFolderField.Hide()
		m.sourcePlaylistSelect.Show()
	}
}

// updateTargetVisibility updates the visibility of target input controls based on the selected target type.
// When switching from folder to playlist, it also reloads playlists from the database.
//
// Parameters:
//   - targetType: The selected target type (folder or playlist)
func (m *DataDuplicatorModule) updateTargetVisibility(targetType SourceType) {
	if targetType == SourceTypeFolder {
		m.targetFolderField.Show()
		m.targetPlaylistSelect.Hide()
	} else {
		// Switch from type folder to playlist will load playlists again
		if err := m.dbMgr.Connect(); err == nil {
			if err := m.loadPlaylists(); err != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Load Playlists",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				}
				m.ErrorHandler.ShowStandardError(err, context)
			}
			m.dbMgr.Finalize()
		}
		m.targetFolderField.Hide()
		m.targetPlaylistSelect.Show()
	}
}

// Start performs the necessary steps before starting the main process.
// It saves the configuration, validates the inputs, informs the user, displays a dialog with a progress bar
// and starts the main process.
// Input validation also includes a test of the connection to the database and creating a backup of it.
// This method is called when the user clicks the submit button.
func (m *DataDuplicatorModule) Start() {

	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate("start"); err != nil {
		return
	}

	// Show progress dialog
	m.ShowProgressDialog(locales.Translate("dataduplicator.dialog.header"))

	// Start processing in goroutine
	go m.processUpdate()

}

// processUpdate performs the actual hot cue synchronization process.
// This method runs in a goroutine and handles the entire synchronization workflow:
// 1. Gets source tracks based on selected source type
// 2. For each source track, finds matching target tracks
// 3. Copies hot cues and metadata from source to target tracks
// 4. Updates progress and handles cancellation throughout the process
// 5. Shows completion status when finished
//
// The method includes panic recovery to ensure the progress dialog is always closed
// even if an unexpected error occurs.
func (m *DataDuplicatorModule) processUpdate() {
	defer func() {
		if r := recover(); r != nil {
			m.CloseProgressDialog()
			context := &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Panic Recovery",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %v", locales.Translate("dataduplicator.err.panic"), r), context)
		}
	}()

	// Get source tracks
	sourceTracks, err := m.getSourceTracks()
	if err != nil {
		m.CloseProgressDialog()
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Get Source Tracks",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("dataduplicator.err.nosourcetracks")), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Check if operation was cancelled
	if m.IsCancelled() {
		m.HandleProcessCancellation("common.status.stopped", 0, len(sourceTracks))
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Update progress
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("dataduplicator.status.srctrackscount"), len(sourceTracks)))

	// Track successful and skipped files
	processedCount := 0
	skippedCount := 0

	// Update progress before processing
	m.AddInfoMessage(locales.Translate("common.status.updating"))

	// Process each source track
	for _, sourceTrack := range sourceTracks {
		// Check if operation was cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("common.status.stopped", processedCount, len(sourceTracks))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Get target tracks
		targetTracks, err := m.getTargetTracks(sourceTrack)
		if err != nil {
			context := &common.ErrorContext{
				Module:      m.GetConfigName(),
				Operation:   "Get Target Tracks",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(err, context)
			m.CloseProgressDialog()
			return
		}

		// Skip if no target tracks found
		if len(targetTracks) == 0 {
			skippedCount++
			continue
		}

		// Update progress
		m.UpdateProcessingProgress(processedCount, len(sourceTracks), fmt.Sprintf("%s: %d/%d", locales.Translate("dataduplicator.diagstatus.process"), processedCount+1, len(sourceTracks)))

		// Process target tracks
		for _, targetTrack := range targetTracks {
			// Check if operation was cancelled
			if m.IsCancelled() {
				m.HandleProcessCancellation("common.status.stopped", processedCount, len(sourceTracks))
				common.UpdateButtonToCompleted(m.submitBtn)
				return
			}

			// Copy hot cues
			err = m.copyHotCues(sourceTrack.ID, targetTrack.ID)
			if err != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Copy Hot Cues",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(err, context)
				m.CloseProgressDialog()
				m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
				return
			}

			// Copy track metadata
			err = m.copyTrackMetadata(sourceTrack.ID, targetTrack.ID)
			if err != nil {
				context := &common.ErrorContext{
					Module:      m.GetConfigName(),
					Operation:   "Copy Track Metadata",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(err, context)
				m.CloseProgressDialog()
				m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
				return
			}
			processedCount++

			// Small delay to prevent database overload
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Update progress and status
	m.CompleteProcessing(fmt.Sprintf(locales.Translate("dataduplicator.status.completed"), processedCount, skippedCount))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("dataduplicator.status.completed"), processedCount, skippedCount))

	// Complete progress dialog and update button
	m.CompleteProgressDialog()

	// Update submit button to show completion
	common.UpdateButtonToCompleted(m.submitBtn)
}
