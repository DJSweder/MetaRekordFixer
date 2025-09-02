// modules/flacfixer.go

// Package modules provides functionality for different modules in the MetaRekordFixer application.
// Each module handles a specific task related to DJ database management and music file operations.

// This module reads metadata directly from FLAC files and updates the database with the extracted information

package modules

import (
	"context"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
)

// FlacFixerModule handles metadata synchronization between different file formats.
// It implements the standard Module interface and provides functionality for synchronizing
// metadata between MP3 and FLAC files in a specified folder, ensuring consistent metadata
// across different formats of the same tracks.
type FlacFixerModule struct {
	// ModuleBase is the base struct for all modules, which contains the module's window,
	// error handler, and configuration manager.
	*common.ModuleBase
	// dbMgr handles database operations
	dbMgr *common.DBManager
	// sourceFolderEntry is the entry field for source folder path
	sourceFolderEntry *widget.Entry
	// folderSelectionField contains the complete folder selection UI component
	folderSelectionField fyne.CanvasObject
	// recursiveCheck determines if the sync should process subfolders
	recursiveCheck *widget.Check
	// submitBtn triggers the synchronization process
	submitBtn *widget.Button
}

// NewFlacFixerModule creates a new instance of FlacFixerModule.
// It initializes the module with the provided window, configuration manager,
// database manager, and error handler, sets up the UI components, and loads
// any saved configuration.
//
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for saving/loading module settings
//   - dbMgr: Database manager for accessing the DJ database
//   - errorHandler: Error handler for displaying and logging errors
//
// Returns:
//   - A fully initialized FlacFixerModule instance
func NewFlacFixerModule(window fyne.Window, configMgr *common.ConfigManager, dbMgr *common.DBManager, errorHandler *common.ErrorHandler) *FlacFixerModule {
	m := &FlacFixerModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		dbMgr:      dbMgr,
	}

	m.initializeUI()

	// Load typed configuration
	m.LoadCfg()

	return m
}

// GetName returns the localized name of this module.
// This implements the Module interface method.
func (m *FlacFixerModule) GetName() string {
	return locales.Translate("flacfixer.mod.name")
}

// GetConfigName returns the configuration key for this module.
// This key is used to store and retrieve module-specific configuration.
func (m *FlacFixerModule) GetConfigName() string {
	return common.ModuleKeyFlacFixer
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method and provides the visual representation
// of this module in the UI.
func (m *FlacFixerModule) GetIcon() fyne.Resource {
	return theme.HomeIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI
// containing the folder selection field, recursive checkbox, and submit button.
func (m *FlacFixerModule) GetModuleContent() fyne.CanvasObject {
	// Create form with folder selection field
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("flacfixer.label.source"), Widget: m.folderSelectionField},
		},
	}

	// Create content container with form and additional widgets
	contentContainer := container.NewVBox(
		form,
		m.recursiveCheck,
	)

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("flacfixer.label.info")),
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

// GetContent returns the module's main UI content.
// This method returns the complete module layout with status messages container.
//
// Note: This module intentionally does not implement database availability checks
// as it operates without an active database connection until backup is created.
// The module also does not implement enable/disable logic for its controls
// as it needs to remain functional even when database is not available.
func (m *FlacFixerModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadCfg loads typed configuration and updates UI elements
func (m *FlacFixerModule) LoadCfg() {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load typed config from ConfigManager
	config, err := m.ConfigMgr.GetModuleCfg(common.ModuleKeyFlacFixer, m.GetConfigName())
	if err != nil {
		// This should not happen with the updated GetModuleCfg(), but handle gracefully
		return
	}

	// Cast to FlacFixer specific config
	if cfg, ok := config.(common.FlacFixerCfg); ok {
		// Update UI elements with loaded values
		m.sourceFolderEntry.SetText(cfg.SourceFolder.Value)
		m.recursiveCheck.SetChecked(cfg.Recursive.Value == "true")
	}
}

// SaveCfg saves current UI state to typed configuration
func (m *FlacFixerModule) SaveCfg() {
	if m.IsLoadingConfig {
		return // Safeguard: no save if config is being loaded
	}

	// Get default configuration with all field definitions
	cfg := common.GetDefaultFlacFixerCfg()

	// Update only the values from current UI state
	cfg.SourceFolder.Value = common.NormalizePath(m.sourceFolderEntry.Text)
	cfg.Recursive.Value = fmt.Sprintf("%t", m.recursiveCheck.Checked)

	// Save typed config via ConfigManager
	m.ConfigMgr.SaveModuleCfg(common.ModuleKeyFlacFixer, m.GetConfigName(), cfg)
}

// initializeUI sets up the user interface components.
// It creates and configures the entry fields, checkboxes, and buttons,
// and sets up their event handlers to respond to user interactions.
func (m *FlacFixerModule) initializeUI() {
	// Initialize folder selection field using standardized function
	m.folderSelectionField = common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		nil,
		m.CreateChangeHandler(func() {
			m.SaveCfg()
		}),
	)
	// Extract the entry widget from the container for direct access
	if container, ok := m.folderSelectionField.(*fyne.Container); ok && len(container.Objects) > 0 {
		if entry, ok := container.Objects[0].(*widget.Entry); ok {
			m.sourceFolderEntry = entry
		}
	}

	// Initialize recursive checkbox using standardized function
	m.recursiveCheck = common.CreateCheckbox(locales.Translate("flacfixer.chkbox.recursive"), func(checked bool) {
		m.SaveCfg()
	})

	// Initialize sync button
	m.submitBtn = common.CreateSubmitButton(locales.Translate("flacfixer.button.sync"), func() {
		go m.Start()
	},
	)
}

