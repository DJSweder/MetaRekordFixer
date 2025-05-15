// modules/music_converter.go

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

// MusicConverterModule implements a module for converting music files between different formats
type MusicConverterModule struct {
	*common.ModuleBase // Embedded pointer to shared base

	// Source and target settings
	makeTargetFolderCheckbox *widget.Check
	sourceFolderEntry        *widget.Entry
	sourceFormatSelect       *widget.Select
	targetFolderEntry        *widget.Entry
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
}

// NewMusicConverterModule creates a new instance of MusicConverterModule
func NewMusicConverterModule(window fyne.Window, configMgr *common.ConfigManager, errorHandler *common.ErrorHandler) *MusicConverterModule {
	m := &MusicConverterModule{
		ModuleBase:   common.NewModuleBase(window, configMgr, errorHandler),
		isConverting: false,
	}
	// Check if module has configuration, if not, create default
	cfg := m.ConfigMgr.GetModuleConfig(m.GetConfigName())

	if len(cfg.Fields) == 0 {
		cfg = m.SetDefaultConfig()
		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	}
	// Initialize UI and load config
	m.IsLoadingConfig = true
	m.initializeUI()
	m.LoadConfig(cfg)
	m.IsLoadingConfig = false

	return m
}

// GetName returns the localized name of the module
func (m *MusicConverterModule) GetName() string {
	return locales.Translate("convert.mod.name")
}

// GetConfigName returns the configuration identifier for the module
func (m *MusicConverterModule) GetConfigName() string {
	return "music_converter"
}

// GetIcon returns the module's icon
func (m *MusicConverterModule) GetIcon() fyne.Resource {
	return theme.MediaMusicIcon()
}

// GetModuleContent returns the module's specific content without status messages
// This implements the method from ModuleBase to provide the module-specific UI
func (m *MusicConverterModule) GetModuleContent() fyne.CanvasObject {
	// Left section - Source and target settings
	leftHeader := common.CreateDescriptionLabel(locales.Translate("convert.label.leftpanel"))

	// Source folder container
	sourceBrowseBtn := common.CreateNativeFolderBrowseButton(
		locales.Translate("common.entry.placeholderpath"),
		"",
		func(path string) {
			m.sourceFolderEntry.SetText(path)
			m.SaveConfig()
		},
	)
	sourceContainer := container.NewBorder(
		nil, nil,
		m.sourceFormatSelect, sourceBrowseBtn,
		m.sourceFolderEntry,
	)

	// Target folder container
	targetBrowseBtn := common.CreateNativeFolderBrowseButton(
		locales.Translate("common.entry.placeholderpath"),
		"",
		func(path string) {
			m.targetFolderEntry.SetText(path)
			m.SaveConfig()
		},
	)
	targetContainer := container.NewBorder(
		nil, nil,
		m.targetFormatSelect, targetBrowseBtn,
		m.targetFolderEntry,
	)

	// Create form for source and target inputs
	inputForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: locales.Translate("convert.label.source"), Widget: sourceContainer},
			{Text: locales.Translate("convert.label.target"), Widget: targetContainer},
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
	rightHeader := common.CreateDescriptionLabel(locales.Translate("convert.label.rightpanel"))

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
		common.CreateDescriptionLabel(locales.Translate("convert.label.info")),
		widget.NewSeparator(),
		horizontalLayout,
	)

	// Add submit button if provided
	if m.submitBtn != nil {
		moduleContent.Add(container.NewHBox(layout.NewSpacer(), m.submitBtn))
	}

	return moduleContent
}

