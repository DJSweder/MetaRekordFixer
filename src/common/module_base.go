// common/module_base.go
// Package common implements shared functionality used across the MetaRekordFixer application.
// This file contains the base module implementation and related interfaces.

package common

import (
	"MetaRekordFixer/locales"
	"database/sql"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// PlaylistItem represents a playlist item from Rekordbox database.
// It contains the playlist's identifier, name, parent ID, and full path.
type PlaylistItem struct {
	ID       string         // Unique identifier of the playlist
	Name     string         // Display name of the playlist
	ParentID sql.NullString // Parent playlist ID (nullable for root playlists)
	Path     string         // Full path of the playlist including parent folders
}

// Module defines the interface that all modules must implement.
// This ensures consistent behavior and functionality across all application modules.
type Module interface {
	// GetName returns the localized display name of the module
	GetName() string
	// GetConfigName returns the identifier used for configuration storage
	GetConfigName() string
	// GetIcon returns the module's icon for display in the UI
	GetIcon() fyne.Resource
	// GetContent returns the complete UI of the module including status messages
	GetContent() fyne.CanvasObject
	// LoadCfg loads module configuration using the new typed system
	LoadCfg()
	// SaveCfg saves the current module configuration using the new typed system
	SaveCfg()
	// GetDatabaseRequirements returns the database access requirements for this module
	GetDatabaseRequirements() DatabaseRequirements
	// SetDatabaseRequirements configures the database access requirements for this module
	SetDatabaseRequirements(needs bool, immediate bool)
}


// ModuleBase provides common functionality for all modules.
// It implements shared behavior and serves as a base struct for specific module implementations.
type ModuleBase struct {
	Window           fyne.Window                // Main application window reference
	Content          fyne.CanvasObject          // Module's UI content
	ConfigMgr        *ConfigManager             // Configuration manager for loading/saving settings
	Progress         *widget.ProgressBar        // Progress indicator for operations
	Status           *widget.Label              // Status text display
	ProgressDialog   *ProgressDialog            // Dialog showing progress with cancel option
	IsLoadingConfig  bool                       // Flag to prevent saving during config loading
	mutex            sync.Mutex                 // Mutex for thread-safe operations
	isCancelled      bool                       // Flag indicating if current operation was cancelled
	ErrorHandler     *ErrorHandler              // Central error handling component
	Logger           *Logger                    // Logger for recording events
	StatusMessages   *StatusMessagesContainer   // Container for status messages
	dbRequirements   DatabaseRequirements       // Database access requirements
	Cfg              interface{}                // Typed configuration field for type-safe configuration
}

// DatabaseRequirements defines how a module uses the database.
// This helps with lazy loading of database connections and proper initialization.
type DatabaseRequirements struct {
	// NeedsDatabase indicates if the module requires database access
	NeedsDatabase bool
	// NeedsImmediateAccess indicates if database access is required during initialization
	NeedsImmediateAccess bool
}

// NewModuleBase creates a new ModuleBase instance with the provided window, configuration manager, and error handler.
// It initializes all necessary components including status messages container and database requirements.
// This is the recommended way to create a new module base for all application modules.
func NewModuleBase(window fyne.Window, configMgr *ConfigManager, errorHandler *ErrorHandler) *ModuleBase {
	if errorHandler == nil {
		panic("ErrorHandler cannot be nil")
	}

	base := &ModuleBase{
		Window:       window,
		ConfigMgr:    configMgr,
		ErrorHandler: errorHandler,
		Logger:       errorHandler.GetLogger(),
	}
	base.initBaseComponents()

	return base
}

// initBaseComponents initializes common UI components used by all modules.
// This includes progress bar, status label, and status messages container.
// Called automatically by NewModuleBase to ensure proper initialization.
func (m *ModuleBase) initBaseComponents() {
	m.Progress = widget.NewProgressBar()
	m.Status = widget.NewLabel("")
	m.Status.Alignment = fyne.TextAlignCenter
	m.StatusMessages = NewStatusMessagesContainer()
}

// GetModuleContent returns the module's content without status messages.
// This method should be overridden by specific module implementations to return their unique UI content.
// It is used by the CreateModuleLayoutWithStatusMessages method to create the full layout with status messages.
// The default implementation returns a placeholder message indicating that the module content is not implemented.
func (m *ModuleBase) GetModuleContent() fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(locales.Translate("common.err.modulecontent")))
}

