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

// SourceType defines the type of source (folder or playlist).
type SourceType string

const (
	SourceTypeFolder   SourceType = "folder"
	SourceTypePlaylist SourceType = "playlist"
)

// HotCueSyncModule handles hot cue synchronization.
type HotCueSyncModule struct {
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

// NewHotCueSyncModule creates a new HotCueSyncModule instance and initializes its UI.
func NewHotCueSyncModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *HotCueSyncModule {
	m := &HotCueSyncModule{
		ModuleBase:     common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:          dbMgr,
		playlists:      make([]common.PlaylistItem, 0),
	}
	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	// Update controls state without loading playlists
	m.updateControlsState()

	return m
}

// GetName returns the localized name of this module.
func (m *HotCueSyncModule) GetName() string {
	return locales.Translate("hotcuesync.mod.name")
}

// GetConfigName returns the configuration key for this module.
func (m *HotCueSyncModule) GetConfigName() string {
	return "hotcue_sync"
}

// GetIcon returns the module's icon resource.
func (m *HotCueSyncModule) GetIcon() fyne.Resource {
	return theme.ContentCopyIcon()
}

// GetModuleContent returns the module's specific content without status messages
// This implements the method from ModuleBase to provide the module-specific UI
func (m *HotCueSyncModule) GetModuleContent() fyne.CanvasObject {
	// Form without submit button
	standardForm := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text: locales.Translate("hotcuesync.label.source"),
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
				Text: locales.Translate("hotcuesync.label.target"),
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
		widget.NewLabel(locales.Translate("hotcuesync.label.info")),
		widget.NewSeparator(),
		standardForm,
	)

	// Add submit button with right alignment
	buttonBox := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), m.submitBtn)
	contentContainer.Add(buttonBox)

	// Set initial values for source and target type without triggering change handlers
	m.IsLoadingConfig = true
	m.sourceType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)))
	m.targetType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)))
	m.IsLoadingConfig = false

	// Update controls visibility
	m.updateControlsState()

	return contentContainer
}

// GetContent constructs and returns the module's UI content.
func (m *HotCueSyncModule) GetContent() fyne.CanvasObject {
	// Check database requirements
	if m.dbMgr.GetDatabasePath() == "" {
		context := &common.ErrorContext{
			Module:      m.GetConfigName(),
			Operation:   "Database Validation",
			Severity:    common.ErrorWarning,
			Recoverable: true,
		}
		m.ErrorHandler.ShowStandardError(fmt.Errorf(locales.Translate("common.err.nodbpath")), context)
		m.sourcePlaylistSelect.Disable()
		m.targetPlaylistSelect.Disable()
		m.submitBtn.Disable()
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
		m.sourcePlaylistSelect.Disable()
		m.targetPlaylistSelect.Disable()
		m.submitBtn.Disable()
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
		m.sourcePlaylistSelect.Disable()
		m.targetPlaylistSelect.Disable()
		m.submitBtn.Disable()
		return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
	}

	// Enable interactive components if all checks passed
	m.sourcePlaylistSelect.Enable()
	m.targetPlaylistSelect.Enable()
	m.submitBtn.Enable()

	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig applies the configuration to the UI components.
func (m *HotCueSyncModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Check if configuration is nil or has uninitialized Extra field
	if common.IsNilConfig(cfg) {
		return
	}

	// Load source type
	sourceTypeStr := cfg.Get("source_type", string(SourceTypeFolder))
	sourceType := SourceType(sourceTypeStr)
	m.sourceType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(sourceType)))

	// Load target type
	targetTypeStr := cfg.Get("target_type", string(SourceTypeFolder))
	targetType := SourceType(targetTypeStr)
	m.targetType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(targetType)))

	// Load folder paths
	m.sourceFolderEntry.SetText(cfg.Get("source_folder", ""))
	m.targetFolderEntry.SetText(cfg.Get("target_folder", ""))

	// Save playlist IDs for later use
	m.sourcePlaylistID = cfg.Get("source_playlist", "")
	m.targetPlaylistID = cfg.Get("target_playlist", "")

	// Update UI visibility without loading playlists
	if sourceType == SourceTypeFolder {
		m.sourceFolderField.Show()
		m.sourcePlaylistSelect.Hide()
	} else {
		m.sourceFolderField.Hide()
		m.sourcePlaylistSelect.Show()
	}

	if targetType == SourceTypeFolder {
		m.targetFolderField.Show()
		m.targetPlaylistSelect.Hide()
	} else {
		m.targetFolderField.Hide()
		m.targetPlaylistSelect.Show()
	}

	// Set playlist selections if playlists are loaded
	if len(m.playlists) > 0 {
		for i, playlist := range m.playlists {
			if playlist.ID == m.sourcePlaylistID {
				if i < len(m.sourcePlaylistSelect.Options) {
					m.sourcePlaylistSelect.SetSelected(m.sourcePlaylistSelect.Options[i])
				}
				break
			}
		}

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

// getSourceType returns the current source type based on the selected dropdown value.
func (m *HotCueSyncModule) getSourceType() SourceType {
	if m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		return SourceTypeFolder
	}
	return SourceTypePlaylist
}

