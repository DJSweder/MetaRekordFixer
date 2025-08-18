// modules/formatconverter.go

// Package modules provides functionality for different modules in the MetaRekordFixer application.
// Each module handles a specific task related to DJ database management and music file operations.

// This module converts music files from the source folder to the destination folder while maintaining the same folder structure.
// It also allows selecting the target format and format-specific settings.

package modules

import (
	"context"
	"errors"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"MetaRekordFixer/assets"
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// FormatConverterModule implements a module for converting music files between different formats.
// It provides a user interface for selecting source and target formats, folders, and conversion parameters,
// and uses ffmpeg to perform the actual audio conversion with metadata preservation.
type FormatConverterModule struct {
	// ModuleBase provides common module functionality like error handling and UI components
	*common.ModuleBase // Embedded pointer to shared base

	// Source and target settings
	makeTargetFolderCheckbox *widget.Check
	sourceFolderEntry        *widget.Entry
	sourceFolderField        fyne.CanvasObject
	sourceFormatSelect       *widget.Select
	targetFolderEntry        *widget.Entry
	targetFolderField        fyne.CanvasObject
	targetFormatSelect       *widget.Select
	rewriteExistingCheckbox  *widget.Check

	// Format-specific settings
	// MP3 settings
	MP3BitrateSelect    *widget.Select
	MP3SampleRateSelect *widget.Select
	// FLAC settings
	FLACBitDepthSelect    *widget.Select
	FLACCompressionSelect *widget.Select
	FLACSampleRateSelect  *widget.Select
	// WAV settings
	WAVBitDepthSelect   *widget.Select
	WAVSampleRateSelect *widget.Select

	// Format settings containers
	FLACSettingsContainer   *fyne.Container
	formatSettingsContainer *fyne.Container
	MP3SettingsContainer    *fyne.Container
	WAVSettingsContainer    *fyne.Container

	// Submit button
	submitBtn *widget.Button

	// Current state
	currentTargetFormat string
	isConverting        bool
	metadataMap         *MetadataMap

	// Current ffmpeg process
	currentProcess *exec.Cmd

	// Cancel context and function for stopping ffmpeg
	cancelFunc context.CancelFunc
	ctx        context.Context

	// Logger for ffmpeg output
	ffmpegLogger *common.Logger
}

// NewFormatConverterModule creates a new instance of FormatConverterModule.
// It initializes the module with the provided window, configuration manager, and error handler,
// sets up the UI components, and loads any saved configuration.
//
// Parameters:
//   - window: The main application window
//   - configMgr: Configuration manager for saving/loading module settings
//   - errorHandler: Error handler for displaying and logging errors
//
// Returns:
//   - A fully initialized FormatConverterModule instance
func NewFormatConverterModule(window fyne.Window, configMgr *common.ConfigManager, errorHandler *common.ErrorHandler) *FormatConverterModule {
	m := &FormatConverterModule{
		ModuleBase:   common.NewModuleBase(window, configMgr, errorHandler),
		isConverting: false,
	}

	// FFmpeg logger initialization
	//
	// We do NOT handle errors here. If the main application logger works,
	// it is almost guaranteed that ffmpeg logger will work too, because it uses
	// the exact same path and permissions logic. If ffmpeg logger fails to initialize
	// (which should never happen in normal operation), we simply do not log ffmpeg output
	// to a separate file. This keeps the code simple and avoids unnecessary user warnings.
	//
	// If you ever change the log path logic or permissions, reconsider this approach.
	ffmpegLogPath, err := common.LocateOrCreatePath("metarekordfixer_ffmpeg.log", "log")
	if err == nil {
		ffmpegLogger, err := common.NewLogger(ffmpegLogPath, 10, 7)
		if err == nil {
			m.ffmpegLogger = ffmpegLogger
		}
	}

	// Initialize UI and load config
	m.initializeUI()
	m.LoadCfg()

	return m
}

// GetName returns the localized name of the module.
// This implements the Module interface method.
func (m *FormatConverterModule) GetName() string {
	return locales.Translate("formatconverter.mod.name")
}

// GetConfigName returns the configuration identifier for the module.
// This key is used to store and retrieve module-specific configuration.
func (m *FormatConverterModule) GetConfigName() string {
	return "formatconverter"
}

// GetIcon returns the module's icon resource.
// This implements the Module interface method and provides the visual representation
// of this module in the UI.
func (m *FormatConverterModule) GetIcon() fyne.Resource {
	return theme.MediaMusicIcon()
}

// GetModuleContent returns the module's specific content without status messages.
// This implements the method from ModuleBase to provide the module-specific UI
// containing the source/target format selection, folder selection, and format-specific settings.
//
// The UI is organized into left and right panels:
// - Left panel: Source and target folder/format selection and general options
// - Right panel: Format-specific settings that change based on the selected target format
func (m *FormatConverterModule) GetModuleContent() fyne.CanvasObject {
	// Left section - Source and target settings
	leftHeader := common.CreateDescriptionLabel(locales.Translate("formatconverter.label.leftpanel"))

	// Source folder container
	sourceFolderField := common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		m.sourceFolderEntry,
		func(path string) {
			m.sourceFolderEntry.SetText(common.NormalizePath(path))
			m.SaveCfg()
		},
	)
	sourceContainer := container.NewBorder(
		nil, nil,
		m.sourceFormatSelect, nil,
		sourceFolderField,
	)

	// Target folder container
	targetFolderField := common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		m.targetFolderEntry,
		func(path string) {
			m.targetFolderEntry.SetText(common.NormalizePath(path))
			m.SaveCfg()
		},
	)
	targetContainer := container.NewBorder(
		nil, nil,
		m.targetFormatSelect, nil,
		targetFolderField,
	)

	// Create form for source and target inputs
	inputForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("formatconverter.label.source"), Widget: sourceContainer},
			{Text: locales.Translate("formatconverter.label.target"), Widget: targetContainer},
		},
		SubmitText: "",
		OnSubmit:   nil,
	}

	// Checkboxes for additional options
	checkboxesContainer := container.NewVBox(
		m.rewriteExistingCheckbox,
		m.makeTargetFolderCheckbox,
	)

	// Combine all elements for the left section
	leftSection := container.NewVBox(
		leftHeader,
		widget.NewSeparator(),
		container.NewVBox(
			inputForm,
			widget.NewSeparator(),
			checkboxesContainer,
		),
	)

	// Right section - Format-specific settings
	rightHeader := common.CreateDescriptionLabel(locales.Translate("formatconverter.label.rightpanel"))

	// Combine all elements for the right section
	rightSection := container.NewVBox(
		rightHeader,
		widget.NewSeparator(),
		m.formatSettingsContainer,
	)

	// Create a horizontal container with left and right sections
	horizontalLayout := container.NewHSplit(leftSection, rightSection)
	// Set a fixed position for the divider (80% for left, 20% for right)
	horizontalLayout.SetOffset(0.8)

	// Create module content with description and separator
	moduleContent := container.NewVBox(
		common.CreateDescriptionLabel(locales.Translate("formatconverter.label.info")),
		widget.NewSeparator(),
		horizontalLayout,
	)

	// Add submit button if provided
	if m.submitBtn != nil {
		moduleContent.Add(container.NewHBox(layout.NewSpacer(), m.submitBtn))
	}

	return moduleContent
}