// GetContent returns the module's main UI content
func (m *MusicConverterModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// LoadConfig loads module configuration
func (m *MusicConverterModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load source and target folder paths
	if m.sourceFolderEntry != nil {
		if sourceFolder := cfg.Get("source_folder", ""); sourceFolder != "" {
			m.sourceFolderEntry.SetText(sourceFolder)
		}
	}

	if m.targetFolderEntry != nil {
		if targetFolder := cfg.Get("target_folder", ""); targetFolder != "" {
			m.targetFolderEntry.SetText(targetFolder)
		}
	}

	// Load format selections
	if m.sourceFormatSelect != nil {
		if sourceFormat := cfg.Get("source_format", ""); sourceFormat != "" {
			m.sourceFormatSelect.SetSelected(sourceFormat)
		} else {
			m.sourceFormatSelect.SetSelected(locales.Translate("convert.srcformats.all"))
		}
	}

	if m.targetFormatSelect != nil {
		if targetFormat := cfg.Get("target_format", ""); targetFormat != "" {
			m.targetFormatSelect.SetSelected(targetFormat)
			// Update format settings panel based on selected format
			m.updateFormatSettings(targetFormat)
		} else {
			m.targetFormatSelect.SetSelected("MP3")
			// Update format settings panel for MP3
			m.updateFormatSettings("MP3")
		}
	}

	// Load checkboxes
	if m.rewriteExistingCheckbox != nil {
		m.rewriteExistingCheckbox.SetChecked(cfg.GetBool("rewrite_existing", false))
	}
	if m.makeTargetFolderCheckbox != nil {
		m.makeTargetFolderCheckbox.SetChecked(cfg.GetBool("make_target_folder", false))
	}

	// Load format-specific settings
	// MP3
	if m.MP3BitrateSelect != nil {
		if mp3Bitrate := cfg.Get("mp3_bitrate", ""); mp3Bitrate != "" {
			m.MP3BitrateSelect.SetSelected(mp3Bitrate)
		}
	}

	if m.MP3SampleRateSelect != nil {
		if mp3SampleRate := cfg.Get("mp3_samplerate", ""); mp3SampleRate != "" {
			m.MP3SampleRateSelect.SetSelected(mp3SampleRate)
		}
	}

	// FLAC
	if m.FLACCompressionSelect != nil {
		if flacCompression := cfg.Get("flac_compression", ""); flacCompression != "" {
			m.FLACCompressionSelect.SetSelected(flacCompression)
		}
	}

	if m.FLACSampleRateSelect != nil {
		if flacSampleRate := cfg.Get("flac_samplerate", ""); flacSampleRate != "" {
			m.FLACSampleRateSelect.SetSelected(flacSampleRate)
		}
	}

	if m.FLACBitDepthSelect != nil {
		if flacBitDepth := cfg.Get("flac_bitdepth", ""); flacBitDepth != "" {
			m.FLACBitDepthSelect.SetSelected(flacBitDepth)
		}
	}

	// WAV
	if m.WAVSampleRateSelect != nil {
		if wavSampleRate := cfg.Get("wav_samplerate", ""); wavSampleRate != "" {
			m.WAVSampleRateSelect.SetSelected(wavSampleRate)
		}
	}
	if m.WAVBitDepthSelect != nil {
		if wavBitDepth := cfg.Get("wav_bitdepth", ""); wavBitDepth != "" {
			m.WAVBitDepthSelect.SetSelected(wavBitDepth)
		}
	}

	// Ensure metadata map is loaded
	m.metadataMap, _ = m.loadMetadataMap()
}

// SaveConfig saves the current module configuration
func (m *MusicConverterModule) SaveConfig() common.ModuleConfig {
	cfg := m.ConfigMgr.GetModuleConfig(m.GetConfigName())
	if cfg.Fields == nil {
		cfg = m.SetDefaultConfig()
	}

	// Save source and target folder paths with validation
	if m.sourceFolderEntry != nil {
		cfg.SetWithDefinition("source_folder", m.sourceFolderEntry.Text, "folder", true, "exists")
	}
	if m.targetFolderEntry != nil {
		cfg.SetWithDefinition("target_folder", m.targetFolderEntry.Text, "folder", true, "exists")
	}

	// Save format selections with validation
	if m.sourceFormatSelect != nil {
		cfg.SetWithDefinition("source_format", m.sourceFormatSelect.Selected, "select", true, "none")
	}
	if m.targetFormatSelect != nil {
		cfg.SetWithDefinition("target_format", m.targetFormatSelect.Selected, "select", true, "none")
	}

	// Save checkboxes
	if m.rewriteExistingCheckbox != nil {
		cfg.SetBoolWithDefinition("rewrite_existing", m.rewriteExistingCheckbox.Checked, false, "none")
	}
	if m.makeTargetFolderCheckbox != nil {
		cfg.SetBoolWithDefinition("make_target_folder", m.makeTargetFolderCheckbox.Checked, false, "none")
	}

	// Save format-specific settings with dependencies
	// MP3
	if m.MP3BitrateSelect != nil {
		cfg.SetWithDependency("mp3_bitrate", m.MP3BitrateSelect.Selected, "select", true, "target_format", "MP3", "none")
	}

	if m.MP3SampleRateSelect != nil {
		cfg.SetWithDependency("mp3_samplerate", m.MP3SampleRateSelect.Selected, "select", true, "target_format", "MP3", "none")
	}

	// FLAC
	if m.FLACCompressionSelect != nil {
		cfg.SetWithDependency("flac_compression", m.FLACCompressionSelect.Selected, "select", true, "target_format", "FLAC", "none")
	}

	if m.FLACSampleRateSelect != nil {
		cfg.SetWithDependency("flac_samplerate", m.FLACSampleRateSelect.Selected, "select", true, "target_format", "FLAC", "none")
	}

	if m.FLACBitDepthSelect != nil {
		cfg.SetWithDependency("flac_bitdepth", m.FLACBitDepthSelect.Selected, "select", true, "target_format", "FLAC", "none")
	}

	// WAV
	if m.WAVSampleRateSelect != nil {
		cfg.SetWithDependency("wav_samplerate", m.WAVSampleRateSelect.Selected, "select", true, "target_format", "WAV", "none")
	}
	if m.WAVBitDepthSelect != nil {
		cfg.SetWithDependency("wav_bitdepth", m.WAVBitDepthSelect.Selected, "select", true, "target_format", "WAV", "none")
	}

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// initializeUI sets up the user interface components
func (m *MusicConverterModule) initializeUI() {
	// Source folder selection
	m.sourceFolderEntry = widget.NewEntry()
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	// Target folder selection
	m.targetFolderEntry = widget.NewEntry()
	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	// Source format selection
	sourceFormats := []string{
		locales.Translate("convert.srcformats.all"),
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
		m.SaveConfig()
	})

	// Checkboxes
	m.rewriteExistingCheckbox = common.CreateCheckbox(locales.Translate("convert.chkbox.rewrite"), nil)
	m.rewriteExistingCheckbox.OnChanged = m.CreateBoolChangeHandler(func() { m.SaveConfig() })

	m.makeTargetFolderCheckbox = common.CreateCheckbox(locales.Translate("convert.chkbox.maketargetfolder"), nil)
	m.makeTargetFolderCheckbox.OnChanged = m.CreateBoolChangeHandler(func() { m.SaveConfig() })

	// Initialize format-specific settings
	// MP3 settings
	mp3BitrateOptions := []string{"320 kbps", "256 kbps", "192 kbps", "128 kbps"}
	m.MP3BitrateSelect = widget.NewSelect(mp3BitrateOptions, nil)
	m.MP3BitrateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	mp3SampleRateOptions := []string{locales.Translate("convert.configpar.copypar"), "44.1 kHz", "48 kHz", "96 kHz", "192 kHz"}
	m.MP3SampleRateSelect = widget.NewSelect(mp3SampleRateOptions, nil)
	m.MP3SampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	// FLAC settings
	flacCompressionOptions := []string{
		locales.Translate("convert.configpar.compressfull"),
		locales.Translate("convert.configpar.compressmed"),
		locales.Translate("convert.configpar.nocompress"),
	}
	m.FLACCompressionSelect = widget.NewSelect(flacCompressionOptions, nil)
	m.FLACCompressionSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	flacSampleRateOptions := []string{locales.Translate("convert.configpar.copypar"), "44.1 kHz", "48 kHz", "96 kHz", "192 kHz"}
	m.FLACSampleRateSelect = widget.NewSelect(flacSampleRateOptions, nil)
	m.FLACSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	flacBitDepthOptions := []string{locales.Translate("convert.configpar.copypar"), "32", "24", "16"}
	m.FLACBitDepthSelect = widget.NewSelect(flacBitDepthOptions, nil)
	m.FLACBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	// WAV settings
	wavSampleRateOptions := []string{locales.Translate("convert.configpar.copypar"), "44.1 kHz", "48 kHz", "96 kHz", "192 kHz"}
	m.WAVSampleRateSelect = widget.NewSelect(wavSampleRateOptions, nil)
	m.WAVSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	wavBitDepthOptions := []string{locales.Translate("convert.configpar.copypar"), "32", "24", "16"}
	m.WAVBitDepthSelect = widget.NewSelect(wavBitDepthOptions, nil)
	m.WAVBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	// Create format settings containers
	mp3BitrateLabel := widget.NewLabel(locales.Translate("convert.configpar.bitrate"))
	mp3SampleRateLabel := widget.NewLabel(locales.Translate("convert.configpar.samplerate"))
	m.MP3SettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, mp3BitrateLabel, m.MP3BitrateSelect),
		container.NewGridWithColumns(2, mp3SampleRateLabel, m.MP3SampleRateSelect),
	)

	FLACCompressionLabel := widget.NewLabel(locales.Translate("convert.configpar.compress"))
	FLACSampleRateLabel := widget.NewLabel(locales.Translate("convert.configpar.samplerate"))
	FLACBitDepthLabel := widget.NewLabel(locales.Translate("convert.configpar.bitdepth"))
	m.FLACSettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, FLACCompressionLabel, m.FLACCompressionSelect),
		container.NewGridWithColumns(2, FLACSampleRateLabel, m.FLACSampleRateSelect),
		container.NewGridWithColumns(2, FLACBitDepthLabel, m.FLACBitDepthSelect),
	)

	WAVSampleRateLabel := widget.NewLabel(locales.Translate("convert.configpar.samplerate"))
	WAVBitDepthLabel := widget.NewLabel(locales.Translate("convert.configpar.bitdepth"))
	m.WAVSettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, WAVSampleRateLabel, m.WAVSampleRateSelect),
		container.NewGridWithColumns(2, WAVBitDepthLabel, m.WAVBitDepthSelect),
	)

	// Main format settings container (will hold the appropriate settings based on selected format)
	m.formatSettingsContainer = container.NewVBox()

	// Submit button
	m.submitBtn = common.CreateSubmitButton(locales.Translate("convert.button.start"), func() {
		go m.startConversion()
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
		_ = m.SaveConfig()
	})
	m.makeTargetFolderCheckbox.OnChanged = m.CreateBoolChangeHandler(func() {
		_ = m.SaveConfig()
	})

	// MP3 settings
	m.MP3BitrateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})
	m.MP3SampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})

	// FLAC settings
	m.FLACCompressionSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})
	m.FLACSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})
	m.FLACBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})

	// WAV settings
	m.WAVSampleRateSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})
	m.WAVBitDepthSelect.OnChanged = m.CreateSelectionChangeHandler(func() {
		_ = m.SaveConfig()
	})

	// Folder entries
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		_ = m.SaveConfig()
	})
	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() {
		_ = m.SaveConfig()
	})
}

