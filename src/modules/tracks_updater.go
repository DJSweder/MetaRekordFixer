package modules

import (
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// TracksUpdater handles updating track file paths and formats in the database
// Implements the standard Module interface
type TracksUpdater struct {
	*common.ModuleBase
	dbMgr          *common.DBManager
	playlistSelect *widget.Select
	folderPath     *widget.Entry
	folderSelect   *widget.Button
	submitBtn      *widget.Button
	playlists      []common.PlaylistItem
}

// NewTracksUpdater creates a new instance of TracksUpdater.
func NewTracksUpdater(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *TracksUpdater {
	m := &TracksUpdater{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	// Inicializace proměnných před voláním initializeUI
	m.folderPath = widget.NewEntry()

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
	return theme.MediaMusicIcon()
}

// GetContent returns the module's main UI content.
func (m *TracksUpdater) GetContent() fyne.CanvasObject {
	// Create form with playlist selector and folder selection field
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("updater.data.source"), Widget: m.playlistSelect},
			{Text: locales.Translate("updater.folder.newfiles"), Widget: container.NewBorder(nil, nil, nil, m.folderSelect, m.folderPath)},
		},
	}

	// Create additional widgets array
	additionalWidgets := []fyne.CanvasObject{
		m.Status,
	}

	// Create content container with form and additional widgets
	contentContainer := container.NewVBox(
		form,
	)

	// Add additional widgets to content container
	for _, widget := range additionalWidgets {
		contentContainer.Add(widget)
	}

	// Create final layout using standardized module layout
	return common.CreateStandardModuleLayout(
		locales.Translate("updater.mod.descr"),
		contentContainer,
		m.submitBtn,
	)
}

// LoadConfig applies the configuration to the UI components.
func (m *TracksUpdater) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Kontrola, zda konfigurace má inicializované Extra pole
	if cfg.Extra == nil {
		return
	}

	if folder, ok := cfg.Extra["folder"]; ok && folder != "" {
		m.folderPath.SetText(folder)
	}

	if playlistID, ok := cfg.Extra["playlist_id"]; ok && playlistID != "" {
		// Find and set playlist by ID
		for i, playlist := range m.playlists {
			if playlist.ID == playlistID {
				if i < len(m.playlistSelect.Options) {
					m.playlistSelect.SetSelected(m.playlistSelect.Options[i])
				}
				break
			}
		}
	} else if len(m.playlistSelect.Options) > 0 {
		m.playlistSelect.SetSelected(m.playlistSelect.Options[0])
	}
}