// GetContent returns the module's main UI content.
// This method returns the complete module layout with status messages container.
func (m *FormatConverterModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadCfg loads typed configuration and sets UI elements
func (m *FormatConverterModule) LoadCfg() {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load typed config from ConfigManager
	config, err := m.ConfigMgr.GetModuleCfg("formatconverter", m.GetConfigName())
	if err != nil {
		// This should not happen with the updated GetModuleCfg(), but handle gracefully
		return
	}

	// Cast to FormatConverter specific config
	if cfg, ok := config.(common.FormatConverterCfg); ok {
		// Update UI elements with loaded values
		if m.sourceFolderEntry != nil {
			m.sourceFolderEntry.SetText(cfg.SourceFolder.Value)
		}
		if m.targetFolderEntry != nil {
			m.targetFolderEntry.SetText(cfg.TargetFolder.Value)
		}
		if m.sourceFormatSelect != nil {
			localizedValue := sourceFormatParams.GetLocalizedValue(cfg.SourceFormat.Value)
			m.sourceFormatSelect.SetSelected(localizedValue)
		}
		if m.targetFormatSelect != nil {
			m.targetFormatSelect.SetSelected(cfg.TargetFormat.Value)
			m.updateFormatSettings(cfg.TargetFormat.Value)
		}
		if m.rewriteExistingCheckbox != nil {
			m.rewriteExistingCheckbox.SetChecked(cfg.RewriteExisting.Value == "true")
		}
		if m.makeTargetFolderCheckbox != nil {
			m.makeTargetFolderCheckbox.SetChecked(cfg.MakeTargetFolder.Value == "true")
		}

		// Load format-specific settings
		if m.MP3BitrateSelect != nil {
			localizedValue := mp3BitrateParams.GetLocalizedValue(cfg.MP3Bitrate.Value)
			m.MP3BitrateSelect.SetSelected(localizedValue)
		}
		if m.MP3SampleRateSelect != nil {
			localizedValue := sampleRateParams.GetLocalizedValue(cfg.MP3Samplerate.Value)
			m.MP3SampleRateSelect.SetSelected(localizedValue)
		}
		if m.FLACCompressionSelect != nil {
			localizedValue := flacCompressionParams.GetLocalizedValue(cfg.FLACCompression.Value)
			m.FLACCompressionSelect.SetSelected(localizedValue)
		}
		if m.FLACSampleRateSelect != nil {
			localizedValue := sampleRateParams.GetLocalizedValue(cfg.FLACSamplerate.Value)
			m.FLACSampleRateSelect.SetSelected(localizedValue)
		}
		if m.FLACBitDepthSelect != nil {
			localizedValue := bitDepthParams.GetLocalizedValue(cfg.FLACBitdepth.Value)
			m.FLACBitDepthSelect.SetSelected(localizedValue)
		}
		if m.WAVSampleRateSelect != nil {
			localizedValue := sampleRateParams.GetLocalizedValue(cfg.WAVSamplerate.Value)
			m.WAVSampleRateSelect.SetSelected(localizedValue)
		}
		if m.WAVBitDepthSelect != nil {
			localizedValue := bitDepthParams.GetLocalizedValue(cfg.WAVBitdepth.Value)
			m.WAVBitDepthSelect.SetSelected(localizedValue)
		}
	}

	// Ensure metadata map is loaded
	m.metadataMap, _ = m.loadMetadataMap()
}

// SaveCfg saves current UI state to typed configuration
func (m *FormatConverterModule) SaveCfg() {
	if m.IsLoadingConfig {
		return // Safeguard: no save if config is being loaded
	}

	// Get default configuration with all field definitions
	cfg := common.GetDefaultFormatConverterCfg()

	// Update only the values from current UI state
	cfg.SourceFolder.Value = common.NormalizePath(m.sourceFolderEntry.Text)
	cfg.TargetFolder.Value = common.NormalizePath(m.targetFolderEntry.Text)
	cfg.SourceFormat.Value = sourceFormatParams.GetConfigValue(m.sourceFormatSelect.Selected)
	cfg.TargetFormat.Value = m.targetFormatSelect.Selected
	cfg.MakeTargetFolder.Value = fmt.Sprintf("%t", m.makeTargetFolderCheckbox.Checked)
	cfg.RewriteExisting.Value = fmt.Sprintf("%t", m.rewriteExistingCheckbox.Checked)
	cfg.MP3Bitrate.Value = mp3BitrateParams.GetConfigValue(m.MP3BitrateSelect.Selected)
	cfg.MP3Samplerate.Value = sampleRateParams.GetConfigValue(m.MP3SampleRateSelect.Selected)
	cfg.FLACBitdepth.Value = bitDepthParams.GetConfigValue(m.FLACBitDepthSelect.Selected)
	cfg.FLACSamplerate.Value = sampleRateParams.GetConfigValue(m.FLACSampleRateSelect.Selected)
	cfg.FLACCompression.Value = flacCompressionParams.GetConfigValue(m.FLACCompressionSelect.Selected)
	cfg.WAVBitdepth.Value = bitDepthParams.GetConfigValue(m.WAVBitDepthSelect.Selected)
	cfg.WAVSamplerate.Value = sampleRateParams.GetConfigValue(m.WAVSampleRateSelect.Selected)

	// Save typed config via ConfigManager
	m.ConfigMgr.SaveModuleCfg("formatconverter", m.GetConfigName(), cfg)
}

// initializeUI sets up the user interface components.
// It creates and configures all UI elements including entry fields, select boxes,
// checkboxes, and buttons, and sets up their event handlers.
func (m *FormatConverterModule) initializeUI() {
	// Source format selection
	sourceFormats := []string{
		locales.Translate("formatconverter.srcformats.all"),
		"MP3",
		"FLAC",
		"WAV",
	}
	m.sourceFormatSelect = widget.NewSelect(sourceFormats, func(format string) {
		m.onSourceFormatChanged(format)
	})

	// Target format selection
	targetFormats := []string{
		"MP3",
		"FLAC",
		"WAV",
	}
	m.targetFormatSelect = widget.NewSelect(targetFormats, func(format string) {
		m.onTargetFormatChanged(format)
		m.SaveCfg()
	})

	// Initialize source folder entry first
	m.sourceFolderEntry = widget.NewEntry()
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveCfg()
	})

	// Source folder selection - store reference for UI usage
	m.sourceFolderField = common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		m.sourceFolderEntry,
		func(path string) {
			m.sourceFolderEntry.SetText(common.NormalizePath(path))
			m.SaveCfg()
		},
	)

	// Initialize target folder entry first
	m.targetFolderEntry = widget.NewEntry()
	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveCfg()
	})

	// Target folder selection - store reference for UI usage
	m.targetFolderField = common.CreateFolderSelectionField(
		locales.Translate("common.entry.placeholderpath"),
		m.targetFolderEntry,
		func(path string) {
			m.targetFolderEntry.SetText(common.NormalizePath(path))
			m.SaveCfg()
		},
	)

	// Checkboxes
	m.rewriteExistingCheckbox = common.CreateCheckbox(locales.Translate("formatconverter.chkbox.rewrite"), nil)
	m.rewriteExistingCheckbox.OnChanged = m.CreateBoolChangeHandler(func() { m.SaveCfg() })

	m.makeTargetFolderCheckbox = common.CreateCheckbox(locales.Translate("formatconverter.chkbox.maketargetfolder"), nil)
	m.makeTargetFolderCheckbox.OnChanged = m.CreateBoolChangeHandler(func() { m.SaveCfg() })

	// Initialize format-specific settings
	// MP3 settings
	mp3BitrateOptions := mp3BitrateParams.GetLocalizedValues()
	m.MP3BitrateSelect = widget.NewSelect(mp3BitrateOptions, nil)
	m.MP3BitrateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	mp3SampleRateOptions := sampleRateParams.GetLocalizedValues()
	m.MP3SampleRateSelect = widget.NewSelect(mp3SampleRateOptions, nil)
	m.MP3SampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	// FLAC settings
	flacCompressionOptions := flacCompressionParams.GetLocalizedValues()
	m.FLACCompressionSelect = widget.NewSelect(flacCompressionOptions, nil)
	m.FLACCompressionSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	flacSampleRateOptions := sampleRateParams.GetLocalizedValues()
	m.FLACSampleRateSelect = widget.NewSelect(flacSampleRateOptions, nil)
	m.FLACSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	flacBitDepthOptions := bitDepthParams.GetLocalizedValues()
	m.FLACBitDepthSelect = widget.NewSelect(flacBitDepthOptions, nil)
	m.FLACBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	// WAV settings
	wavSampleRateOptions := sampleRateParams.GetLocalizedValues()
	m.WAVSampleRateSelect = widget.NewSelect(wavSampleRateOptions, nil)
	m.WAVSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	wavBitDepthOptions := bitDepthParams.GetLocalizedValues()
	m.WAVBitDepthSelect = widget.NewSelect(wavBitDepthOptions, nil)
	m.WAVBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveCfg() })

	// Create format settings containers
	mp3BitrateLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.bitrate"))
	mp3SampleRateLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.samplerate"))
	m.MP3SettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, mp3BitrateLabel, m.MP3BitrateSelect),
		container.NewGridWithColumns(2, mp3SampleRateLabel, m.MP3SampleRateSelect),
	)

	FLACCompressionLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.compress"))
	FLACSampleRateLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.samplerate"))
	FLACBitDepthLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.bitdepth"))
	m.FLACSettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, FLACCompressionLabel, m.FLACCompressionSelect),
		container.NewGridWithColumns(2, FLACSampleRateLabel, m.FLACSampleRateSelect),
		container.NewGridWithColumns(2, FLACBitDepthLabel, m.FLACBitDepthSelect),
	)

	WAVSampleRateLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.samplerate"))
	WAVBitDepthLabel := widget.NewLabel(locales.Translate("formatconverter.configpar.bitdepth"))
	m.WAVSettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, WAVSampleRateLabel, m.WAVSampleRateSelect),
		container.NewGridWithColumns(2, WAVBitDepthLabel, m.WAVBitDepthSelect),
	)

	// Main format settings container (will hold the appropriate settings based on selected format)
	m.formatSettingsContainer = container.NewVBox()

	// Submit button
	m.submitBtn = common.CreateSubmitButton(locales.Translate("formatconverter.button.start"), func() {
		go m.Start()
	},
	)

	// Set up change handlers for all UI components
	// Source and target format selections
	m.sourceFormatSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.onSourceFormatChanged(m.sourceFormatSelect.Selected)
	})
	m.targetFormatSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.onTargetFormatChanged(m.targetFormatSelect.Selected)
	})

	// Checkboxes
	m.rewriteExistingCheckbox.OnChanged = m.CreateBoolChangeHandler(func() {
		m.SaveCfg()
	})
	m.makeTargetFolderCheckbox.OnChanged = m.CreateBoolChangeHandler(func() {
		m.SaveCfg()
	})

	// MP3 settings
	m.MP3BitrateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})
	m.MP3SampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})

	// FLAC settings
	m.FLACCompressionSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})
	m.FLACSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})
	m.FLACBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})

	// WAV settings
	m.WAVSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})
	m.WAVBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		m.SaveCfg()
	})

	// Folder entries
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveCfg()
	})
	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		m.SaveCfg()
	})
}