// onSourceFormatChanged handles changes in source format selection
func (m *MusicConverterModule) onSourceFormatChanged(_ string) {
	// Save configuration
	m.SaveConfig()
}

// onTargetFormatChanged handles changes in target format selection
func (m *MusicConverterModule) onTargetFormatChanged(format string) {

	// Update format settings container
	m.updateFormatSettings(format)

	// Save configuration
	m.SaveConfig()
}

// updateFormatSettings updates the format settings container based on the selected target format
func (m *MusicConverterModule) updateFormatSettings(format string) {
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

		} else {

		}
	case "FLAC":
		if m.FLACSettingsContainer != nil {
			m.formatSettingsContainer.Add(m.FLACSettingsContainer)

		} else {

		}
	case "WAV":
		if m.WAVSettingsContainer != nil {
			m.formatSettingsContainer.Add(m.WAVSettingsContainer)

		} else {

		}
	default:
		// No format selected or unsupported format
		m.formatSettingsContainer.Add(widget.NewLabel(locales.Translate("convert.formatsel.default")))

	}

	// Force refresh of the container
	m.formatSettingsContainer.Refresh()
}

// IsCancelled returns whether the current operation has been cancelled
func (m *MusicConverterModule) IsCancelled() bool {
	isCancelled := m.ModuleBase.IsCancelled()
	if m.currentProcess != nil && isCancelled {
		// Kill the ffmpeg process if it's running
		if err := m.currentProcess.Process.Kill(); err != nil {
			m.AddErrorMessage(fmt.Sprintf(locales.Translate("convert.err.killprocess"), err))
		} else {
		}
	}
	return isCancelled
}