// getTargetType returns the current target type based on the selected dropdown value.
func (m *HotCueSyncModule) getTargetType() SourceType {
	if m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		return SourceTypeFolder
	}
	return SourceTypePlaylist
}

// initializeUI sets up the user interface components.
func (m *HotCueSyncModule) initializeUI() {
	// First create all UI components without handlers
	
	// Create type selectors
	m.sourceType = widget.NewSelect([]string{
		locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)),
		locales.Translate("hotcuesync.dropdown." + string(SourceTypePlaylist)),
	}, nil)

	m.targetType = widget.NewSelect([]string{
		locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)),
		locales.Translate("hotcuesync.dropdown." + string(SourceTypePlaylist)),
	}, nil)

	// Create folder entries
	m.sourceFolderEntry = widget.NewEntry()
	m.targetFolderEntry = widget.NewEntry()

	// Create playlist selectors
	m.sourcePlaylistSelect = common.CreateDisabledSelect([]string{}, nil, "common.select.plsplacehldrinact")
	m.targetPlaylistSelect = common.CreateDisabledSelect([]string{}, nil, "common.select.plsplacehldrinact")

	// Create folder selection fields
	m.sourceFolderField = common.CreateFolderSelectionField(
		locales.Translate("hotcuesync.folder.source"),
		m.sourceFolderEntry,
		func(path string) {
			m.sourceFolderEntry.SetText(path)
			m.SaveConfig()
		},
	)

	m.targetFolderField = common.CreateFolderSelectionField(
		locales.Translate("hotcuesync.folder.target"),
		m.targetFolderEntry,
		func(path string) {
			m.targetFolderEntry.SetText(path)
			m.SaveConfig()
		},
	)

	// Create submit button
	m.submitBtn = widget.NewButton(locales.Translate("hotcuesync.button.start"), func() {
		m.SaveConfig()
		go func() {
			err := m.synchronizeHotCues()
			if err != nil {
				context := common.NewErrorContext(m.GetConfigName(), "Synchronize Hot Cues")
				m.ErrorHandler.ShowStandardError(err, &context)
			}
		}()
	})
	m.submitBtn.Importance = widget.HighImportance

	// Now set up handlers and default values
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	m.sourceType.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.updateSourceVisibility(m.getSourceType())
		m.SaveConfig()
	})

	m.targetType.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.updateTargetVisibility(m.getTargetType())
		m.SaveConfig()
	})

	// Set default values last
	m.sourceType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)))
	m.targetType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)))
}

// updateControlsState updates the state of playlist selectors and loads playlists on first activation.
func (m *HotCueSyncModule) updateControlsState() {
	// Get current source and target types
	var sourceType, targetType SourceType
	if m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		sourceType = SourceTypeFolder
	} else {
		sourceType = SourceTypePlaylist
	}

	if m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
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
func (m *HotCueSyncModule) updateSourceVisibility(sourceType SourceType) {
	if sourceType == SourceTypeFolder {
		m.sourceFolderField.Show()
		m.sourcePlaylistSelect.Hide()
	} else {
		m.sourceFolderField.Hide()
		m.sourcePlaylistSelect.Show()
	}
}