// onSourceFormatChanged handles changes in source format selection.
// It saves the updated configuration when the source format is changed.
//
// Parameters:
//   - _: The selected format (unused in this implementation)
func (m *FormatConverterModule) onSourceFormatChanged(_ string) {
	// Save configuration
	m.SaveCfg()
}

// onTargetFormatChanged handles changes in target format selection.
// It updates the format settings container to show format-specific options
// and saves the updated configuration.
//
// Parameters:
//   - format: The selected target format (MP3, FLAC, WAV)
func (m *FormatConverterModule) onTargetFormatChanged(format string) {

	// Update format settings container
	m.updateFormatSettings(format)

	// Save configuration
	m.SaveCfg()
}

// updateFormatSettings updates the format settings container based on the selected target format.
// It shows different settings panels depending on whether MP3, FLAC, or WAV is selected.
//
// Parameters:
//   - format: The selected target format (MP3, FLAC, WAV)
func (m *FormatConverterModule) updateFormatSettings(format string) {
	// Safety check - if containers are not initialized yet, return
	if m.formatSettingsContainer == nil {

		return
	}

	// Clear current content
	m.formatSettingsContainer.Objects = []fyne.CanvasObject{}
	m.currentTargetFormat = format

	// Exact string comparison for format types
	switch format {
	case "MP3":
		if m.MP3SettingsContainer != nil {
			m.formatSettingsContainer.Add(m.MP3SettingsContainer)

		}
	case "FLAC":
		if m.FLACSettingsContainer != nil {
			m.formatSettingsContainer.Add(m.FLACSettingsContainer)

		}
	case "WAV":
		if m.WAVSettingsContainer != nil {
			m.formatSettingsContainer.Add(m.WAVSettingsContainer)

		}
	default:
		// No format selected or unsupported format
		m.formatSettingsContainer.Add(widget.NewLabel(locales.Translate("formatconverter.formatsel.default")))

	}

	// Force refresh of the container
	m.formatSettingsContainer.Refresh()
}