// startConversion begins the conversion process
func (m *MusicConverterModule) startConversion() {
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

	// Clear previous status messages
	m.ClearStatusMessages()

	// Validate inputs
	sourceFolder := m.sourceFolderEntry.Text
	targetFolder := m.targetFolderEntry.Text
	targetFormat := m.targetFormatSelect.Selected

	// Validate source folder exists
	if _, err := os.Stat(sourceFolder); os.IsNotExist(err) {
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "startConversion",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("convert.err.nosource")), context)
		return
	}

	// Get format-specific settings
	formatSettings := make(map[string]string)

	switch targetFormat {
	case "MP3":
		// MP3 settings
		bitrate := m.MP3BitrateSelect.Selected
		sampleRateSetting := m.MP3SampleRateSelect.Selected
		formatSettings["bitrate"] = bitrate
		formatSettings["sample_rate"] = sampleRateSetting
	case "FLAC":
		// FLAC settings
		compression := m.FLACCompressionSelect.Selected
		sampleRate := m.FLACSampleRateSelect.Selected
		bitDepth := m.FLACBitDepthSelect.Selected
		formatSettings["compression"] = compression
		formatSettings["sample_rate"] = sampleRate
		formatSettings["bit_depth"] = bitDepth
	case "WAV":
		// WAV settings
		sampleRate := m.WAVSampleRateSelect.Selected
		bitDepth := m.WAVBitDepthSelect.Selected
		formatSettings["sample_rate"] = sampleRate
		formatSettings["bit_depth"] = bitDepth
	}

	// Check if target folder exists, create if needed and option is selected
	if m.makeTargetFolderCheckbox.Checked {
		// Create target folder if it doesn't exist
		if _, err := os.Stat(targetFolder); os.IsNotExist(err) {
			err := os.MkdirAll(targetFolder, 0755)
			if err != nil {
				m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
				context := &common.ErrorContext{
					Module:      m.GetName(),
					Operation:   "startConversion",
					Severity:    common.SeverityCritical,
					Recoverable: false,
				}
				m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("convert.err.createfolder")), context)
				return
			}
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.foldercreated"), targetFolder))
		}
	} else {
		// Check if target folder exists
		if _, err := os.Stat(targetFolder); os.IsNotExist(err) {
			m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
			context := &common.ErrorContext{
				Module:      m.GetName(),
				Operation:   "startConversion",
				Severity:    common.SeverityCritical,
				Recoverable: false,
			}
			m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("convert.err.nofolder")), context)
			return
		}
	}
	// Try to make test file for write permission check
	testFile := filepath.Join(targetFolder, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		// Permission denied
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))
		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "startConversion",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("convert.err.nowriteaccess")), context)
		return
	}
	// Clean up test file
	f.Close()
	os.Remove(testFile)

	// Add initial status message
	m.AddInfoMessage(locales.Translate("convert.status.starting"))

	// Log conversion parameters
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.source"), sourceFolder))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.target"), targetFolder))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.format"), targetFormat))

	// Perform the actual conversion
	go m.convertFiles(sourceFolder, targetFolder, targetFormat, formatSettings)
}