// updateTargetVisibility updates the visibility of target input controls based on the selected target type.
func (m *HotCueSyncModule) updateTargetVisibility(targetType SourceType) {
	if targetType == SourceTypeFolder {
		m.targetFolderField.Show()
		m.targetPlaylistSelect.Hide()
	} else {
		m.targetFolderField.Hide()
		m.targetPlaylistSelect.Show()
	}
}

// loadPlaylists loads playlist items from the database and updates the playlist selectors.
func (m *HotCueSyncModule) loadPlaylists() error {
	// Update UI to show loading state
	m.UpdateProgressStatus(0, locales.Translate("hotcuesync.status.playlistload"))

	// Ensure database connection
	err := m.dbMgr.EnsureConnected(false)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("common.err.connectdb"), err)
	}

	// Query playlists from database
	query := `
		SELECT DISTINCT
			P.ID,
			P.Name,
			P.ParentID,
			CASE 
				WHEN p2.Name IS NOT NULL THEN p2.Name || ' > ' || P.Name
				ELSE P.Name
			END as Path
		FROM djmdPlaylist P
		LEFT JOIN djmdPlaylist p2 ON P.ParentID = p2.ID
		ORDER BY 
			CASE WHEN p2.ID IS NULL THEN P.Seq ELSE p2.Seq END,
			CASE WHEN p2.ID IS NULL THEN 0 ELSE P.Seq + 1 END
	`

	rows, err := m.dbMgr.Query(query)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("common.err.playlistload"), err)
	}
	defer rows.Close()

	// Clear existing playlists
	m.playlists = make([]common.PlaylistItem, 0)
	playlistNames := make([]string, 0)

	// Process query results
	for rows.Next() {
		var playlist common.PlaylistItem
		err := rows.Scan(
			&playlist.ID,
			&playlist.Name,
			&playlist.ParentID,
			&playlist.Path,
		)
		if err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("common.err.playlistload"), err)
		}

		m.playlists = append(m.playlists, playlist)
		playlistNames = append(playlistNames, playlist.Path)
	}

	// Update playlist selectors with new data
	m.sourcePlaylistSelect.Options = playlistNames
	m.targetPlaylistSelect.Options = playlistNames

	// Update progress status
	m.UpdateProgressStatus(0, "")

	return nil
}

// getSourceTracks retrieves source tracks from the database based on the selected source type.
func (m *HotCueSyncModule) getSourceTracks() ([]common.TrackItem, error) {
	// Debug output at the start of loading source tracks
	fmt.Printf("Starting to load source tracks...\n")
	fmt.Printf("Source type: %s\n", m.sourceType.Selected)

	var tracks []common.TrackItem
	var err error

	if m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		// Source is a folder
		folderPath := m.sourceFolderEntry.Text
		if folderPath == "" {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.nosrcfolder"))
		}

		// Debug output
		fmt.Printf("Loading source tracks from folder: %s\n", folderPath)

		tracks, err = m.dbMgr.GetTracksBasedOnFolder(folderPath)
		if err != nil {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.loadtrackdata"))
		}
	} else {
		// Source is a playlist
		if m.sourcePlaylistSelect.Selected == "" {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.nosrcplaylist"))
		}

		// Find playlist ID
		var playlistID string

		// First try to find the playlist by path
		for _, p := range m.playlists {
			if p.Path == m.sourcePlaylistSelect.Selected {
				playlistID = p.ID
				fmt.Printf("Found PlaylistID: %s for playlist path: %s\n", playlistID, p.Path)
				break
			}
		}

		if playlistID == "" {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.playlistnotfound"))
		}

		// Debug output for playlist ID
		fmt.Printf("Loading source tracks from playlist ID: %s\n", playlistID)

		tracks, err = m.dbMgr.GetTracksBasedOnPlaylist(playlistID)
		if err != nil {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.loadtrackdata"))
		}
	}

	if len(tracks) == 0 {
		return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.nosourcetracks"))
	}

	// Debug output for list of count of loaded tracks
	fmt.Printf("Loaded %d source tracks\n", len(tracks))

	return tracks, nil
}