// IsCancelled returns whether the current operation has been cancelled.
// It extends the base implementation to also kill any running ffmpeg process
// when cancellation is detected.
//
// Returns:
//   - true if the operation has been cancelled, false otherwise
func (m *FormatConverterModule) IsCancelled() bool {
	isCancelled := m.ModuleBase.IsCancelled()
	if m.currentProcess != nil && isCancelled {
		// Kill the ffmpeg process if it's running
		if err := m.currentProcess.Process.Kill(); err != nil {
			context := &common.ErrorContext{
				Module:      m.GetName(),
				Operation:   "killProcess",
				Severity:    common.SeverityWarning,
				Recoverable: true,
			}
			m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("formatconverter.err.killprocess")), context)
		}
	}
	return isCancelled
}

// Start performs the necessary steps before starting the main process.
// It validates the inputs and starts the conversion process if validation passes.
func (m *FormatConverterModule) Start() {

	// Create and run validator
	validator := common.NewValidator(m, m.ConfigMgr, nil, m.ErrorHandler)
	if err := validator.Validate("start"); err != nil {
		return
	}

	// Start the conversion process
	m.startConversion()
}

// startConversion begins the conversion process.
// It checks if a conversion is already in progress, disables the submit button,
// retrieves configuration values, and starts the file conversion in a goroutine.
func (m *FormatConverterModule) startConversion() {
	// Check if already converting
	if m.isConverting {
		return
	}
	m.isConverting = true
	defer func() { m.isConverting = false }()

	// Disable the button during processing and set icon after completion
	m.submitBtn.Disable()
	defer func() {
		m.submitBtn.Enable()
		m.submitBtn.SetIcon(theme.ConfirmIcon())
	}()

	// Get values from typed configuration
	config, err := m.ConfigMgr.GetModuleCfg("formatconverter", m.GetConfigName())
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "startConversion",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	cfg, ok := config.(common.FormatConverterCfg)
	if !ok {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "startConversion",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New("invalid configuration type"), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	sourceFolder := cfg.SourceFolder.Value
	targetFolder := cfg.TargetFolder.Value
	targetFormat := cfg.TargetFormat.Value

	// Get format-specific settings
	formatSettings := make(map[string]string)

	switch targetFormat {
	case "MP3":
		formatSettings["bitrate"] = cfg.MP3Bitrate.Value
		if formatSettings["bitrate"] == "" {
			formatSettings["bitrate"] = "320k"
		}
		formatSettings["samplerate"] = cfg.MP3Samplerate.Value
		if formatSettings["samplerate"] == "" {
			formatSettings["samplerate"] = "copy"
		}
	case "FLAC":
		formatSettings["compression"] = cfg.FLACCompression.Value
		if formatSettings["compression"] == "" {
			formatSettings["compression"] = "12"
		}
		formatSettings["samplerate"] = cfg.FLACSamplerate.Value
		if formatSettings["samplerate"] == "" {
			formatSettings["samplerate"] = "copy"
		}
		formatSettings["bitdepth"] = cfg.FLACBitdepth.Value
		if formatSettings["bitdepth"] == "" {
			formatSettings["bitdepth"] = "copy"
		}
	case "WAV":
		formatSettings["samplerate"] = cfg.WAVSamplerate.Value
		if formatSettings["samplerate"] == "" {
			formatSettings["samplerate"] = "copy"
		}
		formatSettings["bitdepth"] = cfg.WAVBitdepth.Value
		if formatSettings["bitdepth"] == "" {
			formatSettings["bitdepth"] = "copy"
		}
	}

	// Log conversion parameters
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.source"), sourceFolder))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.target"), targetFolder))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.format"), targetFormat))

	// Perform the actual conversion
	go m.convertFiles(sourceFolder, targetFolder, targetFormat, formatSettings)
}