// SaveConfig reads UI state and saves it into a new ModuleConfig.
func (m *TracksUpdater) SaveConfig() common.ModuleConfig {
	if m.IsLoadingConfig {
		return common.NewModuleConfig() // Safeguard: no save if config is being loaded
	}

	cfg := common.NewModuleConfig()

	// Save folder path using NormalizePath which now handles empty strings correctly
	cfg.Set("folder", common.NormalizePath(m.folderPath.Text))

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

// initializeUI sets up the user interface of the module.
func (m *TracksUpdater) initializeUI() {
	// Initialize playlist selector
	m.playlistSelect = widget.NewSelect([]string{}, m.CreateSelectionChangeHandler(func() {
		m.SaveConfig()
	}))

	// Initialize folder path entry
	m.folderPath.OnChanged = m.CreateChangeHandler(func() {
		m.SaveConfig()
	})

	// Create folder selection field using standardized function
	folderSelectionField := common.CreateFolderSelectionField(
		locales.Translate("updater.folder.newfiles"),
		m.folderPath,
		func(path string) {
			m.folderPath.SetText(path)
			m.SaveConfig()
		},
	)

	// Store the button reference for backward compatibility
	m.folderSelect = folderSelectionField.(*fyne.Container).Objects[1].(*widget.Button)

	// Create submit button using standardized function
	m.submitBtn = common.CreateSubmitButton(
		locales.Translate("updater.button.libupd"),
		func() {
			go m.Start()
		},
	)

	// Load playlists from database
	if err := m.loadPlaylists(); err != nil {
		m.Status.SetText(fmt.Sprintf("%s: %v", locales.Translate("updater.err.dbread"), err))
	}
}

// addStatus adds a status message to the status label.
func (m *TracksUpdater) addStatus(message string, replace bool) {
	lines := strings.Split(m.Status.Text, "\n")
	if currentText := m.Status.Text; currentText == "" {
		m.Status.SetText(message)
	} else if replace && len(lines) > 0 {
		// Replace last line only
		lines[len(lines)-1] = message
		m.Status.SetText(strings.Join(lines, "\n"))
	} else {
		m.Status.SetText(currentText + "\n" + message)
	}
	if !replace {
		time.Sleep(100 * time.Millisecond) // Add small delay for non-replacing messages
		m.Window.Canvas().Refresh(m.Status)
	}
}

func (m *TracksUpdater) Start() {
	// Save configuration before starting the process
	m.SaveConfig()

	// Disable the button during processing
	m.submitBtn.Disable()
	defer func() {
		// Enable the button and set success icon after completion
		m.submitBtn.Enable()
		m.submitBtn.SetIcon(theme.ConfirmIcon())
	}()

	m.Status.SetText("") // Clear previous status
	m.addStatus(locales.Translate("updater.tracks.starting"), false)

	// Show a progress dialog
	m.ShowProgressDialog(locales.Translate("updater.diag.header"))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// In case of panic
				m.CloseProgressDialog()
				m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("updater.err.panic"), r), common.NewErrorContext(m.GetConfigName(), "Panic"), m.Window, m.Status)
			}
		}()
		defer func() {
			if err := m.dbMgr.Finalize(); err != nil {
				m.ErrorHandler.HandleError(fmt.Errorf(locales.Translate("updater.err.finalize"), err),
					common.NewErrorContext(m.GetConfigName(), "Database Finalize"), m.Window, m.Status)
			}
		}()

		// Backup database
		m.UpdateProgressStatus(0.1, locales.Translate("updater.db.backup"))
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

		// Database connection
		m.UpdateProgressStatus(0.2, locales.Translate("updater.db.conn"))
		err = m.dbMgr.Connect()
		if err != nil {
			m.CloseProgressDialog()
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Database Connection")
			context.Severity = common.ErrorWarning
			m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
			return
		}

		// Get selected playlist
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
			m.addStatus(locales.Translate("updater.err.noplaylist"), false)
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Get tracks from playlist
		m.UpdateProgressStatus(0.4, locales.Translate("updater.tracks.gettracks"))
		rows, err := m.dbMgr.Query(`
        SELECT c.ID, c.FileNameL
        FROM djmdContent c
        JOIN djmdSongPlaylist sp ON c.ID = sp.ContentID
        WHERE sp.PlaylistID = ?
    `, selectedPlaylist)
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
				// Create error context with module name and operation
				context := common.NewErrorContext(m.GetConfigName(), "Database Scan")
				context.Severity = common.ErrorWarning
				m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
				return
			}
			tracks = append(tracks, t)
		}

		// Report playlist track count
		m.UpdateProgressStatus(0.5, fmt.Sprintf(locales.Translate("updater.tracks.playlistcount"), len(tracks)))

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Get all files in target folder
		m.UpdateProgressStatus(0.6, locales.Translate("updater.tracks.scanfolder"))
		_, err = filepath.Glob(filepath.Join(m.folderPath.Text, "*.*"))
		if err != nil {
			m.CloseProgressDialog()
			// Create error context with module name and operation
			context := common.NewErrorContext(m.GetConfigName(), "Filepath Glob")
			context.Severity = common.ErrorWarning
			m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
			return
		}

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Count matching files and non-matching files
		m.UpdateProgressStatus(0.7, locales.Translate("updater.tracks.matching"))
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
			newFiles, err := filepath.Glob(filepath.Join(m.folderPath.Text, baseName+".*"))
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

		// Check if operation was cancelled
		if m.IsCancelled() {
			m.CloseProgressDialog()
			return
		}

		// Process tracks
		m.UpdateProgressStatus(0.8, locales.Translate("updater.tracks.updating"))
		updateCount := 0
		for i, updateTrack := range updateTracks {
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
				// Create error context with module name and operation
				context := common.NewErrorContext(m.GetConfigName(), "Database Update")
				context.Severity = common.ErrorWarning
				m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
				return
			}

			updateCount++
			progress := 0.8 + (float64(i+1) / float64(len(updateTracks)) * 0.2)
			m.UpdateProgressStatus(progress, fmt.Sprintf(locales.Translate("updater.status.process"), i+1, len(updateTracks)))

			// Check if operation was cancelled
			if m.IsCancelled() {
				m.CloseProgressDialog()
				return
			}
		}

		// Update progress and status
		m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("updater.status.completed"), updateCount))

		// Mark the progress dialog as completed instead of closing it
		m.CompleteProgressDialog()
	}()
}

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
		return fmt.Errorf("%s: %w", locales.Translate("updater.err.dbopen"), err)
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
		return fmt.Errorf("%s: %w", locales.Translate("updater.err.dbread"), err)
	}
	defer rows.Close()

	m.playlists = make([]common.PlaylistItem, 0)
	var playlistPaths []string
	for rows.Next() {
		var p common.PlaylistItem
		if err := rows.Scan(&p.ID, &p.Name, &p.ParentID, &p.Path); err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("updater.err.dbread"), err)
		}
		m.playlists = append(m.playlists, p)
		playlistPaths = append(playlistPaths, p.Path)
	}

	m.playlistSelect.Options = playlistPaths
	return nil
}