// getTargetTracks retrieves target tracks from the database based on the selected target type.
func (m *HotCueSyncModule) getTargetTracks(sourceTrack common.TrackItem) ([]struct {
	ID       string
	FileName string
}, error) {
	// Debug output for the target type
	fmt.Printf("Target type: %s\n", m.targetType.Selected)

	// Extract the file name from the source track's folder path without extension
	fileName := filepath.Base(sourceTrack.FolderPath)
	relativePathWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	fmt.Printf("Source track relative path (without extension): %s\n", relativePathWithoutExt)

	var targetTracks []common.TrackItem
	var err error

	if m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		// Target is a folder
		targetFolderPath := m.targetFolderEntry.Text
		if targetFolderPath == "" {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.notgtfolder"))
		}

		// Debug output for folder path
		fmt.Printf("Loading target tracks from folder: %s\n", targetFolderPath)

		targetTracks, err = m.dbMgr.GetTracksBasedOnFolder(targetFolderPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.loadtrackdata"), err)
		}
	} else {
		// Target is a playlist
		if m.targetPlaylistSelect.Selected == "" {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.notgtplaylist"))
		}

		// Find playlist ID
		var playlistID string

		// First try to find the playlist by path
		for _, p := range m.playlists {
			if p.Path == m.targetPlaylistSelect.Selected {
				playlistID = p.ID
				fmt.Printf("Found PlaylistID: %s for playlist path: %s\n", playlistID, p.Path)
				break
			}
		}

		if playlistID == "" {
			return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.playlistnotfound"))
		}

		// Debug output for playlist ID
		fmt.Printf("Loading target tracks from playlist ID: %s\n", playlistID)

		targetTracks, err = m.dbMgr.GetTracksBasedOnPlaylist(playlistID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.loadtrackdata"), err)
		}
	}

	// Prepare final result slice
	var result []struct {
		ID       string
		FileName string
	}

	// Debug output for target tracks
	fmt.Printf("DEBUG: Finding match for source file: %s\n", relativePathWithoutExt)
	fmt.Printf("DEBUG: All files found in target:\n")

	// Omit the source track from the destination
	for _, track := range targetTracks {
		if track.ID == sourceTrack.ID {
			continue
		}

		// Get the relative path of the target file without the extension
		targetFileName := filepath.Base(track.FolderPath)
		targetRelativePathWithoutExt := strings.TrimSuffix(targetFileName, filepath.Ext(targetFileName))
		fmt.Printf("DEBUG: - %s (ID: %s)\n", targetRelativePathWithoutExt, track.ID)

		// Compare relative paths (without extension) using case-sensitive comparison
		if targetRelativePathWithoutExt == relativePathWithoutExt {
			fmt.Printf("MATCH FOUND: Source=%s, Target=%s\n", relativePathWithoutExt, targetRelativePathWithoutExt)
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
		// No matching target tracks found
		fmt.Printf(locales.Translate("hotcuesync.err.notgttracks")+": %s\n", fileName)
		return []struct {
			ID       string
			FileName string
		}{}, nil
	}

	return result, nil
}

// copyHotCues copies hot cues from the source track to the target track.
func (m *HotCueSyncModule) copyHotCues(sourceID, targetID string) error {
	fmt.Printf("    Copying hot cues from source ID %s to target ID %s\n", sourceID, targetID)

	hotCues, err := m.dbMgr.GetTrackHotCues(sourceID)
	if err != nil {
		fmt.Printf("    Error querying hot cues: %v\n", err)
		return fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.querycues"), err)
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
			fmt.Printf("    Warning: Hot cue without Kind value found\n")
			continue
		}

		// List the hot cue details
		fmt.Printf("    Processing hot cue %d: ID=%v, Kind=%v\n",
			hotCueCount, hotCue["ID"], kind)

		// Delete existing hot cues with the same Kind value in the target track
		err = m.dbMgr.Execute(`DELETE FROM djmdCue WHERE ContentID = ? AND Kind = ?`, targetID, kind)
		if err != nil {
			fmt.Printf("    Error deleting existing hot cue: %v\n", err)
			return fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.deletecue"), err)
		}

		// Generate a new ID for the hot cue in the target track
		var maxID int64
		err = m.dbMgr.QueryRow("SELECT COALESCE(MAX(CAST(ID AS INTEGER)), 0) FROM djmdCue").Scan(&maxID)
		if err != nil {
			fmt.Printf("    Error getting max ID: %v\n", err)
			return fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.maxidcheck"), err)
		}
		maxID++
		newID := fmt.Sprintf("%d", maxID)
		fmt.Printf("    Generated new ID: %s for hot cue\n", newID)

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
			fmt.Printf("    Error inserting hot cue: %v\n", err)
			return fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.cueinsert"), err)
		}
		fmt.Printf("    Successfully inserted hot cue with ID %s\n", newID)
	}

	if hotCueCount == 0 {
		fmt.Printf("    No hot cues found for source track ID %s\n", sourceID)
	} else {
		fmt.Printf("    Successfully copied %d hot cues from source ID %s to target ID %s\n",
			hotCueCount, sourceID, targetID)
	}

	return nil
}