// convertFiles performs the actual conversion of audio files using ffmpeg.
// It finds all audio files in the source folder, creates the necessary folder structure,
// and converts each file with the specified format settings while preserving metadata.
//
// Parameters:
//   - sourceFolder: Path to the folder containing source audio files
//   - targetFolder: Path where converted files will be saved
//   - targetFormat: Target format (MP3, FLAC, WAV)
//   - formatSettings: Map of format-specific settings like bitrate, compression level, etc.
func (m *FormatConverterModule) convertFiles(sourceFolder, targetFolder, targetFormat string, formatSettings map[string]string) {
	// Get values from typed configuration
	config, err := m.ConfigMgr.GetModuleCfg("formatconverter", m.GetConfigName())
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "convertFiles",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	cfg, ok := config.(common.FormatConverterCfg)
	if !ok {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "convertFiles",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New("invalid configuration type"), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Find all audio files in the source folder
	sourceFormat := cfg.SourceFormat.Value
	if sourceFormat == "" {
		sourceFormat = "All"
	}
	files, err := m.findAudioFiles(sourceFolder, sourceFormat)
	if err != nil {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "findAudioFiles",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(err, context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}
	if len(files) == 0 {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "convertFiles",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("formatconverter.err.nosourcefiles")), context)
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		return
	}

	// Create cancelable context for ffmpeg
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancelFunc = cancel
	m.ShowProgressDialog(
		locales.Translate("formatconverter.dialog.header"),
		func() {
			cancel()
			m.HandleProcessCancellation("common.status.stopping")
		},
	)

	// Show progress dialog only after all validations pass
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.filesfound"), len(files)))

	// Create initial target folder structure if needed
	basePath := targetFolder
	makeTargetFolder := cfg.MakeTargetFolder.Value == "true"
	if makeTargetFolder {
		sourceFolderBase := filepath.Base(sourceFolder)
		basePath = filepath.Join(targetFolder, sourceFolderBase)

		// Ensure base target directory exists
		if err := os.MkdirAll(basePath, 0755); err != nil {
			context := &common.ErrorContext{
				Module:      m.GetName(),
				Operation:   "createTargetFolder",
				Severity:    common.SeverityWarning,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(err, context)
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatconverter.err.createfolder"), err))
			return
		}
		m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.foldercreated"), sourceFolderBase))
	}

	// Track conversion statistics
	successCount := 0
	skippedCount := 0
	failedFiles := []string{}

	// Process each file
	for i, file := range files {
		// Check if cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("formatconverter.dialog.stop", successCount, len(files))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Update progress
		progress := float64(i) / float64(len(files))
		statusText := fmt.Sprintf(locales.Translate("formatconverter.status.progress"), i+1, len(files))
		m.UpdateProgressStatus(progress, statusText)

		// Get relative path from source folder
		relPath, _ := filepath.Rel(sourceFolder, file)

		// Determine target path
		targetPath := basePath

		// Get directory part of relative path
		relDir := filepath.Dir(relPath)
		if relDir != "." {
			targetPath = filepath.Join(targetPath, relDir)

			// Create subdirectories in target
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				context := &common.ErrorContext{
					Module:    m.GetName(),
					Operation: "createSubdirectories",
					Severity:  common.SeverityWarning,
				}
				m.ErrorHandler.ShowStandardError(err, context)
				m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatconverter.err.createfolder"), err))
				failedFiles = append(failedFiles, file)
				continue
			}
		}
		// Get filename without extension
		fileBase := filepath.Base(file)
		fileNameWithoutExt := strings.TrimSuffix(fileBase, filepath.Ext(fileBase))

		// Determine target file extension based on format
		var targetExt string
		switch targetFormat {
		case "MP3":
			targetExt = ".mp3"
		case "FLAC":
			targetExt = ".flac"
		case "WAV":
			targetExt = ".wav"
		default:
			targetExt = ".mp3" // Fallback to MP3 as default
		}

		// Full target file path
		targetFile := filepath.Join(targetPath, fileNameWithoutExt+targetExt)

		// Check if target file exists and if we should skip it
		rewriteExisting := cfg.RewriteExisting.Value == "true"
		if _, err := os.Stat(targetFile); err == nil && !rewriteExisting {
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.skipping"), filepath.Base(targetFile)))
			skippedCount++
			continue
		}

		// Extract metadata from source file using ffprobe
		metadata, err := m.extractMetadata(file)
		if err != nil {
			context := &common.ErrorContext{
				Module:    m.GetName(),
				Operation: "extractMetadata",
				Severity:  common.SeverityWarning,
			}
			m.ErrorHandler.ShowStandardError(err, context)
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatconverter.err.readmeta"), err))
			failedFiles = append(failedFiles, file)
			continue
		}

		// Convert file with ffmpeg
		bitDepth, sampleRate, err := m.getAudioProperties(file)
		if err != nil {
			context := &common.ErrorContext{
				Module:    m.GetName(),
				Operation: "getAudioProperties",
				Severity:  common.SeverityWarning,
			}
			m.ErrorHandler.ShowStandardError(err, context)
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatconverter.err.readprops"), err))
			failedFiles = append(failedFiles, file)
			continue
		}

		err = m.convertFile(file, targetFile, targetFormat, formatSettings, metadata, bitDepth, sampleRate, m.metadataMap)
		if err != nil {
			// Check if the error is due to cancellation
			if m.IsCancelled() {
				m.HandleProcessCancellation("formatconverter.dialog.stop", successCount, len(files))
				common.UpdateButtonToCompleted(m.submitBtn)
				return
			} else {
				// Handle regular conversion error
				context := &common.ErrorContext{
					Module:      m.GetName(),
					Operation:   "convertFiles",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("formatconverter.err.duringconv")), context)
				failedFiles = append(failedFiles, file)
				continue
			}
		}

		successCount++
	}

	// Complete the process
	m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("formatconverter.status.done"), successCount, len(files)))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.doneall"), successCount))

	// Report skipped files
	if skippedCount > 0 {
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatconverter.status.skipped"), skippedCount))
	}

	// Report failed files
	if len(failedFiles) > 0 {
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("formatconverter.status.failed"), len(failedFiles)))
		for _, file := range failedFiles {
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("formatconverter.status.faileditems"), filepath.Base(file)))
		}
	}
	// Mark progress dialog as completed
	m.CompleteProgressDialog()
	common.UpdateButtonToCompleted(m.submitBtn)
}