// convertFiles performs the actual conversion of audio files using ffmpeg
func (m *MusicConverterModule) convertFiles(sourceFolder, targetFolder, targetFormat string, formatSettings map[string]string) {
	// Find all audio files in the source folder
	files, _ := m.findAudioFiles(sourceFolder, m.sourceFormatSelect.Selected)

	if len(files) == 0 {
		// Add final error message to status messages
		m.AddErrorMessage(locales.Translate("common.err.statusfinal"))

		context := &common.ErrorContext{
			Module:      m.GetName(),
			Operation:   "convertFiles",
			Severity:    common.SeverityCritical,
			Recoverable: false,
		}
		m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("convert.err.nosourcefiles")), context)
		return
	}

	// Create cancelable context for ffmpeg
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancelFunc = cancel
	m.ShowProgressDialog(
		locales.Translate("convert.dialog.header"),
		func() {
			cancel()
			m.HandleProcessCancellation("convert.dialog.stop")
		},
	)

	// Show progress dialog only after all validations pass
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.filesfound"), len(files)))

	// Track conversion statistics
	successCount := 0
	skippedCount := 0
	failedFiles := []string{}

	// Process each file
	for i, file := range files {
		// Check if cancelled
		if m.IsCancelled() {
			m.HandleProcessCancellation("convert.dialog.stop", successCount, len(files))
			common.UpdateButtonToCompleted(m.submitBtn)
			return
		}

		// Update progress
		progress := float64(i) / float64(len(files))
		statusText := fmt.Sprintf(locales.Translate("convert.status.progress"), i+1, len(files))
		m.UpdateProgressStatus(progress, statusText)

		// Get relative path from source folder
		relPath, _ := filepath.Rel(sourceFolder, file)

		// Determine target path
		targetPath := targetFolder
		if m.makeTargetFolderCheckbox.Checked {
			// Create target folder if it doesn't exist
			sourceFolderBase := filepath.Base(sourceFolder)
			targetPath = filepath.Join(targetFolder, sourceFolderBase)

			// Ensure target directory exists
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				context := &common.ErrorContext{
					Module:    m.GetName(),
					Operation: "createTargetFolder",
					Severity:  common.SeverityWarning,
				}
				m.ErrorHandler.ShowStandardError(err, context)
				m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.err.createfolder"), err))
				failedFiles = append(failedFiles, file)
				continue
			}
		}

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
				m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.err.createfolder"), err))
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
		if _, err := os.Stat(targetFile); err == nil && !m.rewriteExistingCheckbox.Checked {
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.skipping"), filepath.Base(targetFile)))
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
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.err.readmeta"), err))
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
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.err.readprops"), err))
			failedFiles = append(failedFiles, file)
			continue
		}

		err = m.convertFile(file, targetFile, targetFormat, formatSettings, metadata, bitDepth, sampleRate, m.metadataMap)
		if err != nil {
			// Check if the error is due to cancellation
			if m.IsCancelled() {
				m.HandleProcessCancellation("convert.dialog.stop", successCount, len(files))
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
				m.ErrorHandler.ShowStandardError(errors.New(locales.Translate("convert.err.duringconv")), context)
				failedFiles = append(failedFiles, file)
				continue
			}
		}

		successCount++
	}

	// Complete the process
	m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("convert.status.done"), successCount, len(files)))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.doneall"), successCount))

	// Report skipped files
	if skippedCount > 0 {
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.status.skipped"), skippedCount))
	}

	// Report failed files
	if len(failedFiles) > 0 {
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.status.failed"), len(failedFiles)))
		for _, file := range failedFiles {
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.faileditems"), filepath.Base(file)))
		}
	}
	// Mark progress dialog as completed
	m.CompleteProgressDialog()
	common.UpdateButtonToCompleted(m.submitBtn)
}