// copyTrackMetadata copies specific metadata fields from source track to target track.
// Fields copied: StockDate, DateCreated, ColorID, DJPlayCount
func (m *HotCueSyncModule) copyTrackMetadata(sourceID, targetID string) error {
	fmt.Printf("    Copying track metadata from source ID %s to target ID %s\n", sourceID, targetID)

	// Query to get source track metadata
	query := `
        SELECT StockDate, DateCreated, ColorID, DJPlayCount
        FROM djmdContent
        WHERE ID = ?
    `

	row := m.dbMgr.QueryRow(query, sourceID)
	if row == nil {
		return fmt.Errorf("%s", locales.Translate("hotcuesync.err.querysource"))
	}

	// Scan source track metadata using our custom null types
	var stockDate common.NullString
	var dateCreated common.NullString
	var colorID common.NullInt64
	var djPlayCount common.NullInt64

	err := row.Scan(&stockDate, &dateCreated, &colorID, &djPlayCount)
	if err != nil {
		fmt.Printf("    Error scanning source track metadata: %v\n", err)
		return fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.metadatascan"), err)
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
		fmt.Printf("    Error updating target track metadata: %v\n", err)
		return fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.metadataupdate"), err)
	}

	fmt.Printf("    Successfully copied metadata from source ID %s to target ID %s\n", sourceID, targetID)
	return nil
}

