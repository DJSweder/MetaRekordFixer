package modules

import (
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"time"

	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

func (m *HotCueSyncModule) GetName() string {
	return locales.Translate("hotcuesync.mod.name")
}

// GetConfigName returns the configuration key for this module.
func (m *HotCueSyncModule) GetConfigName() string {
	return "hotcue_sync"
}

func (m *HotCueSyncModule) GetIcon() fyne.Resource {
	return theme.MediaPlayIcon()
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

	// Predeclare submit button.
	var submitBtn *widget.Button
	// Create a button with a dynamic icon.
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
				// Odstraněno zobrazení informačního dialogu, protože progress bar již zobrazuje informaci o dokončení
				// dialog.ShowInformation(locales.Translate("hotcuesync.success.title"), locales.Translate("hotcuesync.success.msg"), m.Window)
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
		// Odstraněno: progress bar a status label, protože jsou nahrazeny dialogem s progress barem
		// m.Progress,
		// m.Status,
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
	// Odstraněno: inicializace progress baru a status labelu, protože jsou nahrazeny dialogem s progress barem
	// m.Progress = widget.NewProgressBar()
	// m.Status = widget.NewLabel("")
	// m.Status.Alignment = fyne.TextAlignCenter

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
				} else {
					// Odstraněno zobrazení informačního dialogu, protože progress bar již zobrazuje informaci o dokončení
					// dialog.ShowInformation(
					// 	locales.Translate("hotcuesync.success.title"),
					// 	locales.Translate("hotcuesync.success.msg"),
					// 	m.Window,
					// )
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
		// Odstraněno: progress bar a status label, protože jsou nahrazeny dialogem s progress barem
		// m.Progress,
		// m.Status,
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
		err := m.loadPlaylists()
		if err != nil {
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
	defer m.dbMgr.Finalize()

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
func (m *HotCueSyncModule) getSourceTracks() ([]struct {
	ID          string
	FolderPath  string
	FileName    string
	StockDate   sql.NullString
	DateCreated sql.NullString
	ColorID     sql.NullInt64
	DJPlayCount sql.NullInt64
}, error) {
	// Debug output at the start of loading source tracks
	fmt.Printf("Starting to load source tracks...\n")
	fmt.Printf("Source type: %s\n", m.sourceType.Selected)
	fmt.Printf("Target type: %s\n", m.targetType.Selected)

	// Prepare query based on source type
	var query string
	var args []interface{}
	var playlistID string

	if m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		// Source is a folder
		folderPath := m.sourceFolderEntry.Text
		if folderPath == "" {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.nosrcfolder"))
		}

		// Convert path to database format
		dbPath := common.ToDbPath(folderPath, true)

		// Debug output for folder path
		fmt.Printf("Querying tracks in folder path: %s\n", dbPath)

		// Query tracks in the folder
		query = `
			SELECT c.ID, c.FolderPath, c.FileNameL, c.StockDate, c.DateCreated, c.ColorID, c.DJPlayCount
			FROM djmdContent c
			WHERE c.FolderPath LIKE ?
		`
		args = append(args, dbPath+"%")
		fmt.Printf("Loading source tracks from folder: %s\n", folderPath)
	} else {
		// Source is a playlist
		if m.sourcePlaylistSelect.Selected == "" {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.nosrcplaylist"))
		}

		// Nejprve zkusíme najít playlist podle ID v konfiguraci
		playlistID = m.sourcePlaylistSelect.Selected

		// Pokud to není ID, zkusíme najít podle cesty
		if _, err := strconv.ParseInt(playlistID, 10, 64); err != nil {
			// Není to číslo, takže to není ID - hledáme podle cesty
			playlistID = ""
			for _, p := range m.playlists {
				if p.Path == m.sourcePlaylistSelect.Selected {
					playlistID = p.ID
					fmt.Printf("Found PlaylistID: %s for playlist path: %s\n", playlistID, p.Path)
					break
				}
			}
		} else {
			// Je to číslo, takže to je ID - vypíšeme informaci
			fmt.Printf("Using direct PlaylistID: %s\n", playlistID)
		}

		if playlistID == "" {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.playlistnotfound"))
		}

		// Debug output for playlist ID
		fmt.Printf("Querying tracks for playlist ID: %s\n", playlistID)

		// Query tracks in the playlist
		query = `
			SELECT c.ID, c.FolderPath, c.FileNameL, c.StockDate, c.DateCreated, c.ColorID, c.DJPlayCount
			FROM djmdContent c
			JOIN djmdSongPlaylist sp ON c.ID = sp.ContentID
			WHERE sp.PlaylistID = ?
		`
		args = append(args, playlistID)
		fmt.Printf("Loading source tracks from playlist ID: %s\n", playlistID)
	}

	// Debug output for the query and arguments
	fmt.Printf("Executing query:\n%s with args: %v\n", query, args)
	// Execute the query
	rows, err := m.dbMgr.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process results
	var sourceTracks []struct {
		ID          string
		FolderPath  string
		FileName    string
		StockDate   sql.NullString
		DateCreated sql.NullString
		ColorID     sql.NullInt64
		DJPlayCount sql.NullInt64
	}

	for rows.Next() {
		var track struct {
			ID          string
			FolderPath  string
			FileName    string
			StockDate   sql.NullString
			DateCreated sql.NullString
			ColorID     sql.NullInt64
			DJPlayCount sql.NullInt64
		}

		if err := rows.Scan(
			&track.ID,
			&track.FolderPath,
			&track.FileName,
			&track.StockDate,
			&track.DateCreated,
			&track.ColorID,
			&track.DJPlayCount,
		); err != nil {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.loadtrackdata"), err)
		}

		// Debug output for track
		fmt.Printf("Loaded source track: ID=%s, FolderPath=%s, FileName=%s\n", track.ID, track.FolderPath, track.FileName)

		sourceTracks = append(sourceTracks, track)
	}

	if len(sourceTracks) == 0 {
		return nil, fmt.Errorf("%s", locales.Translate("hotcuesync.err.nosourcetracks"))
	}

	return sourceTracks, nil
}

// getTargetTracks retrieves target tracks from the database based on the selected target type.
func (m *HotCueSyncModule) getTargetTracks(sourceTrack struct {
	ID          string
	FolderPath  string
	FileName    string
	StockDate   sql.NullString
	DateCreated sql.NullString
	ColorID     sql.NullInt64
	DJPlayCount sql.NullInt64
}) ([]struct {
	ID       string
	FileName string
}, error) {
	// Debug output for the target type
	fmt.Printf("Target type: %s\n", m.targetType.Selected)

	// Extrahujeme relativní cestu bez přípony pro porovnání
	fileName := filepath.Base(sourceTrack.FolderPath)
	relativePathWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	fmt.Printf("Source track relative path (without extension): %s\n", relativePathWithoutExt)

	// Prepare query based on target type
	var query string
	var args []interface{}

	if m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) {
		// Target is a folder
		targetFolderPath := m.targetFolderEntry.Text
		if targetFolderPath == "" {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.notgtfolder"))
		}

		// Convert path to database format
		dbPath := common.ToDbPath(targetFolderPath, true)

		// Debug output for folder path
		fmt.Printf("Querying tracks in target folder path: %s\n", dbPath)

		// Získáme všechny soubory v cílové složce
		query = `
			SELECT c.ID, c.FileNameL, c.FolderPath
			FROM djmdContent c
			WHERE c.FolderPath LIKE ?
			AND c.ID <> ?
			ORDER BY c.FileNameL
		`
		args = append(args, dbPath+"%", sourceTrack.ID)
	} else {
		// Target is a playlist
		if m.targetPlaylistSelect.Selected == "" {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.notgtplaylist"))
		}

		// Find playlist ID
		var playlistID string

		// Nejprve zkusíme najít playlist podle ID v konfiguraci
		playlistID = m.targetPlaylistSelect.Selected

		// Pokud to není ID, zkusíme najít podle cesty
		if _, err := strconv.ParseInt(playlistID, 10, 64); err != nil {
			// Není to číslo, takže to není ID - hledáme podle cesty
			for _, p := range m.playlists {
				if p.Path == m.targetPlaylistSelect.Selected {
					playlistID = p.ID
					fmt.Printf("Found PlaylistID: %s for playlist path: %s\n", playlistID, p.Path)
					break
				}
			}
		} else {
			// Je to číslo, takže to je ID - vypíšeme informaci
			fmt.Printf("Using direct PlaylistID: %s\n", playlistID)
		}

		if playlistID == "" {
			return nil, fmt.Errorf(locales.Translate("hotcuesync.err.playlistnotfound"))
		}

		// Debug output for playlist ID
		fmt.Printf("Querying tracks for playlist ID: %s\n", playlistID)

		// Získáme všechny soubory v playlistu
		query = `
			SELECT c.ID, c.FileNameL, c.FolderPath
			FROM djmdContent c
			JOIN djmdSongPlaylist sp ON c.ID = sp.ContentID
			WHERE sp.PlaylistID = ?
			AND c.ID <> ?
			ORDER BY c.FileNameL
		`
		args = append(args, playlistID, sourceTrack.ID)
	}

	// Debug output for the query and arguments
	fmt.Printf("Executing query: %s with args: %v\n", query, args)
	// Execute the query
	rows, err := m.dbMgr.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process results
	var targetTracks []struct {
		ID       string
		FileName string
	}

	// Přidáme debugovací výpisy pro všechna nalezená soubory
	fmt.Printf("DEBUG: Hledáme shodu pro zdrojový soubor: %s\n", relativePathWithoutExt)
	fmt.Printf("DEBUG: Všechna nalezená soubory v cíli:\n")

	for rows.Next() {
		var track struct {
			ID         string
			FileName   string
			FolderPath string
		}
		if err := rows.Scan(&track.ID, &track.FileName, &track.FolderPath); err != nil {
			return nil, err
		}

		// Získáme relativní cestu cílového souboru bez přípony
		targetFileName := filepath.Base(track.FolderPath)
		targetRelativePathWithoutExt := strings.TrimSuffix(targetFileName, filepath.Ext(targetFileName))
		fmt.Printf("DEBUG: - %s (ID: %s)\n", targetRelativePathWithoutExt, track.ID)

		// Porovnáme relativní cesty (bez přípony) - použijeme case-insensitive porovnání
		// a také zkontrolujeme, zda jedna cesta neobsahuje druhou
		if strings.EqualFold(targetRelativePathWithoutExt, relativePathWithoutExt) ||
			strings.Contains(strings.ToLower(targetRelativePathWithoutExt), strings.ToLower(relativePathWithoutExt)) ||
			strings.Contains(strings.ToLower(relativePathWithoutExt), strings.ToLower(targetRelativePathWithoutExt)) {
			fmt.Printf("MATCH FOUND: Source=%s, Target=%s\n", relativePathWithoutExt, targetRelativePathWithoutExt)
			targetTracks = append(targetTracks, struct {
				ID       string
				FileName string
			}{
				ID:       track.ID,
				FileName: track.FileName,
			})
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(targetTracks) == 0 {
		// Místo vracení chyby jen vypíšeme informaci a vrátíme prázdný seznam
		fmt.Printf(locales.Translate("hotcuesync.warning.notgttracks")+": %s\n", fileName)
		return []struct {
			ID       string
			FileName string
		}{}, nil
	}

	return targetTracks, nil
}

// copyHotCues copies hot cues from the source track to the target track.
func (m *HotCueSyncModule) copyHotCues(sourceID, targetID string) error {
	fmt.Printf("    Copying hot cues from source ID %s to target ID %s\n", sourceID, targetID)

	// Query hot cues for the source track
	rows, err := m.dbMgr.Query(`
		SELECT 
			ID, ContentID, InMsec, InFrame, InMpegFrame, InMpegAbs, OutMsec, OutFrame, OutMpegFrame, 
			OutMpegAbs, Kind, Color, ColorTableIndex, ActiveLoop, Comment, BeatLoopSize, CueMicrosec, 
			InPointSeekInfo, OutPointSeekInfo, ContentUUID, UUID, rb_data_status, rb_local_data_status, 
			rb_local_deleted, rb_local_synced
		FROM djmdCue 
		WHERE ContentID = ?
	`, sourceID)
	if err != nil {
		fmt.Printf("    Error querying hot cues: %v\n", err)
		return fmt.Errorf(locales.Translate("hotcuesync.err.querycues"), err)
	}
	defer rows.Close()

	// Použijeme podědující pro sledování počtu zpracovaných hot cues
	hotCueCount := 0

	// Process each hot cue
	for rows.Next() {
		// Define a struct to hold the hot cue data
		var cue struct {
			ID                   string
			ContentID            string
			InMsec               sql.NullInt64
			InFrame              sql.NullInt64
			InMpegFrame          sql.NullInt64
			InMpegAbs            sql.NullInt64
			OutMsec              sql.NullInt64
			OutFrame             sql.NullInt64
			OutMpegFrame         sql.NullInt64
			OutMpegAbs           sql.NullInt64
			Kind                 sql.NullInt64
			Color                sql.NullInt64
			ColorTableIndex      sql.NullInt64
			ActiveLoop           sql.NullInt64
			Comment              sql.NullString
			BeatLoopSize         sql.NullInt64
			CueMicrosec          sql.NullInt64
			InPointSeekInfo      sql.NullString
			OutPointSeekInfo     sql.NullString
			ContentUUID          sql.NullString
			UUID                 sql.NullString
			rb_data_status       sql.NullInt64
			rb_local_data_status sql.NullInt64
			rb_local_deleted     sql.NullInt64
			rb_local_synced      sql.NullInt64
		}

		// Scan the row into the struct
		err := rows.Scan(
			&cue.ID, &cue.ContentID, &cue.InMsec, &cue.InFrame, &cue.InMpegFrame, &cue.InMpegAbs, &cue.OutMsec, &cue.OutFrame, &cue.OutMpegFrame,
			&cue.OutMpegAbs, &cue.Kind, &cue.Color, &cue.ColorTableIndex, &cue.ActiveLoop, &cue.Comment, &cue.BeatLoopSize,
			&cue.CueMicrosec, &cue.InPointSeekInfo, &cue.OutPointSeekInfo, &cue.ContentUUID, &cue.UUID, &cue.rb_data_status,
			&cue.rb_local_data_status, &cue.rb_local_deleted, &cue.rb_local_synced,
		)
		if err != nil {
			fmt.Printf("    Error scanning hot cue: %v\n", err)
			return fmt.Errorf(locales.Translate("hotcuesync.err.scancues"), err)
		}

		// Zvýšíme podědující
		hotCueCount++

		// Výpis informací o hot cue
		fmt.Printf("    Processing hot cue %d: ID=%s, Kind=%v\n",
			hotCueCount, cue.ID, cue.Kind.Int64)

		// Delete existing hot cues with the same Kind value in the target track
		err = m.dbMgr.Execute(`DELETE FROM djmdCue WHERE ContentID = ? AND Kind = ?`, targetID, cue.Kind)
		if err != nil {
			fmt.Printf("    Error deleting existing hot cue: %v\n", err)
			return fmt.Errorf(locales.Translate("hotcuesync.err.deletecue"), err)
		}

		// Generate a new ID for the hot cue in the target track
		var maxID int64
		err = m.dbMgr.QueryRow("SELECT COALESCE(MAX(CAST(ID AS INTEGER)), 0) FROM djmdCue").Scan(&maxID)
		if err != nil {
			fmt.Printf("    Error getting max ID: %v\n", err)
			return fmt.Errorf(locales.Translate("hotcuesync.err.maxidcheck"), err)
		}
		maxID++
		newID := fmt.Sprintf("%d", maxID)
		fmt.Printf("    Generated new ID: %s for hot cue\n", newID)

		// Get current timestamp for created_at
		currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000 +00:00")

		// Insert the hot cue into the target track
		err = m.dbMgr.Execute(`
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
		`,
			newID, targetID, cue.InMsec, cue.InFrame, cue.InMpegFrame, cue.InMpegAbs, cue.OutMsec, cue.OutFrame, cue.OutMpegFrame,
			cue.OutMpegAbs, cue.Kind, cue.Color, cue.ColorTableIndex, cue.ActiveLoop, cue.Comment, cue.BeatLoopSize,
			cue.CueMicrosec, cue.InPointSeekInfo, cue.OutPointSeekInfo, cue.ContentUUID, cue.UUID, cue.rb_data_status,
			cue.rb_local_data_status, cue.rb_local_deleted, cue.rb_local_synced, currentTime, currentTime,
		)
		if err != nil {
			fmt.Printf("    Error inserting hot cue: %v\n", err)
			return fmt.Errorf(locales.Translate("hotcuesync.err.cueinsert"), err)
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

func (m *HotCueSyncModule) synchronizeHotCues() error {
	// Disable the button during processing
	m.submitBtn.Disable()
	defer func() {
		m.submitBtn.Enable()
	}()

	// Basic validation
	if (m.sourceType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) && m.sourceFolderEntry.Text == "") ||
		(m.targetType.Selected == locales.Translate("hotcuesync.dropdown."+string(SourceTypeFolder)) && m.targetFolderEntry.Text == "") {
		m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("hotcuesync.err.emptypaths")), common.NewErrorContext(m.GetConfigName(), "Empty Paths"), m.Window, m.Status)
		return fmt.Errorf(locales.Translate("hotcuesync.err.emptypaths"))
	}

	// Show a progress dialog
	m.ShowProgressDialog(locales.Translate("hotcuesync.diag.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("hotcuesync.err.panic"), r), common.NewErrorContext(m.GetConfigName(), "Panic"), m.Window, m.Status)
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
			m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("hotcuesync.err.transtart"), err), common.NewErrorContext(m.GetConfigName(), "Transaction Start"), m.Window, m.Status)
			return
		}

		// Ensure database connection is properly closed when done
		defer m.dbMgr.Finalize()

		// Ensure transaction is rolled back on error
		defer func() {
			if err != nil {
				m.dbMgr.RollbackTransaction()
			}
		}()

		// Použijeme podědující pro úspěšné a přeskočené soubory
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
				m.ErrorHandler.HandleError(fmt.Errorf("Error processing track: %v", err), common.NewErrorContext(m.GetConfigName(), "Target Tracks"), m.Window, m.Status)
				m.dbMgr.RollbackTransaction()
				return
			}

			// Pokud nejsou nalezeny žádné cílové skladby, přeskočíme tento soubor
			if len(targetTracks) == 0 {
				skippedCount++
				continue
			}

			// Copy hot cues to each target track
			for _, targetTrack := range targetTracks {
				// Check if operation was cancelled
				if m.IsCancelled() {
					m.CloseProgressDialog()
					m.dbMgr.RollbackTransaction()
					return
				}

				err = m.copyHotCues(sourceTrack.ID, targetTrack.ID)
				if err != nil {
					m.CloseProgressDialog()
					m.ErrorHandler.HandleError(err, common.NewErrorContext(m.GetConfigName(), "Copy HotCues"), m.Window, m.Status)
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
			m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("hotcuesync.err.trancommit"), err), common.NewErrorContext(m.GetConfigName(), "Transaction Commit"), m.Window, m.Status)
			return
		}

		// Zobrazíme souhrnnou zprávu
		summaryMessage := fmt.Sprintf(locales.Translate("hotcuesync.status.completed"), successCount, skippedCount)

		// Aktualizace stavové zprávy
		m.UpdateProgressStatus(1.0, summaryMessage)

		// Mark the progress dialog as completed instead of closing it
		m.CompleteProgressDialog()
	}()

	return nil
}