// CreateModuleLayoutWithStatusMessages creates a layout with module content and status messages.
// The module content is placed at the top and status messages at the bottom in a border container.
// This method is used by module implementations to create their complete UI layout including status messages.
// It is typically called from the GetContent method of specific module implementations.
//
// Parameters:
//   - moduleContent: The specific module content to be displayed in the main area
//
// Returns:
//   - A complete layout with the module content and status messages container properly positioned
func (m *ModuleBase) CreateModuleLayoutWithStatusMessages(moduleContent fyne.CanvasObject) fyne.CanvasObject {
	// Create a container for the module content
	mainContent := container.NewVBox(moduleContent)

	// Create a container for status messages
	statusMessagesContainer := m.GetStatusMessagesContainer().scroll

	// Use BorderLayout to make status messages fill the remaining space
	// The top part (mainContent) has fixed size based on its content
	// The bottom part (statusMessagesContainer) will expand to fill remaining space
	return container.New(
		layout.NewBorderLayout(mainContent, nil, nil, nil),
		mainContent,
		statusMessagesContainer,
	)
}

// GetName returns the display name of the module.
// This is a placeholder implementation that should be overridden by specific modules.
// The module name is typically used in the UI for tab labels and other identifying purposes.
// Returns "Unknown Module" as a default value.
func (m *ModuleBase) GetName() string {
	return ""
}

// GetConfigName returns the configuration identifier for this module.
// This is a placeholder implementation that should be overridden by specific modules.
// The config name is used as a key for storing and retrieving module-specific configuration.
// Returns "unknown" as a default value.
func (m *ModuleBase) GetConfigName() string {
	return "unknown_module"
}

// GetIcon returns the icon resource for this module.
// This is a placeholder implementation that should be overridden by specific modules.
// The icon is typically displayed in the UI next to the module name in tabs or menus.
// Returns nil as a default value, which will cause the UI to use a default icon.
func (m *ModuleBase) GetIcon() fyne.Resource {
	return nil
}

// LoadCfg loads module configuration using the new typed system.
// This is a placeholder implementation that should be overridden by specific modules.
// Modules should implement this method to restore their state from saved configuration.
// When implementing this method, modules should set IsLoadingConfig to true at the start
// and false at the end to prevent unwanted save operations during loading.
func (m *ModuleBase) LoadCfg() {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()
}

// SaveCfg saves the current module configuration using the new typed system.
// This is a placeholder implementation that should be overridden by specific modules.
// Modules should implement this method to store their current state for later restoration.
// The base implementation includes a safety check to prevent saving during configuration loading.
func (m *ModuleBase) SaveCfg() {
	if m.IsLoadingConfig {
		return
	}
	// Placeholder - specific modules should override this method
}

