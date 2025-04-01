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

	_ "github.com/mutecomm/go-sqlcipher/v4"
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
	IsInitializing       bool
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
		IsInitializing: true, // Set initializing flag to prevent database connection during startup
	}
	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	// Update controls state without loading playlists
	m.updateControlsState()

	// Reset initializing flag after initialization is complete
	m.IsInitializing = false

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

// GetContent constructs and returns the module's UI content.
func (m *HotCueSyncModule) GetContent() fyne.CanvasObject {
	infoLabel := widget.NewLabel(locales.Translate("hotcuesync.label.info"))
	infoLabel.Wrapping = fyne.TextWrapWord
	infoLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Containers for source inputs.
	sourceTypeContainer := container.NewBorder(
		nil, nil,
		nil, nil,
		m.sourceType,
	)
	sourceContainer := container.NewBorder(
		nil, nil,
		sourceTypeContainer, nil,
		container.NewStack(
			m.sourceFolderField,
			m.sourcePlaylistSelect,
		),
	)

	// Containers for target inputs.
	targetTypeContainer := container.NewBorder(
		nil, nil,
		nil, nil,
		m.targetType,
	)
	targetContainer := container.NewBorder(
		nil, nil,
		targetTypeContainer, nil,
		container.NewStack(
			m.targetFolderField,
			m.targetPlaylistSelect,
		),
	)

	// Create a button with a dynamic icon.
	var submitBtn *widget.Button
	submitBtn = widget.NewButtonWithIcon(locales.Translate("hotcuesync.button.start"), nil, func() {
		// Save configuration before synchronization.
		m.SaveConfig()
		go func() {
			err := m.synchronizeHotCues()
			if err != nil {
				common.ShowError(err, m.Window)
			} else {
				// Set the icon to a checkmark upon successful completion.
				submitBtn.SetIcon(theme.ConfirmIcon())
			}
		}()
	})
	submitBtn.Importance = widget.HighImportance
	m.submitBtn = submitBtn

	// Form with submit button.
	standardForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("hotcuesync.label.source"), Widget: sourceContainer},
			{Text: locales.Translate("hotcuesync.label.target"), Widget: targetContainer},
		},
		SubmitText: "",
		OnSubmit:   nil, // OnSubmit is now controlled directly by the button.
	}

	// Insert the button into the layout.
	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		standardForm,
		container.NewHBox(layout.NewSpacer(), submitBtn),
	)

	return content
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

	// Save playlist IDs for later use when playlists are loaded
	m.sourcePlaylistID = cfg.Get("source_playlist", "")
	m.targetPlaylistID = cfg.Get("target_playlist", "")

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

	// Update UI state based on loaded configuration only if not initializing
	if !m.IsInitializing {
		m.updateControlsState()
	}
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
func (m *HotCueSyncModule) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	cfg := common.NewModuleConfig()

	// Save source type
	var sourceType SourceType
	if m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		sourceType = SourceTypeFolder
	} else {
		sourceType = SourceTypePlaylist
	}
	cfg.Set("source_type", string(sourceType))

	// Save target type
	var targetType SourceType
	if m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		targetType = SourceTypeFolder
	} else {
		targetType = SourceTypePlaylist
	}
	cfg.Set("target_type", string(targetType))

	// Save folder paths
	cfg.Set("source_folder", m.sourceFolderEntry.Text)
	cfg.Set("target_folder", m.targetFolderEntry.Text)

	// Save playlist selections
	if sourceType == SourceTypePlaylist && m.sourcePlaylistSelect.Selected != "" {
		for _, playlist := range m.playlists {
			if playlist.Path == m.sourcePlaylistSelect.Selected {
				cfg.Set("source_playlist", playlist.ID)
				break
			}
		}
	}

	if targetType == SourceTypePlaylist && m.targetPlaylistSelect.Selected != "" {
		for _, playlist := range m.playlists {
			if playlist.Path == m.targetPlaylistSelect.Selected {
				cfg.Set("target_playlist", playlist.ID)
				break
			}
		}
	}

	// Save config to the config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)

	return cfg
}