// convertFile converts a single audio file using ffmpeg.
// It builds the appropriate ffmpeg command line arguments based on the target format
// and settings, maps metadata between formats, and executes the conversion.
//
// Parameters:
//   - sourcePath: Path to the source audio file
//   - targetPath: Path where the converted file will be saved
//   - targetFormat: Target format (MP3, FLAC, WAV)
//   - formatSettings: Map of format-specific settings
//   - metadata: Map of metadata from the source file
//   - bitDepth: Bit depth of the source file
//   - sampleRate: Sample rate of the source file
//   - metadataMap: Mapping rules for metadata between different formats
//
// Returns:
//   - error if the conversion fails, nil otherwise
func (m *FormatConverterModule) convertFile(sourcePath, targetPath, targetFormat string, formatSettings map[string]string, metadata map[string]string, bitDepth string, sampleRate string, metadataMap *MetadataMap) error {
	// Build ffmpeg arguments
	args := []string{
		"-i", sourcePath,
		"-y",                  // Overwrite output file without asking
		"-map_metadata", "-1", // Prevent metadata copying using ffmpeg rules. We apply own rules for metadata mapping.
	}

	// Add format-specific settings
	switch targetFormat {
	case "MP3":
		// MP3 settings
		bitrateConfig := formatSettings["bitrate"]
		sampleRateConfig := formatSettings["sample_rate"]

		args = append(args, "-c:a", "libmp3lame")

		// Use value for ffmpeg based on configuration
		if bitrateConfig != "" {
			bitrateValue := mp3BitrateParams.GetFFmpegValue(bitrateConfig, "")
			if bitrateValue != "-" {
				args = append(args, "-b:a", bitrateValue)
			}
		}

		// Use value for ffmpeg based on configuration and source file
		if sampleRateConfig != "" {
			sampleRateValue := sampleRateParams.GetFFmpegValue(sampleRateConfig, sampleRate)
			if sampleRateValue != "-" {
				args = append(args, "-ar", sampleRateValue)
			}
		}

		// Set ID3v2.4 version
		args = append(args, "-id3v2_version", "4")
	case "FLAC":
		// Add FLAC specific settings
		compressionConfig := formatSettings["compression"]
		sampleRateConfig := formatSettings["sample_rate"]
		bitDepthConfig := formatSettings["bit_depth"]

		args = append(args, "-c:a", "flac")

		// Use value for ffmpeg based on configuration
		if compressionConfig != "" {
			compressionValue := flacCompressionParams.GetFFmpegValue(compressionConfig, "")
			if compressionValue != "-" {
				args = append(args, "-compression_level", compressionValue)
			}
		}

		// Use value for ffmpeg based on configuration and source file
		if sampleRateConfig != "" {
			sampleRateValue := sampleRateParams.GetFFmpegValue(sampleRateConfig, sampleRate)
			if sampleRateValue != "-" {
				args = append(args, "-ar", sampleRateValue)
			}
		}

		// Use value for ffmpeg based on configuration and source file
		if bitDepthConfig != "" {
			// For FLAC we need to convert bit depth to sample format
			bitDepthValue := bitDepthParams.GetFFmpegValue(bitDepthConfig, bitDepth)
			if bitDepthValue != "-" {
				// Convert to sample format for FLAC
				var sampleFormat string
				switch bitDepthValue {
				case "16":
					sampleFormat = "s16"
				case "24":
					sampleFormat = "s24"
				case "32":
					sampleFormat = "s32"
				default:
					sampleFormat = "s16" // Default to 16-bit
				}
				args = append(args, "-sample_fmt", sampleFormat)
			}
		}

	case "WAV":
		// Add WAV specific settings
		sampleRateConfig := formatSettings["sample_rate"]
		bitDepthConfig := formatSettings["bit_depth"]

		// Use value for ffmpeg based on configuration and source file
		// For WAV we need to convert bit depth to codec format
		if bitDepthConfig != "" {
			bitDepthValue := bitDepthParams.GetFFmpegValue(bitDepthConfig, bitDepth)
			if bitDepthValue != "-" {
				// Convert to codec format for WAV
				var sampleFormat string
				switch bitDepthValue {
				case "16":
					sampleFormat = "pcm_s16le"
				case "24":
					sampleFormat = "pcm_s24le"
				case "32":
					sampleFormat = "pcm_s32le"
				default:
					sampleFormat = "pcm_s24le" // Default to 24-bit
				}
				args = append(args, "-c:a", sampleFormat)
			}
		}

		// Use value for ffmpeg based on configuration and source file
		if sampleRateConfig != "" {
			sampleRateValue := sampleRateParams.GetFFmpegValue(sampleRateConfig, sampleRate)
			if sampleRateValue != "-" {
				args = append(args, "-ar", sampleRateValue)
			}
		}
	}

	// Create a sorted slice of metadata items to ensure consistent order
	type metadataItem struct {
		key   string
		value string
	}
	var metadataItems []metadataItem

	// Map metadata from source to target format
	for internalName, targetField := range metadataMap.InternalToMP3 {
		// Find a matching metadata field in the source
		var foundValue string
		var found bool

		// First try to find a matching field in the source
		for sourceField, value := range metadata {
			if strings.EqualFold(sourceField, internalName) {
				foundValue = value
				found = true
				break
			}
		}

		// Special case for album_artist, which may be in different formats
		if !found && (strings.EqualFold(internalName, "ALBUMARTIST") || strings.EqualFold(internalName, "album_artist")) {
			// Check for different possible formats
			for sourceField, value := range metadata {
				if strings.EqualFold(sourceField, "ALBUMARTIST") ||
					strings.EqualFold(sourceField, "album_artist") ||
					strings.EqualFold(sourceField, "ALBUM_ARTIST") ||
					strings.EqualFold(sourceField, "AlbumArtist") {
					foundValue = value
					found = true
					break
				}
			}
		}

		if found {
			// Escape special characters in the value part
			escapedValue := foundValue
			escapedValue = strings.ReplaceAll(escapedValue, "\\", "\\\\")
			escapedValue = strings.ReplaceAll(escapedValue, "\"", "\\\"")

			// Add to metadata items slice
			metadataItems = append(metadataItems, metadataItem{
				key:   targetField,
				value: escapedValue,
			})
		}
	}

	// Sort metadata items by key to ensure consistent order
	sort.Slice(metadataItems, func(i, j int) bool {
		return metadataItems[i].key < metadataItems[j].key
	})

	// Add sorted metadata to ffmpeg arguments
	for _, item := range metadataItems {
		args = append(args, "-metadata", fmt.Sprintf("%s=%s", item.key, item.value))
	}

	// Add output file path
	args = append(args, targetPath)

	// Create ffmpeg command
	cmd := exec.CommandContext(m.ctx, "tools/ffmpeg.exe", args...)
	m.currentProcess = cmd

	// Run ffmpeg and get output
	output, err := cmd.CombinedOutput()

	// Clear process reference
	m.currentProcess = nil

	// Always log ffmpeg output
	if m.ffmpegLogger != nil {
		m.ffmpegLogger.Info("FFMPEG %s -> %s\n%s", sourcePath, targetPath, string(output))
	}

	// Check if process was cancelled
	if m.IsCancelled() {
		// Remove partial output
		os.Remove(targetPath)

		// Log the cancellation
		m.Logger.Info("Module: %s, Operation: %s - %s", m.GetName(), "convertFile", locales.Translate("common.log.cancelled"))

		return errors.New(locales.Translate("common.log.cancelled"))
	}

	// Check for other errors
	if err != nil {
		// Log the ffmpeg error
		m.Logger.Error("Module: %s, Operation: %s - %s", m.GetName(), "convertFile", fmt.Sprintf(locales.Translate("formatconverter.err.ffmpeg"), err))

		return fmt.Errorf(locales.Translate("formatconverter.err.ffmpeg"), err)
	}

	return nil
}