// ShowProgressDialog displays a progress dialog with stop button and optional cancel callback.
// This creates a modal dialog with a progress bar and a stop button that allows the user
// to cancel the current operation. The dialog is shown immediately.
//
// Parameters:
//   - title: The title to display in the dialog header
//   - onCancel: Optional callback functions to execute when the user cancels the operation
//
// The dialog will remain visible until CloseProgressDialog or CompleteProgressDialog is called.
func (m *ModuleBase) ShowProgressDialog(title string, onCancel ...func()) {
	// Reset cancellation flag
	m.mutex.Lock()
	m.isCancelled = false
	m.mutex.Unlock()

	// Create cancel handler function
	var cancelHandler func()
	if len(onCancel) > 0 && onCancel[0] != nil {
		cancelHandler = func() {
			m.mutex.Lock()
			m.isCancelled = true
			m.mutex.Unlock()
			onCancel[0]()
		}
	} else {
		cancelHandler = func() {
			m.mutex.Lock()
			m.isCancelled = true
			m.mutex.Unlock()
		}
	}

	// Create and show progress dialog
	m.mutex.Lock()
	m.ProgressDialog = NewProgressDialog(m.Window, title, "", cancelHandler)
	m.mutex.Unlock()
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

// CloseProgressDialog hides and destroys the progress dialog.
// This should be called when an operation completes or is cancelled.
// After calling this method, the progress dialog will no longer be visible
// and its resources will be released.
//
// This method is safe to call from goroutines as it uses mutex protection.
// If the progress dialog is not currently shown, this method has no effect.
func (m *ModuleBase) CloseProgressDialog() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ProgressDialog != nil {
		m.ProgressDialog.Hide()
		m.ProgressDialog = nil
	}
}

// CompleteProgressDialog marks the progress dialog as completed and changes the stop button to OK.
// This should be called when an operation successfully completes but you want to keep
// the dialog visible to show the final status to the user.
//
// The dialog will remain visible until the user clicks the OK button or CloseProgressDialog is called.
// This provides a better user experience than immediately closing the dialog as it allows
// the user to see that the operation completed successfully.
//
// This method is safe to call from goroutines as it uses mutex protection.
// If the progress dialog is not currently shown, this method has no effect.
func (m *ModuleBase) CompleteProgressDialog() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ProgressDialog != nil {
		m.ProgressDialog.MarkCompleted()
	}
}

// IsCancelled returns true if the current operation has been cancelled by the user.
// This method is used to check if the user has clicked the cancel button in the progress dialog.
// It is safe to call from goroutines as it uses mutex protection.
//
// Returns:
//   - true if the operation has been cancelled, false otherwise
func (m *ModuleBase) IsCancelled() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.isCancelled
}

// ShowError displays a simple error dialog with the provided error message.
// This is a convenience method for displaying errors without additional context information.
//
// Parameters:
//   - err: The error to display
//
// If the ErrorHandler is not initialized, this method will silently return without displaying the error.
func (m *ModuleBase) ShowError(err error) {
	if m.ErrorHandler == nil {
		return
	}

	m.ErrorHandler.ShowError(err)
}

// AddInfoMessage adds an information message to the status messages container and logs it.
// This method is used to display non-critical information to the user and record it in the log.
//
// Parameters:
//   - message: The information message to display and log
//
// The message will be displayed with an information icon in the status messages area
// and will be logged with INFO level if a logger is available.
func (m *ModuleBase) AddInfoMessage(message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.StatusMessages != nil {
		m.StatusMessages.AddMessage(MessageInfo, message)
	}
	if m.Logger != nil {
		m.Logger.Info("%s", message)
	}
}

// AddWarningMessage adds a warning message to the status messages container and logs it.
// This method is used to display important but non-critical warnings to the user and record them in the log.
//
// Parameters:
//   - message: The warning message to display and log
//
// The message will be displayed with a warning icon in the status messages area
// and will be logged with WARNING level if a logger is available.
func (m *ModuleBase) AddWarningMessage(message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.StatusMessages != nil {
		m.StatusMessages.AddMessage(MessageWarning, message)
	}
	if m.Logger != nil {
		m.Logger.Warning("%s", message)
	}
}

// AddErrorMessage adds an error message to the status messages container and logs it.
// This method is used to display critical error information to the user and record it in the log.
//
// Parameters:
//   - message: The error message to display and log
//
// The message will be displayed with an error icon in the status messages area
// and will be logged with ERROR level if a logger is available.
func (m *ModuleBase) AddErrorMessage(message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.StatusMessages != nil {
		m.StatusMessages.AddMessage(MessageError, message)
	}
	if m.Logger != nil {
		m.Logger.Error("%s", message)
	}
}