// initializeUI sets up the user interface components.
func (m *HotCueSyncModule) initializeUI() {
	// Initialize source type selector
	m.sourceType = widget.NewSelect([]string{
		locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)),
		locales.Translate("hotcuesync.dropdown." + string(SourceTypePlaylist)),
	}, func(selected string) {
		var sourceType SourceType
		if selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
			sourceType = SourceTypeFolder
		} else {
			sourceType = SourceTypePlaylist
		}
		m.updateSourceVisibility(sourceType)
		if !m.IsLoadingConfig {
			m.SaveConfig()
		}
	})

	// Initialize target type selector
	m.targetType = widget.NewSelect([]string{
		locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)),
		locales.Translate("hotcuesync.dropdown." + string(SourceTypePlaylist)),
	}, func(selected string) {
		var targetType SourceType
		if selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
			targetType = SourceTypeFolder
		} else {
			targetType = SourceTypePlaylist
		}
		m.updateTargetVisibility(targetType)
		if !m.IsLoadingConfig {
			m.SaveConfig()
		}
	})

	// Initialize source folder field
	m.sourceFolderEntry = widget.NewEntry()
	m.sourceFolderEntry.TextStyle = fyne.TextStyle{Monospace: true}
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})
	m.sourceFolderField = common.CreateFolderSelectionField(
		locales.Translate("hotcuesync.data.source"),
		m.sourceFolderEntry,
		m.CreateChangeHandler(func() {
			m.SaveConfig()
		}),
	)

	// Initialize target folder field
	m.targetFolderEntry = widget.NewEntry()
	m.targetFolderEntry.TextStyle = fyne.TextStyle{Monospace: true}
	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})
	m.targetFolderField = common.CreateFolderSelectionField(
		locales.Translate("hotcuesync.data.target"),
		m.targetFolderEntry,
		m.CreateChangeHandler(func() {
			m.SaveConfig()
		}),
	)

	// Initialize source playlist selector
	m.sourcePlaylistSelect = widget.NewSelect([]string{}, nil)
	m.sourcePlaylistSelect.PlaceHolder = locales.Translate("hotcuesync.label.playlistsel")
	m.sourcePlaylistSelect.Disable() // Initially disabled
	m.sourcePlaylistSelect.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Initialize target playlist selector
	m.targetPlaylistSelect = widget.NewSelect([]string{}, nil)
	m.targetPlaylistSelect.PlaceHolder = locales.Translate("hotcuesync.label.playlistsel")
	m.targetPlaylistSelect.Disable() // Initially disabled
	m.targetPlaylistSelect.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Create a form with source and target containers.
	standardForm := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text: locales.Translate("hotcuesync.data.source"),
				Widget: container.NewBorder(
					nil, nil,
					widget.NewLabel(locales.Translate("hotcuesync.label.sourcetype")), nil,
					m.sourceType,
				),
			},
			{
				Text: locales.Translate("hotcuesync.data.target"),
				Widget: container.NewBorder(
					nil, nil,
					widget.NewLabel(locales.Translate("hotcuesync.data.targettype")), nil,
					m.targetType,
				),
			},
		},
		OnSubmit: func() {
			if !m.IsLoadingConfig {
				m.SaveConfig()
			}
			go func() {
				err := m.synchronizeHotCues()
				if err != nil {
					common.ShowError(err, m.Window)
				}
			}()
		},
		SubmitText: locales.Translate("hotcuesync.button.start"),
	}

	// Arrange components in a vertical box.
	m.Window.SetContent(container.NewVBox(
		widget.NewLabel(locales.Translate("hotcuesync.label.info")),
		widget.NewSeparator(),
		standardForm,
		widget.NewSeparator(),
	))

	// Set initial values for source and target type without triggering change handlers
	m.IsLoadingConfig = true
	m.sourceType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)))
	m.targetType.SetSelected(locales.Translate("hotcuesync.dropdown." + string(SourceTypeFolder)))
	m.IsLoadingConfig = false
}

