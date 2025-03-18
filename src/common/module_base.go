// common/module_base.go

package common

import (
	"database/sql"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// PlaylistItem represents a playlist item from Rekordbox database
type PlaylistItem struct {
	ID       string
	Name     string
	ParentID sql.NullString
	Path     string
}

// Module defines the interface that all modules must implement
type Module interface {
	GetName() string
	GetConfigName() string
	GetIcon() fyne.Resource
	GetContent() fyne.CanvasObject
	LoadConfig(config ModuleConfig)
	SaveConfig() ModuleConfig
}

// ModuleBase provides common functionality for all modules
type ModuleBase struct {
	Window          fyne.Window
	Content         fyne.CanvasObject
	ConfigMgr       *ConfigManager
	Progress        *widget.ProgressBar
	Status          *widget.Label
	ProgressDialog  *ProgressDialog
	IsLoadingConfig bool
	mutex           sync.Mutex
	isCancelled     bool
	ErrorHandler    *ErrorHandler
}

// NewModuleBase initializes a new ModuleBase
func NewModuleBase(window fyne.Window, configMgr *ConfigManager, errorHandler *ErrorHandler) *ModuleBase {
	if errorHandler == nil {
		errorHandler = NewErrorHandler(nil) // Default ErrorHandler if none provided
	}

	base := &ModuleBase{
		Window:       window,
		ConfigMgr:    configMgr,
		ErrorHandler: errorHandler,
	}
	base.initBaseComponents()
	// Odstranění automatického volání LoadModuleConfig, bude voláno až po inicializaci modulu
	return base
}

// initBaseComponents initializes common UI components
func (m *ModuleBase) initBaseComponents() {
	m.Progress = widget.NewProgressBar()
	m.Status = widget.NewLabel("")
	m.Status.Alignment = fyne.TextAlignCenter
}

// GetName returns an empty name, should be overridden in modules
func (m *ModuleBase) GetName() string {
	return ""
}

// GetConfigName returns an unknown module name, should be overridden
func (m *ModuleBase) GetConfigName() string {
	return "unknown_module"
}

// GetIcon returns a default icon, should be overridden in modules
func (m *ModuleBase) GetIcon() fyne.Resource {
	return nil
}

// GetContent returns the stored content, should be overridden in modules
func (m *ModuleBase) GetContent() fyne.CanvasObject {
	return m.Content
}

// LoadConfig is a placeholder for configuration loading
func (m *ModuleBase) LoadConfig(cfg ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()
}

// SaveConfig ensures that a valid `ModuleConfig` is returned
func (m *ModuleBase) SaveConfig() ModuleConfig {
	if m.IsLoadingConfig {
		return NewModuleConfig()
	}
	return NewModuleConfig()
}

// ShowProgressDialog displays a progress dialog with stop button
func (m *ModuleBase) ShowProgressDialog(title string) {
	// Reset cancellation flag
	m.mutex.Lock()
	m.isCancelled = false
	m.mutex.Unlock()

	// Create cancel handler function
	cancelHandler := func() {
		m.mutex.Lock()
		m.isCancelled = true
		m.mutex.Unlock()
	}

	// Create and show progress dialog
	m.ProgressDialog = NewProgressDialog(m.Window, title, "", cancelHandler)
	m.ProgressDialog.Show()
}

// UpdateProgressStatus updates the progress bar and status text
func (m *ModuleBase) UpdateProgressStatus(progress float64, statusText string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Progress.SetValue(progress)
	m.Status.SetText(statusText)

	if m.ProgressDialog != nil {
		m.ProgressDialog.UpdateProgress(progress)
		m.ProgressDialog.UpdateStatus(statusText)
	}
}

// CloseProgressDialog hides the progress dialog
func (m *ModuleBase) CloseProgressDialog() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ProgressDialog != nil {
		m.ProgressDialog.Hide()
		m.ProgressDialog = nil
	}
}

// CompleteProgressDialog marks the progress dialog as completed and changes the stop button to OK
func (m *ModuleBase) CompleteProgressDialog() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ProgressDialog != nil {
		m.ProgressDialog.MarkCompleted()
	}
}

// IsCancelled returns true if the current operation has been cancelled by the user
func (m *ModuleBase) IsCancelled() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.isCancelled
}

// ShowError displays an error message in a dialog using ErrorHandler
func (m *ModuleBase) ShowError(err error) {
	if m.ErrorHandler != nil {
		// Create error context with module name
		context := NewErrorContext(m.GetConfigName(), "")
		context.Severity = ErrorWarning
		m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
	} else {
		// Fallback to simple error dialog if ErrorHandler is not available
		ShowError(err, m.Window)
	}
}

// CreateChangeHandler prevents unwanted save triggers during config loading
func (m *ModuleBase) CreateChangeHandler(handler func()) func(string) {
	return func(s string) {
		if !m.IsLoadingConfig {
			handler()
		}
	}
}

// CreateBoolChangeHandler handles boolean input changes safely
func (m *ModuleBase) CreateBoolChangeHandler(handler func()) func(bool) {
	return func(b bool) {
		if !m.IsLoadingConfig {
			handler()
		}
	}
}

// CreateSelectionChangeHandler handles selection input changes safely
func (m *ModuleBase) CreateSelectionChangeHandler(handler func()) func(string) {
	return func(s string) {
		if !m.IsLoadingConfig {
			handler()
		}
	}
}

// LoadFolderEntries loads folder entries from the configuration
func LoadFolderEntries(cfg ModuleConfig, key string) []*widget.Entry {
	entries := []*widget.Entry{}
	folders := strings.Split(cfg.Get(key, ""), "|")
	for _, folder := range folders {
		if folder != "" {
			entry := widget.NewEntry()
			entry.SetText(folder)
			entries = append(entries, entry)
		}
	}
	return entries
}

// SaveFolderEntries saves folder entries into the configuration
func SaveFolderEntries(cfg ModuleConfig, key string, entries []*widget.Entry) {
	folders := []string{}
	for _, entry := range entries {
		if entry.Text != "" {
			folders = append(folders, entry.Text)
		}
	}
	cfg.Set(key, strings.Join(folders, "|"))
}