// ClearStatusMessages clears all status messages from the status messages container.
// This method is typically called when starting a new operation or resetting the module state.
//
// If the status messages container is not initialized, this method has no effect.
func (m *ModuleBase) ClearStatusMessages() {
	if m.StatusMessages != nil {
		m.StatusMessages.ClearMessages()
	}
}

// GetStatusMessagesContainer returns the status messages container.
// If the container doesn't exist, it creates a new one, ensuring that status messages
// can always be added without checking for nil.
//
// Returns:
//   - A pointer to the StatusMessagesContainer for this module
func (m *ModuleBase) GetStatusMessagesContainer() *StatusMessagesContainer {
	// Make sure StatusMessages is initialized
	if m.StatusMessages == nil {
		m.StatusMessages = NewStatusMessagesContainer()
	}

	// Return the status messages container
	return m.StatusMessages
}

// CreateChangeHandler creates a wrapper function that only calls the provided handler
// when the module is not loading configuration. This prevents unwanted save operations
// during configuration loading.
//
// Parameters:
//   - handler: The function to call when a change occurs and config is not loading
//
// Returns:
//   - A function that takes a string parameter and conditionally calls the handler
func (m *ModuleBase) CreateChangeHandler(handler func()) func(string) {
	return func(s string) {
		if !m.IsLoadingConfig {
			handler()
		}
	}
}

// CreateBoolChangeHandler creates a wrapper function for boolean input changes.
// Similar to CreateChangeHandler, it prevents handler execution during config loading.
//
// Parameters:
//   - handler: The function to call when a boolean value changes and config is not loading
//
// Returns:
//   - A function that takes a bool parameter and conditionally calls the handler
func (m *ModuleBase) CreateBoolChangeHandler(handler func()) func(bool) {
	return func(b bool) {
		if !m.IsLoadingConfig {
			handler()
		}
	}
}

// CreateSelectionChangeHandler creates a wrapper function for selection input changes.
// Similar to CreateChangeHandler, it prevents handler execution during config loading.
//
// Parameters:
//   - handler: The function to call when a selection changes and config is not loading
//
// Returns:
//   - A function that takes a string parameter and conditionally calls the handler
func (m *ModuleBase) CreateSelectionChangeHandler(handler func()) func(string) {
	return func(s string) {
		if !m.IsLoadingConfig {
			handler()
		}
	}
}

// SetDatabaseRequirements sets the database requirements for this module.
// This configures whether the module needs database access and if that access
// should be established immediately during initialization.
//
// Parameters:
//   - needs: Whether the module requires database access at all
//   - immediate: Whether database access is needed during initialization
func (m *ModuleBase) SetDatabaseRequirements(needs bool, immediate bool) {
	m.dbRequirements = DatabaseRequirements{
		NeedsDatabase:        needs,
		NeedsImmediateAccess: immediate,
	}
}

// GetDatabaseRequirements returns the database requirements for this module.
// This is used by the application to determine if and when to establish
// database connections for this module.
//
// Returns:
//   - A DatabaseRequirements struct indicating the module's database needs
func (m *ModuleBase) GetDatabaseRequirements() DatabaseRequirements {
	return m.dbRequirements
}

// HandleProcessCancellation handles the standard process cancellation flow.
// This method updates the UI to indicate that an operation was cancelled,
// showing a localized message and completing the progress dialog.
//
// Parameters:
//   - message: The localization key for the status message to display
//   - params: Optional parameters for message formatting
func (m *ModuleBase) HandleProcessCancellation(message string, params ...interface{}) {
	// Update progress dialog status
	stoppedMessage := fmt.Sprintf(locales.Translate(message), params...)
	m.UpdateProgressStatus(1.0, stoppedMessage)
	m.AddInfoMessage(stoppedMessage)

	// Complete progress dialog and update UI
	m.CompleteProgressDialog()
}