// convertFile converts a single audio file using ffmpeg
func (m *MusicConverterModule) convertFile(sourcePath, targetPath, targetFormat string, formatSettings map[string]string, metadata map[string]string, bitDepth string, sampleRate string, metadataMap *MetadataMap) error {
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
		bitrate := formatSettings["bitrate"]
		sampleRateSetting := formatSettings["sample_rate"]

		args = append(args, "-c:a", "libmp3lame")
		if bitrate != "" && bitrate != locales.Translate("convert.configpar.copypar") {
			// Extract numeric part from bitrate (e.g. "320" from "320 kbps")
			bitrateValue := strings.Split(bitrate, " ")[0]
			args = append(args, "-b:a", bitrateValue+"k")
		}
		if sampleRateSetting != "" && sampleRateSetting != locales.Translate("convert.configpar.copypar") {
			// Extract numeric part from sample rate (e.g. "44.1" from "44.1 kHz")
			sampleRateValue := strings.Split(sampleRateSetting, " ")[0]
			// Convert to proper Hz value
			if strings.Contains(sampleRateValue, ".") {
				// For 44.1, convert to 44100
				sampleRateValue = strings.ReplaceAll(sampleRateValue, ".", "")
				args = append(args, "-ar", sampleRateValue+"00")
			} else {
				// For 48, convert to 48000
				args = append(args, "-ar", sampleRateValue+"000")
			}
		} else {
			// Use sample rate from source file
			args = append(args, "-ar", sampleRate)
		}

		// Set ID3v2.4 version
		args = append(args, "-id3v2_version", "4")
	case "FLAC":
		// Add FLAC specific settings
		compression := formatSettings["compression"]
		sampleRateSetting := formatSettings["sample_rate"]
		bitDepthSetting := formatSettings["bit_depth"]

		args = append(args, "-c:a", "flac")
		if compression != "" {
			var compressionLevel string
			switch compression {
			case locales.Translate("convert.configpar.nocompress"):
				compressionLevel = "0"
			case locales.Translate("convert.configpar.compressmed"):
				compressionLevel = "5"
			case locales.Translate("convert.configpar.compressfull"):
				compressionLevel = "12"
			default:
				compressionLevel = "5" // Default to medium compression
			}
			args = append(args, "-compression_level", compressionLevel)
		}

		if sampleRateSetting != "" && sampleRateSetting != locales.Translate("convert.configpar.copypar") {
			// Extract numeric part from sample rate (e.g. "44.1" from "44.1 kHz")
			sampleRateValue := strings.Split(sampleRateSetting, " ")[0]
			// Convert to proper Hz value
			if strings.Contains(sampleRateValue, ".") {
				// For 44.1, convert to 44100
				sampleRateValue = strings.ReplaceAll(sampleRateValue, ".", "")
				args = append(args, "-ar", sampleRateValue+"00")
			} else {
				// For 48, convert to 48000
				args = append(args, "-ar", sampleRateValue+"000")
			}
		} else {
			// Use sample rate from source file

			args = append(args, "-ar", sampleRate)
		}

		if bitDepthSetting != "" && bitDepthSetting != locales.Translate("convert.configpar.copypar") {
			// Convert bit depth to sample format
			var sampleFormat string
			switch bitDepthSetting {
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
		} else {
			// Use bit depth from source file
			var sampleFormat string
			switch bitDepth {
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

	case "WAV":
		// Add WAV specific settings
		sampleRate := formatSettings["sample_rate"]
		bitDepth := formatSettings["bit_depth"]

		// If bit depth is not set or set to "copy", use bit depth from source file
		if bitDepth == "" || bitDepth == locales.Translate("convert.configpar.copypar") {
			// Use bit depth from source file
			var sampleFormat string
			switch bitDepth {
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
		} else {
			// Set codec based on bit depth
			var sampleFormat string
			switch bitDepth {
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

		if sampleRate != "" && sampleRate != locales.Translate("convert.configpar.copypar") {
			// Extract numeric part from sample rate (e.g. "44.1" from "44.1 kHz")
			sampleRateValue := strings.Split(sampleRate, " ")[0]
			// Convert to proper Hz value
			if strings.Contains(sampleRateValue, ".") {
				// For 44.1, convert to 44100
				sampleRateValue = strings.ReplaceAll(sampleRateValue, ".", "")
				args = append(args, "-ar", sampleRateValue+"00")
			} else {
				// For 48, convert to 48000
				args = append(args, "-ar", sampleRateValue+"000")
			}
		} else {
			// Use sample rate from source file
			args = append(args, "-ar", sampleRate)
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

	// Run the command and wait for it to finish
	err := cmd.Run()

	// Clear process reference
	m.currentProcess = nil

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
		m.Logger.Error("Module: %s, Operation: %s - %s", m.GetName(), "convertFile", fmt.Sprintf(locales.Translate("convert.err.ffmpeg"), err))

		return fmt.Errorf(locales.Translate("convert.err.ffmpeg"), err)
	}

	return nil
}

// MetadataMap represents the mapping between metadata fields for different formats
type MetadataMap struct {
	InternalToMP3  map[string]string
	InternalToFLAC map[string]string
	InternalToWAV  map[string]string
}

// loadMetadataMap loads the metadata mapping from the CSV file
func (m *MusicConverterModule) loadMetadataMap() (*MetadataMap, error) {
	// Load the CSV content from the embedded file
	csvContent := assets.ResourceMetadataMapCSV.Content()

	// Create a new CSV reader from the content
	reader := csv.NewReader(bytes.NewReader(csvContent))

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		return nil, errors.New(locales.Translate("convert.err.readmap"))
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
		return nil, errors.New(locales.Translate("convert.err.missingcolumns"))
	}

	// Read and process each row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New(locales.Translate("convert.err.readmaprow"))
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

// findAudioFiles recursively finds all audio files in the given directory
// If sourceFormat is specified (not "All"), only files of that format are returned
func (m *MusicConverterModule) findAudioFiles(dir string, sourceFormat string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf(locales.Translate("convert.err.accesspath"), path)
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
		return nil, fmt.Errorf(locales.Translate("convert.err.scandir"), dir)
	}

	return files, nil
}

// extractMetadata extracts metadata from an audio file using ffprobe
func (m *MusicConverterModule) extractMetadata(filePath string) (map[string]string, error) {
	cmd := exec.Command("tools/ffprobe.exe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)

	// Get command output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(locales.Translate("convert.err.readmeta")+": %s - %v", filepath.Base(filePath), err)
	}

	// Parse JSON output
	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, errors.New(locales.Translate("convert.err.parsemeta"))
	}

	return result.Format.Tags, nil
}

// getAudioProperties extracts audio properties (bit depth, sample rate) from a file using ffprobe
func (m *MusicConverterModule) getAudioProperties(filePath string) (bitDepth string, sampleRate string, err error) {
	cmd := exec.Command("tools/ffprobe.exe", "-v", "quiet", "-print_format", "json", "-show_streams", filePath)

	// Get command output
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf(locales.Translate("convert.err.readprops")+": %s - %v", filepath.Base(filePath), err)
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
		return "", "", errors.New(locales.Translate("convert.err.parseprops"))
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

	return bitDepth, sampleRate, errors.New(locales.Translate("convert.err.noaudio"))
}

// SetDefaultConfig sets the default configuration values for the module
func (m *MusicConverterModule) SetDefaultConfig() common.ModuleConfig {

	// Create new config
	cfg := common.NewModuleConfig()

	// Set default source and target folders to empty strings
	cfg.SetWithDefinition("source_folder", "", "folder", true, "exists")
	cfg.SetWithDefinition("target_folder", "", "folder", true, "exists")

	// Set default formats
	// cfg.Set("source_format", locales.Translate("convert.srcformats.all"))
	cfg.SetWithDefinition("source_format", locales.Translate("convert.srcformats.all"), "select", true, "none")
	cfg.SetWithDefinition("target_format", "MP3", "select", true, "none")

	// Set default checkboxes
	cfg.SetBoolWithDefinition("rewrite_existing", false, false, "none")
	cfg.SetBoolWithDefinition("make_target_folder", false, false, "none")

	// Set default MP3 settings
	cfg.SetWithDependency("mp3_bitrate", "320 kbps", "select", true, "target_format", "MP3", "none")
	cfg.SetWithDependency("mp3_samplerate", locales.Translate("convert.configpar.copypar"), "select", true, "target_format", "MP3", "none")

	// Set default FLAC settings
	cfg.SetWithDependency("flac_compression", locales.Translate("convert.configpar.compressfull"), "select", true, "target_format", "FLAC", "none")
	cfg.SetWithDependency("flac_samplerate", locales.Translate("convert.configpar.copypar"), "select", true, "target_format", "FLAC", "none")
	cfg.SetWithDependency("flac_bitdepth", locales.Translate("convert.configpar.copypar"), "select", true, "target_format", "FLAC", "none")

	// Set default WAV settings
	cfg.SetWithDependency("wav_samplerate", locales.Translate("convert.configpar.copypar"), "select", true, "target_format", "WAV", "none")
	cfg.SetWithDependency("wav_bitdepth", locales.Translate("convert.configpar.copypar"), "select", true, "target_format", "WAV", "none")

	return cfg
}