// updateControlsState updates the state of playlist selectors and loads playlists on first activation.
func (m *HotCueSyncModule) updateControlsState() {
	// Load playlists if needed and not initializing
	if len(m.playlists) == 0 && !m.IsInitializing {
		if err := m.loadPlaylists(); err != nil {
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Load Playlists")
			m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
		}
	}

	// Update visibility based on current selections
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

	// Only update visibility if not initializing, or update without loading playlists
	if m.IsInitializing {
		// Just update UI visibility without loading playlists
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
	} else {
		// Normal operation - update visibility with possible playlist loading
		m.updateSourceVisibility(sourceType)
		m.updateTargetVisibility(targetType)
	}
}

// updateSourceVisibility updates the visibility of source input controls based on the selected source type.
func (m *HotCueSyncModule) updateSourceVisibility(sourceType SourceType) {
	// Load playlists only if playlist type is selected and not initializing
	if sourceType == SourceTypePlaylist && !m.IsInitializing {
		m.loadPlaylists()
	}

	// Update visibility based on selected source type
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
	// Load playlists only if playlist type is selected and not initializing
	if targetType == SourceTypePlaylist && !m.IsInitializing {
		m.loadPlaylists()
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

// loadPlaylists loads playlist items from the database and updates the playlist selectors.
func (m *HotCueSyncModule) loadPlaylists() error {
	// Clear existing playlists
	m.playlists = nil

	// Update UI to show loading state
	m.UpdateProgressStatus(0, locales.Translate("hotcuesync.status.loadingplaylists"))

	// Skip database connection during initialization
	if m.IsInitializing {
		// During initialization, we don't need to load playlists
		return nil
	}

	// Ensure database connection
	err := m.dbMgr.EnsureConnected(false)
	if err != nil {
		return err
	}

	// Make sure to close the database connection when done
	defer func() {
		if finalizeErr := m.dbMgr.Finalize(); finalizeErr != nil {
			fmt.Printf("Error closing database connection: %v\n", finalizeErr)
		}
	}()

	// Query playlists from the database
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
		return fmt.Errorf(locales.Translate("hotcuesync.err.loadplaylists"), err)
	}
	defer rows.Close()

	// Process query results
	var playlistNames []string
	for rows.Next() {
		var playlist common.PlaylistItem
		err := rows.Scan(&playlist.ID, &playlist.Name, &playlist.ParentID, &playlist.Path)
		if err != nil {
			return fmt.Errorf(locales.Translate("hotcuesync.err.scanplaylists"), err)
		}

		m.playlists = append(m.playlists, playlist)
		playlistNames = append(playlistNames, playlist.Path)
	}

	// Debug output for available playlists
	fmt.Printf("Available playlists:\n")
	for _, p := range m.playlists {
		fmt.Printf("- %s (ID: %s)\n", p.Path, p.ID)
	}

	// Update playlist selectors
	m.sourcePlaylistSelect.Options = playlistNames
	m.targetPlaylistSelect.Options = playlistNames

	// Enable playlist selectors if we have playlists
	if len(playlistNames) > 0 {
		m.sourcePlaylistSelect.Enable()
		m.targetPlaylistSelect.Enable()

		// Use stored playlist IDs from module
		// Restore source playlist selection
		for i, playlist := range m.playlists {
			if playlist.ID == m.sourcePlaylistID {
				if i < len(m.sourcePlaylistSelect.Options) {
					m.sourcePlaylistSelect.SetSelected(m.sourcePlaylistSelect.Options[i])
				}
				break
			}
		}

		// Restore target playlist selection
		for i, playlist := range m.playlists {
			if playlist.ID == m.targetPlaylistID {
				if i < len(m.targetPlaylistSelect.Options) {
					m.targetPlaylistSelect.SetSelected(m.targetPlaylistSelect.Options[i])
				}
				break
			}
		}
	}

	// Clear loading status
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

		// Debug output
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
		m.ErrorHandler.HandleError(fmt.Errorf("%s", locales.Translate("hotcuesync.err.emptypaths")), common.NewErrorContext(m.GetConfigName(), "Empty Paths"), m.Window, m.Status)
		return fmt.Errorf("%s", locales.Translate("hotcuesync.err.emptypaths"))
	}

	// Show a progress dialog
	m.ShowProgressDialog(locales.Translate("hotcuesync.dialog.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf("%s: %v", locales.Translate("hotcuesync.err.panic"), r), common.NewErrorContext(m.GetConfigName(), "Panic"), m.Window, m.Status)
			}
		}()

		// Initial progress
		m.UpdateProgressStatus(0.0, locales.Translate("hotcuesync.status.start"))

		// Get source tracks based on selected source type
		sourceTracks, err := m.getSourceTracks()
		if err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.HandleError(err, common.NewErrorContext(m.GetConfigName(), "Source Tracks"), m.Window, m.Status)
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Update progress
		m.UpdateProgressStatus(0.1, locales.Translate("hotcuesync.status.reading"))

		// Create a backup of the database
		err = m.dbMgr.BackupDatabase()
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

		// Start a database transaction
		err = m.dbMgr.BeginTransaction()
		if err != nil {
			m.CloseProgressDialog()
			m.ErrorHandler.HandleError(fmt.Errorf("%s: %w", locales.Translate("common.db.txbeginerr"), err), common.NewErrorContext(m.GetConfigName(), "Transaction Start"), m.Window, m.Status)
			return
		}

		// Ensure database connection is properly closed when done
		defer func() {
			if err := m.dbMgr.Finalize(); err != nil {
				m.ErrorHandler.HandleError(fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.dbclose"), err),
					common.NewErrorContext(m.GetConfigName(), "Database Close"), m.Window, m.Status)
			}
		}()

		// Ensure transaction is rolled back on error
		defer func() {
			if err != nil {
				if rollbackErr := m.dbMgr.RollbackTransaction(); rollbackErr != nil {
					m.ErrorHandler.HandleError(fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.rollback"), rollbackErr),
						common.NewErrorContext(m.GetConfigName(), "Transaction Rollback"), m.Window, m.Status)
				}
			}
		}()

		// Track successful and skipped files
		successCount := 0
		skippedCount := 0

		// Update progress before processing
		m.UpdateProgressStatus(0.2, locales.Translate("hotcuesync.status.updating"))

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
			m.UpdateProgressStatus(progress, fmt.Sprintf("%s: %d/%d", locales.Translate("hotcuesync.status.process"), i+1, len(sourceTracks)))

			// Get target tracks for this source track
			targetTracks, err := m.getTargetTracks(sourceTrack)
			if err != nil {
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf("%s: %w", locales.Translate("hotcuesync.err.processtrack"), err), common.NewErrorContext(m.GetConfigName(), "Target Tracks"), m.Window, m.Status)
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
					m.ErrorHandler.HandleError(err, common.NewErrorContext(m.GetConfigName(), "Copy HotCues"), m.Window, m.Status)
					m.dbMgr.RollbackTransaction()
					return
				}

				// Copy track metadata
				err = m.copyTrackMetadata(sourceTrack.ID, targetTrack.ID)
				if err != nil {
					m.CloseProgressDialog()
					m.ErrorHandler.HandleError(err, common.NewErrorContext(m.GetConfigName(), "Copy Metadata"), m.Window, m.Status)
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
			m.ErrorHandler.HandleError(fmt.Errorf("%s: %w", locales.Translate("common.db.txcommiterr"), err), common.NewErrorContext(m.GetConfigName(), "Transaction Commit"), m.Window, m.Status)
			return
		}

		// Create summary message
		summaryMessage := fmt.Sprintf(locales.Translate("hotcuesync.status.completed"), successCount, skippedCount)

		// Update status message
		m.UpdateProgressStatus(1.0, summaryMessage)

		// Mark the progress dialog as completed instead of closing it
		m.CompleteProgressDialog()
	}()

	return nil
}