// MetadataMap represents the mapping between metadata fields for different formats.
// It provides translation tables between internal field names and format-specific field names.
type MetadataMap struct {
	// InternalToMP3 maps internal field names to MP3 (ID3) field names
	InternalToMP3 map[string]string
	// InternalToFLAC maps internal field names to FLAC field names
	InternalToFLAC map[string]string
	// InternalToWAV maps internal field names to WAV field names
	InternalToWAV map[string]string
}

// ConversionParameter represents a single parameter option for conversion.
// It stores the mapping between UI representation, configuration value, and ffmpeg value.
type ConversionParameter struct {
	ConfigValue string // value stored in configuration (e.g. "5", "copy")
	FFmpegValue string // value for ffmpeg (e.g. "5", "-")
	LocaleKey   string // localization key (e.g. "convert.compression.medium")
	IsCopy      bool   // indicates if this is a "copy" parameter
}

// ConversionParameterSet represents a set of parameters for a specific setting.
// It provides methods to convert between localized values, configuration values, and ffmpeg values.
type ConversionParameterSet struct {
	// Parameters is the list of available parameter options
	Parameters []ConversionParameter
}

// GetLocalizedValues returns a list of localized values for GUI display.
// This is used to populate select boxes with human-readable, localized options.
//
// Returns:
//   - A slice of localized strings for all parameters in the set
func (ps *ConversionParameterSet) GetLocalizedValues() []string {
	values := make([]string, len(ps.Parameters))
	for i, p := range ps.Parameters {
		values[i] = locales.Translate(p.LocaleKey)
	}
	return values
}

// GetConfigValue returns a configuration value based on localized text.
// This converts from the UI display value to the value stored in configuration.
//
// Parameters:
//   - localizedValue: The localized text displayed in the UI
//
// Returns:
//   - The corresponding configuration value, or "copy" as fallback
func (ps *ConversionParameterSet) GetConfigValue(localizedValue string) string {
	for _, p := range ps.Parameters {
		if locales.Translate(p.LocaleKey) == localizedValue {
			return p.ConfigValue
		}
	}
	return "copy" // fallback to copy if no match found
}

// GetFFmpegValue returns a value for ffmpeg based on configuration value and optionally source properties.
// For "copy" parameters, it returns the source value if provided.
//
// Parameters:
//   - configValue: The value from configuration
//   - sourceValue: The value from the source file (used for "copy" parameters)
//
// Returns:
//   - The value to use in ffmpeg command line arguments
func (ps *ConversionParameterSet) GetFFmpegValue(configValue string, sourceValue string) string {
	for _, p := range ps.Parameters {
		if p.ConfigValue == configValue {
			if p.IsCopy && sourceValue != "" {
				return sourceValue // use value from source file
			}
			return p.FFmpegValue
		}
	}
	return "-" // fallback to copy
}

// GetLocalizedValue returns localized text based on configuration value.
// This converts from the stored configuration value to the UI display value.
//
// Parameters:
//   - configValue: The value from configuration
//
// Returns:
//   - The corresponding localized text for UI display
func (ps *ConversionParameterSet) GetLocalizedValue(configValue string) string {
	for _, p := range ps.Parameters {
		if p.ConfigValue == configValue {
			return locales.Translate(p.LocaleKey)
		}
	}
	return locales.Translate("formatconverter.configpar.copypar") // fallback to copy
}

// Parameter definitions for conversion
var (
	// Source format parameters
	sourceFormatParams = ConversionParameterSet{
		Parameters: []ConversionParameter{
			{ConfigValue: "All", FFmpegValue: "All", LocaleKey: "formatconverter.srcformats.all", IsCopy: false},
			{ConfigValue: "MP3", FFmpegValue: "MP3", LocaleKey: "formatconverter.srcformats.mp3", IsCopy: false},
			{ConfigValue: "FLAC", FFmpegValue: "FLAC", LocaleKey: "formatconverter.srcformats.flac", IsCopy: false},
			{ConfigValue: "WAV", FFmpegValue: "WAV", LocaleKey: "formatconverter.srcformats.wav", IsCopy: false},
		},
	}

	// FLAC compression parameters
	flacCompressionParams = ConversionParameterSet{
		Parameters: []ConversionParameter{
			{ConfigValue: "5", FFmpegValue: "5", LocaleKey: "formatconverter.configpar.compressmed", IsCopy: false},
			{ConfigValue: "12", FFmpegValue: "12", LocaleKey: "formatconverter.configpar.compressfull", IsCopy: false},
			{ConfigValue: "0", FFmpegValue: "0", LocaleKey: "formatconverter.configpar.nocompress", IsCopy: false},
		},
	}

	// MP3 bitrate parameters
	mp3BitrateParams = ConversionParameterSet{
		Parameters: []ConversionParameter{
			{ConfigValue: "copy", FFmpegValue: "-", LocaleKey: "formatconverter.configpar.copypar", IsCopy: true},
			{ConfigValue: "128k", FFmpegValue: "128k", LocaleKey: "formatconverter.bitrate.128", IsCopy: false},
			{ConfigValue: "192k", FFmpegValue: "192k", LocaleKey: "formatconverter.bitrate.192", IsCopy: false},
			{ConfigValue: "256k", FFmpegValue: "256k", LocaleKey: "formatconverter.bitrate.256", IsCopy: false},
			{ConfigValue: "320k", FFmpegValue: "320k", LocaleKey: "formatconverter.bitrate.320", IsCopy: false},
		},
	}

	// Sample rate parameters
	sampleRateParams = ConversionParameterSet{
		Parameters: []ConversionParameter{
			{ConfigValue: "copy", FFmpegValue: "-", LocaleKey: "formatconverter.configpar.copypar", IsCopy: true},
			{ConfigValue: "44100", FFmpegValue: "44100", LocaleKey: "formatconverter.samplerate.44", IsCopy: false},
			{ConfigValue: "48000", FFmpegValue: "48000", LocaleKey: "formatconverter.samplerate.48", IsCopy: false},
			{ConfigValue: "96000", FFmpegValue: "96000", LocaleKey: "formatconverter.samplerate.96", IsCopy: false},
			{ConfigValue: "192000", FFmpegValue: "192000", LocaleKey: "formatconverter.samplerate.192", IsCopy: false},
		},
	}

	// Bit depth parameters
	bitDepthParams = ConversionParameterSet{
		Parameters: []ConversionParameter{
			{ConfigValue: "copy", FFmpegValue: "-", LocaleKey: "formatconverter.configpar.copypar", IsCopy: true},
			{ConfigValue: "16", FFmpegValue: "16", LocaleKey: "formatconverter.bitdepth.16", IsCopy: false},
			{ConfigValue: "24", FFmpegValue: "24", LocaleKey: "formatconverter.bitdepth.24", IsCopy: false},
			{ConfigValue: "32", FFmpegValue: "32", LocaleKey: "formatconverter.bitdepth.32", IsCopy: false},
		},
	}
)