// Start performs the necessary steps before starting the main process.
// It saves the configuration, validates the inputs, informs the user, displays a dialog with a progress bar
// and starts the main process.
// Input validation also includes a test of the connection to the database and creating a backup of it.
//
// This method is called when the user clicks the submit button and runs the validation
// before launching the actual synchronization process in a goroutine.
func (m *FlacFixerModule) Start() {

	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, m.dbMgr, m.ErrorHandler)
	if err := validator.Validate("start"); err != nil {
		return
	}

	sourcePath := common.NormalizePath(m.sourceFolderEntry.Text)

	// Prepare cancelable context and show progress dialog with cancel support
	ctx, cancel := context.WithCancel(context.Background())
	// Store cancel locally via closure; when Stop is pressed, cancel context and show stopping info
	m.ShowProgressDialog(
		locales.Translate("flacfixer.dialog.header"),
		func() {
			cancel()
			sourcePath := common.NormalizePath(m.sourceFolderEntry.Text)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						context := &common.ErrorContext{
							Module:      m.GetName(),
							Operation:   "Metadata Sync",
							Severity:    common.SeverityCritical,
							Recoverable: false,
						}
						m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
						m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
					}
				}()

				// Process metadata copy with cancellation context
				m.processFlacFixer(ctx, sourcePath)
			}()
		},
	)

	// Start processing in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.CloseProgressDialog()
				context := &common.ErrorContext{
					Module:      m.GetName(),
					Operation:   "Metadata Sync",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(fmt.Errorf("%v", r), context)
				m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
			}
		}()

		// Process metadata copy with cancellation context
		m.processFlacFixer(ctx, sourcePath)
	}()
}

// processFlacFixer handles the actual metadata processing from FLAC files.
// It reads metadata directly from FLAC files in the specified folder and updates
// the database with the extracted information, managing progress and status updates.
//
// The method performs the following steps:
// 1. Finds all FLAC files in the specified folder (recursively if enabled)
// 2. Reads metadata directly from each FLAC file
// 3. Updates the database with artist, album, and track metadata
// 4. Updates progress and handles cancellation throughout the process
//
// Parameters:
//   - ctx: The context for cancellation
//   - sourcePath: The folder path to process for metadata extraction
func (m *FlacFixerModule) processFlacFixer(ctx context.Context, sourcePath string) {
	defer m.dbMgr.Finalize()

	// Normalize paths
	sourcePath = common.NormalizePath(sourcePath)

	// Do not show initial generic progress; validator already provided start status,
	// and specific progress will appear as soon as counts are known.

	// Process all FLAC files in the folder
	summary, err := common.ProcessFolderMetadata(
		ctx,
		m.dbMgr,
		sourcePath,
		m.recursiveCheck.Checked,
		func(total int) {
			// Inform about files found
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.filesfound"), total))
		},
		func(progress float64, updated int, total int) {
			// Update progress with localized status
			status := fmt.Sprintf(locales.Translate("common.status.progress"), updated, total)
			m.UpdateProgressStatus(progress, status)

			// Check for cancellation during processing
			if m.IsCancelled() {
				return
			}
		},
	)

	if err != nil {
		// Handle cancellation explicitly
		if errors.Is(err, common.ErrCancelled) {
			m.HandleProcessCancellation("common.status.stopped", summary.Updated, summary.Total)
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}
		m.CloseProgressDialog()
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "FLAC Metadata Processing",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Check if cancelled after processing
	if m.IsCancelled() {
		m.HandleProcessCancellation("common.status.stopped", summary.Updated, summary.Total)
		common.UpdateButtonToCompleted(m.submitBtn)
		return
	}

	// Add completion status messages
	if summary.Total == 0 {
		m.AddErrorMessage(locales.Translate("common.err.nofiles"))
		m.UpdateProgressStatus(1.0, locales.Translate("common.err.nofiles"))
	} else {
		finalMsg := fmt.Sprintf(
			locales.Translate("flacfixer.status.summary"),
			summary.Total,
			summary.Updated,
			summary.NoChange,
			summary.SkippedZero,
			summary.MetadataErrs,
			summary.DbMisses,
			summary.DbUpdateErrs,
			summary.SkippedDirs,
		)
		m.AddInfoMessage(finalMsg)
		m.CompleteProcessing(finalMsg)
	}

	// Mark the progress dialog as completed and update button
	m.CompleteProgressDialog()
	common.UpdateButtonToCompleted(m.submitBtn)
}