// synchronizeHotCues performs the main hot cue synchronization process.
// It copies hot cues from source tracks to matching target tracks.
func (m *HotCueSyncModule) synchronizeHotCues() error {
	// Disable the button during processing
	m.submitBtn.Disable()
	defer func() {
		m.submitBtn.Enable()
	}()

	// Basic validation
	if (m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) && m.sourceFolderEntry.Text == "") ||
		(m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) && m.targetFolderEntry.Text == "") {
		context := common.NewErrorContext(m.GetConfigName(), "Validate Paths")
		m.ErrorHandler.ShowStandardError(fmt.Errorf("%s", locales.Translate("hotcuesync.err.emptypaths")), &context)
		return fmt.Errorf("%s", locales.Translate("hotcuesync.err.emptypaths"))
	}

	// Show a progress dialog
	m.ShowProgressDialog(locales.Translate("hotcuesync.dialog.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				context := common.NewErrorContext(m.GetConfigName(), "Panic Recovery")
				m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %v", locales.Translate("hotcuesync.err.panic"), r), &context)
			}
		}()

		// Clear previous status messages
		m.ClearStatusMessages()

		// Initial progress
		m.UpdateProgressStatus(0.0, locales.Translate("common.status.start"))

		// Add initial status message
		m.AddInfoMessage(locales.Translate("common.status.start"))

		// Get source tracks based on selected source type
		sourceTracks, err := m.getSourceTracks()
		if err != nil {
			m.CloseProgressDialog()
			context := common.NewErrorContext(m.GetConfigName(), "Get Source Tracks")
			m.ErrorHandler.ShowStandardError(err, &context)
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Update progress
		m.UpdateProgressStatus(0.1, locales.Translate("common.status.reading"))

		// Add status message about reading source tracks
		m.AddInfoMessage(locales.Translate("common.status.reading"))

		// Create a backup of the database
		err = m.dbMgr.BackupDatabase()
		if err != nil {
			m.CloseProgressDialog()
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Database Backup")
			context.Severity = common.ErrorWarning
			m.ErrorHandler.ShowStandardError(err, &context)
			return
		}

		// Add status message about successful database backup
		m.AddInfoMessage(locales.Translate("common.db.backupdone"))

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Start a database transaction
		err = m.dbMgr.BeginTransaction()
		if err != nil {
			m.CloseProgressDialog()
			context := common.NewErrorContext(m.GetConfigName(), "Begin Transaction")
			m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.db.txbeginerr"), err), &context)
			return
		}

		// Ensure database connection is properly closed when done
		defer func() {
			if err := m.dbMgr.Finalize(); err != nil {
				finalizeContext := common.NewErrorContext(m.GetConfigName(), "Close Database")
				m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.db.dbcloseerr"), err), &finalizeContext)
			}
		}()

		// Ensure transaction is rolled back on error
		defer func() {
			if err != nil {
				if rollbackErr := m.dbMgr.RollbackTransaction(); rollbackErr != nil {
					rollbackContext := common.NewErrorContext(m.GetConfigName(), "Rollback Transaction")
					m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.db.rollbackerr"), rollbackErr), &rollbackContext)
				}
			}
		}()

		// Track successful and skipped files
		successCount := 0
		skippedCount := 0

		// Update progress before processing
		m.UpdateProgressStatus(0.2, locales.Translate("common.status.updating"))

		// Add status message about starting the update process
		m.AddInfoMessage(locales.Translate("common.status.updating"))

		// Process each source track
		for i, sourceTrack := range sourceTracks {
			// Check if operation was cancelled
			if m.IsCancelled() {
				m.CloseProgressDialog()
				m.dbMgr.RollbackTransaction()
				return
			}

			// Update progress
			progress := 0.2 + (float64(i+1) / float64(len(sourceTracks)) * 0.8)
			m.UpdateProgressStatus(progress, fmt.Sprintf("%s: %d/%d", locales.Translate("hotcuesync.diagstatus.process"), i+1, len(sourceTracks)))

			// Get target tracks for this source track
			targetTracks, err := m.getTargetTracks(sourceTrack)
			if err != nil {
				m.CloseProgressDialog()
				context := common.NewErrorContext(m.GetConfigName(), "Get Target Tracks")
				m.ErrorHandler.ShowStandardError(err, &context)
				m.dbMgr.RollbackTransaction()
				return
			}

			// Skip if no target tracks found
			if len(targetTracks) == 0 {
				skippedCount++
				continue
			}

			// Copy hot cues and metadata to each target track
			for _, targetTrack := range targetTracks {
				// Check if operation was cancelled
				if m.IsCancelled() {
					m.CloseProgressDialog()
					m.dbMgr.RollbackTransaction()
					return
				}

				// Copy hot cues
				err = m.copyHotCues(sourceTrack.ID, targetTrack.ID)
				if err != nil {
					m.CloseProgressDialog()
					context := common.NewErrorContext(m.GetConfigName(), "Copy Hot Cues")
					m.ErrorHandler.ShowStandardError(err, &context)
					m.dbMgr.RollbackTransaction()
					return
				}

				// Copy track metadata
				err = m.copyTrackMetadata(sourceTrack.ID, targetTrack.ID)
				if err != nil {
					m.CloseProgressDialog()
					context := common.NewErrorContext(m.GetConfigName(), "Copy Track Metadata")
					m.ErrorHandler.ShowStandardError(err, &context)
					m.dbMgr.RollbackTransaction()
					return
				}
			}
			successCount++

			// Small delay to prevent database overload
			time.Sleep(10 * time.Millisecond)
		}

		// Commit the transaction
		err = m.dbMgr.CommitTransaction()
		if err != nil {
			m.CloseProgressDialog()
			context := common.NewErrorContext(m.GetConfigName(), "Commit Transaction")
			m.ErrorHandler.ShowStandardError(fmt.Errorf("%s: %w", locales.Translate("common.db.txcommiterr"), err), &context)
			return
		}

		// Create summary message
		summaryMessage := fmt.Sprintf(locales.Translate("hotcuesync.status.completed"), successCount, skippedCount)

		// Update status message
		m.UpdateProgressStatus(1.0, summaryMessage)

		// Add final status message about completion
		m.AddInfoMessage(summaryMessage)

		// Mark the progress dialog as completed instead of closing it
		m.CompleteProgressDialog()
		m.submitBtn.SetIcon(theme.ConfirmIcon()) // Set icon to checkmark on success
	}()

	return nil
}