// loadMetadataMap loads the metadata mapping from the embedded CSV file.
// The CSV file defines how metadata fields should be mapped between different audio formats.
//
// Returns:
//   - A populated MetadataMap structure and nil error on success
//   - nil and an error if loading or parsing fails
func (m *FormatConverterModule) loadMetadataMap() (*MetadataMap, error) {
	// Load the CSV content from the embedded file
	csvContent := assets.ResourceMetadataMapCSV.Content()

	// Create a new CSV reader from the content
	reader := csv.NewReader(bytes.NewReader(csvContent))

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("formatconverter.err.readcsvheader"), err)
	}

	// Initialize maps
	result := &MetadataMap{
		InternalToMP3:  make(map[string]string),
		InternalToFLAC: make(map[string]string),
		InternalToWAV:  make(map[string]string),
	}

	// Find column indices
	mpIndex := -1
	flacIndex := -1
	wavIndex := -1
	for i, col := range header {
		switch col {
		case "MP3":
			mpIndex = i
		case "FLAC":
			flacIndex = i
		case "WAV":
			wavIndex = i
		}
	}

	if mpIndex == -1 || flacIndex == -1 || wavIndex == -1 {
		return nil, errors.New(locales.Translate("formatconverter.err.metamapheader"))
	}

	// Read and process each row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New(locales.Translate("formatconverter.err.readmaprow"))

		}

		// Skip empty rows
		if len(record) == 0 || record[0] == "" {
			continue
		}

		// Map the fields
		internalName := record[0]
		result.InternalToMP3[internalName] = record[mpIndex]
		result.InternalToFLAC[internalName] = record[flacIndex]
		result.InternalToWAV[internalName] = record[wavIndex]
	}

	return result, nil
}

// findAudioFiles recursively finds all audio files in the given directory.
// If sourceFormat is specified (not "All"), only files of that format are returned.
//
// Parameters:
//   - dir: The directory to search for audio files
//   - sourceFormat: The format to filter by ("All", "MP3", "FLAC", "WAV")
//
// Returns:
//   - A slice of paths to matching audio files
//   - An error if directory reading fails
func (m *FormatConverterModule) findAudioFiles(dir string, sourceFormat string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%s '%s': %w", locales.Translate("formatconverter.err.accesspath"), path, err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get file extension
		ext := strings.ToLower(filepath.Ext(path))

		// Filter by format if specified
		if sourceFormat != "All" {
			switch sourceFormat {
			case "MP3":
				if ext != ".mp3" {
					return nil
				}
			case "FLAC":
				if ext != ".flac" {
					return nil
				}
			case "WAV":
				if ext != ".wav" {
					return nil
				}
			}
		} else {
			// For "All", accept any supported format
			if ext != ".mp3" && ext != ".flac" && ext != ".wav" {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s '%s': %w", locales.Translate("formatconverter.err.accesspath"), dir, err)
	}

	return files, nil
}

// extractMetadata extracts metadata from an audio file using ffprobe
func (m *FormatConverterModule) extractMetadata(filePath string) (map[string]string, error) {
	cmd := exec.Command("tools/ffprobe.exe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)

	// Get command output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s '%s': %w", locales.Translate("formatconverter.err.readmeta"), filepath.Base(filePath), err)

	}

	// Parse JSON output
	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("%s: %w", locales.Translate("formatconverter.err.parsemeta"), err)
	}

	return result.Format.Tags, nil
}

// getAudioProperties extracts audio properties (bit depth, sample rate) from a file using ffprobe
func (m *FormatConverterModule) getAudioProperties(filePath string) (bitDepth string, sampleRate string, err error) {
	cmd := exec.Command("tools/ffprobe.exe", "-v", "quiet", "-print_format", "json", "-show_streams", filePath)

	// Get command output
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("%s '%s': %w", locales.Translate("formatconverter.err.readprops"), filepath.Base(filePath), err)
	}

	// Parse JSON output
	var result struct {
		Streams []struct {
			CodecType   string      `json:"codec_type"`
			SampleRate  string      `json:"sample_rate"`
			SampleFmt   string      `json:"sample_fmt"`
			BitsPerRaw  json.Number `json:"bits_per_raw_sample"`
			BitsPerSamp json.Number `json:"bits_per_sample"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", "", fmt.Errorf("%s: %w", locales.Translate("formatconverter.err.parseprops"), err)
	}

	// Find the audio stream
	for _, stream := range result.Streams {
		if stream.CodecType == "audio" {
			// Get sample rate
			sampleRate = stream.SampleRate

			// Try to determine bit depth
			if stream.BitsPerRaw != "" {
				bitDepth = string(stream.BitsPerRaw)
			} else if stream.BitsPerSamp != "" {
				bitDepth = string(stream.BitsPerSamp)
			} else {
				// Try to determine from sample format
				switch stream.SampleFmt {
				case "u8", "u8p":
					bitDepth = "8"
				case "s16", "s16p":
					bitDepth = "16"
				case "s32", "s32p", "flt", "fltp":
					bitDepth = "32"
				case "s64", "s64p", "dbl", "dblp":
					bitDepth = "64"
				default:
					bitDepth = "16" // Default to 16-bit if unknown
				}
			}

			return bitDepth, sampleRate, nil
		}
	}

	return bitDepth, sampleRate, errors.New(locales.Translate("formatconverter.err.noaudio"))
}

// Close releases resources held by the module (logger for ffmpeg included)
func (m *FormatConverterModule) Close() {
	if m.ffmpegLogger != nil {
		_ = m.ffmpegLogger.Close()
	}
}
