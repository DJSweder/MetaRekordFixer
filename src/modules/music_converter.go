// modules/music_converter.go

package modules

import (
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
	"strings"
)

// MusicConverterModule implements a module for converting music files between different formats
type MusicConverterModule struct {
	*common.ModuleBase // Embedded pointer to shared base

	// Source and target settings
	entrySourceFolder        *widget.Entry
	entryTargetFolder        *widget.Entry
	selectSourceFormat       *widget.Select
	selectTargetFormat       *widget.Select
	checkboxRewriteExisting  *widget.Check
	checkboxMakeTargetFolder *widget.Check

	// Format-specific settings
	// MP3 settings
	selectMP3Bitrate    *widget.Select
	selectMP3SampleRate *widget.Select
	// FLAC settings
	selectFLACCompression *widget.Select
	selectFLACSampleRate  *widget.Select
	selectFLACBitDepth    *widget.Select
	// WAV settings
	selectWAVSampleRate *widget.Select
	selectWAVBitDepth   *widget.Select

	// Format settings containers
	mp3SettingsContainer    *fyne.Container
	flacSettingsContainer   *fyne.Container
	wavSettingsContainer    *fyne.Container
	formatSettingsContainer *fyne.Container

	// Submit button
	submitButton *widget.Button

	// Current state
	currentTargetFormat string
	isConverting        bool
	isCancelled         bool
	metadataMap         *MetadataMap
}

// NewMusicConverterModule creates a new instance of MusicConverterModule
func NewMusicConverterModule(window fyne.Window, configMgr *common.ConfigManager, errorHandler *common.ErrorHandler) *MusicConverterModule {
	m := &MusicConverterModule{
		ModuleBase:   common.NewModuleBase(window, configMgr, errorHandler),
		isConverting: false,
		isCancelled:  false,
	}

	// Initialize UI components without triggering callbacks
	m.IsLoadingConfig = true
	m.initializeUI()
	m.IsLoadingConfig = false

	// Check if module has configuration, if not, create default
	cfg := m.ConfigMgr.GetModuleConfig(m.GetConfigName())

	// Check if config is empty by checking if any MP3 settings exist
	if cfg.Get("mp3_bitrate", "") == "" {
		// No existing config found, create default
		cfg = m.SetDefaultConfig()
		m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	}

	// Load configuration
	m.LoadConfig(cfg)

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
	leftHeader := widget.NewLabel(locales.Translate("convert.label.leftpanel"))
	leftHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Source folder container
	sourceBrowseBtn := common.CreateNativeFolderBrowseButton(
		locales.Translate("convert.label.sourcefolder"),
		"",
		func(path string) {
			m.entrySourceFolder.SetText(path)
			m.SaveConfig()
		},
	)
	sourceContainer := container.NewBorder(
		nil, nil,
		m.selectSourceFormat, sourceBrowseBtn,
		m.entrySourceFolder,
	)

	// Target folder container
	targetBrowseBtn := common.CreateNativeFolderBrowseButton(
		locales.Translate("convert.label.targetfolder"),
		"",
		func(path string) {
			m.entryTargetFolder.SetText(path)
			m.SaveConfig()
		},
	)
	targetContainer := container.NewBorder(
		nil, nil,
		m.selectTargetFormat, targetBrowseBtn,
		m.entryTargetFolder,
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
		m.checkboxRewriteExisting,
		m.checkboxMakeTargetFolder,
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
	rightHeader := widget.NewLabel(locales.Translate("convert.label.rightpanel"))
	rightHeader.TextStyle = fyne.TextStyle{Bold: true}

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
		widget.NewLabel(locales.Translate("convert.label.info")),
		widget.NewSeparator(),
		horizontalLayout,
	)

	// Add submit button if provided
	if m.submitButton != nil {
		moduleContent.Add(container.NewHBox(layout.NewSpacer(), m.submitButton))
	}

	return moduleContent
}

// GetContent returns the module's main UI content
func (m *MusicConverterModule) GetContent() fyne.CanvasObject {
	// Create the complete module layout with status messages container
	return m.CreateModuleLayoutWithStatusMessages(m.GetModuleContent())
}

// initializeUI sets up the user interface components
func (m *MusicConverterModule) initializeUI() {
	// Source folder selection
	m.entrySourceFolder = widget.NewEntry()
	m.entrySourceFolder.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	// Target folder selection
	m.entryTargetFolder = widget.NewEntry()
	m.entryTargetFolder.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	// Source format selection
	sourceFormats := []string{
		locales.Translate("convert.srcformats.all"),
		"MP3",
		"FLAC",
		"WAV",
	}
	m.selectSourceFormat = widget.NewSelect(sourceFormats, func(format string) {
		m.onSourceFormatChanged(format)
	})

	// Target format selection
	targetFormats := []string{
		"MP3",
		"FLAC",
		"WAV",
	}
	m.selectTargetFormat = widget.NewSelect(targetFormats, func(format string) {
		m.onTargetFormatChanged(format)
		m.SaveConfig()
	})

	// Checkboxes
	m.checkboxRewriteExisting = widget.NewCheck(locales.Translate("convert.chkbox.rewrite"), nil)
	m.checkboxRewriteExisting.OnChanged = m.CreateBoolChangeHandler(func() { m.SaveConfig() })

	m.checkboxMakeTargetFolder = widget.NewCheck(locales.Translate("convert.chkbox.maketargetfolder"), nil)
	m.checkboxMakeTargetFolder.OnChanged = m.CreateBoolChangeHandler(func() { m.SaveConfig() })

	// Initialize format-specific settings
	// MP3 settings
	mp3BitrateOptions := []string{"320 kbps", "256 kbps", "192 kbps", "128 kbps"}
	m.selectMP3Bitrate = widget.NewSelect(mp3BitrateOptions, nil)
	m.selectMP3Bitrate.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	mp3SampleRateOptions := []string{locales.Translate("convert.configpar.copypar"), "44.1 kHz", "48 kHz", "96 kHz", "192 kHz"}
	m.selectMP3SampleRate = widget.NewSelect(mp3SampleRateOptions, nil)
	m.selectMP3SampleRate.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	// FLAC settings
	flacCompressionOptions := []string{
		locales.Translate("convert.configpar.compressfull"),
		locales.Translate("convert.configpar.compressmed"),
		locales.Translate("convert.configpar.nocompress"),
	}
	m.selectFLACCompression = widget.NewSelect(flacCompressionOptions, nil)
	m.selectFLACCompression.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	flacSampleRateOptions := []string{locales.Translate("convert.configpar.copypar"), "44.1 kHz", "48 kHz", "96 kHz", "192 kHz"}
	m.selectFLACSampleRate = widget.NewSelect(flacSampleRateOptions, nil)
	m.selectFLACSampleRate.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	flacBitDepthOptions := []string{locales.Translate("convert.configpar.copypar"), "32", "24", "16"}
	m.selectFLACBitDepth = widget.NewSelect(flacBitDepthOptions, nil)
	m.selectFLACBitDepth.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	// WAV settings
	wavSampleRateOptions := []string{locales.Translate("convert.configpar.copypar"), "44.1 kHz", "48 kHz", "96 kHz", "192 kHz"}
	m.selectWAVSampleRate = widget.NewSelect(wavSampleRateOptions, nil)
	m.selectWAVSampleRate.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	wavBitDepthOptions := []string{locales.Translate("convert.configpar.copypar"), "32", "24", "16"}
	m.selectWAVBitDepth = widget.NewSelect(wavBitDepthOptions, nil)
	m.selectWAVBitDepth.OnChanged = m.CreateSelectionChangeHandler(func() { m.SaveConfig() })

	// Create format settings containers
	mp3BitrateLabel := widget.NewLabel(locales.Translate("convert.configpar.bitrate"))
	mp3SampleRateLabel := widget.NewLabel(locales.Translate("convert.configpar.samplerate"))
	m.mp3SettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, mp3BitrateLabel, m.selectMP3Bitrate),
		container.NewGridWithColumns(2, mp3SampleRateLabel, m.selectMP3SampleRate),
	)

	flacCompressionLabel := widget.NewLabel(locales.Translate("convert.configpar.compress"))
	flacSampleRateLabel := widget.NewLabel(locales.Translate("convert.configpar.samplerate"))
	flacBitDepthLabel := widget.NewLabel(locales.Translate("convert.configpar.bitdepth"))
	m.flacSettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, flacCompressionLabel, m.selectFLACCompression),
		container.NewGridWithColumns(2, flacSampleRateLabel, m.selectFLACSampleRate),
		container.NewGridWithColumns(2, flacBitDepthLabel, m.selectFLACBitDepth),
	)

	wavSampleRateLabel := widget.NewLabel(locales.Translate("convert.configpar.samplerate"))
	wavBitDepthLabel := widget.NewLabel(locales.Translate("convert.configpar.bitdepth"))
	m.wavSettingsContainer = container.NewVBox(
		container.NewGridWithColumns(2, wavSampleRateLabel, m.selectWAVSampleRate),
		container.NewGridWithColumns(2, wavBitDepthLabel, m.selectWAVBitDepth),
	)

	// Main format settings container (will hold the appropriate settings based on selected format)
	m.formatSettingsContainer = container.NewVBox()

	// Submit button
	m.submitButton = common.CreateSubmitButton(
		locales.Translate("convert.button.start"),
		func() {
			m.ClearStatusMessages()
			go m.startConversion()
		},
	)
}

// onSourceFormatChanged handles changes in source format selection
func (m *MusicConverterModule) onSourceFormatChanged(format string) {
	// Debug log
	m.debugLog("Source format changed to: '%s'", format)

	// Save configuration
	m.SaveConfig()
}

// onTargetFormatChanged handles changes in target format selection
func (m *MusicConverterModule) onTargetFormatChanged(format string) {
	// Debug log
	m.debugLog("Target format changed to: '%s'", format)

	// Update format settings container
	m.updateFormatSettings(format)

	// Save configuration
	m.SaveConfig()
}

// updateFormatSettings updates the format settings container based on the selected target format
func (m *MusicConverterModule) updateFormatSettings(format string) {
	// Safety check - if containers are not initialized yet, return
	if m.formatSettingsContainer == nil {
		m.debugLog("WARNING: Format settings container is nil")
		return
	}

	// Clear current content
	m.formatSettingsContainer.Objects = []fyne.CanvasObject{}
	m.currentTargetFormat = format

	// Debug log
	m.debugLog("Updating format settings for: '%s'", format)

	// Exact string comparison for format types
	switch format {
	case "MP3":
		if m.mp3SettingsContainer != nil {
			m.formatSettingsContainer.Add(m.mp3SettingsContainer)
			m.debugLog("Adding MP3 settings container")
		} else {
			m.debugLog("WARNING: MP3 settings container is nil")
		}
	case "FLAC":
		if m.flacSettingsContainer != nil {
			m.formatSettingsContainer.Add(m.flacSettingsContainer)
			m.debugLog("Adding FLAC settings container")
		} else {
			m.debugLog("WARNING: FLAC settings container is nil")
		}
	case "WAV":
		if m.wavSettingsContainer != nil {
			m.formatSettingsContainer.Add(m.wavSettingsContainer)
			m.debugLog("Adding WAV settings container")
		} else {
			m.debugLog("WARNING: WAV settings container is nil")
		}
	default:
		// No format selected or unsupported format
		m.formatSettingsContainer.Add(widget.NewLabel(locales.Translate("convert.formatsel.default")))
		m.debugLog("Using default settings container")
	}

	// Force refresh of the container
	m.formatSettingsContainer.Refresh()
}

// LoadConfig loads module configuration
func (m *MusicConverterModule) LoadConfig(cfg common.ModuleConfig) {
	m.IsLoadingConfig = true
	defer func() { m.IsLoadingConfig = false }()

	// Load source and target folder paths
	if m.entrySourceFolder != nil {
		if sourceFolder := cfg.Get("source_folder", ""); sourceFolder != "" {
			m.entrySourceFolder.SetText(sourceFolder)
		}
	}

	if m.entryTargetFolder != nil {
		if targetFolder := cfg.Get("target_folder", ""); targetFolder != "" {
			m.entryTargetFolder.SetText(targetFolder)
		}
	}

	// Load format selections
	if m.selectSourceFormat != nil {
		if sourceFormat := cfg.Get("source_format", ""); sourceFormat != "" {
			m.selectSourceFormat.SetSelected(sourceFormat)
		} else {
			m.selectSourceFormat.SetSelected(locales.Translate("convert.srcformats.all"))
		}
	}

	if m.selectTargetFormat != nil {
		if targetFormat := cfg.Get("target_format", ""); targetFormat != "" {
			m.selectTargetFormat.SetSelected(targetFormat)
			// Aktualizujeme panel s parametry podle vybranu00e9ho formu00e1tu
			m.updateFormatSettings(targetFormat)
		} else {
			m.selectTargetFormat.SetSelected("MP3")
			// Aktualizujeme panel s parametry pro MP3
			m.updateFormatSettings("MP3")
		}
	}

	// Load checkboxes
	if m.checkboxRewriteExisting != nil {
		m.checkboxRewriteExisting.SetChecked(cfg.GetBool("rewrite_existing", false))
	}
	if m.checkboxMakeTargetFolder != nil {
		m.checkboxMakeTargetFolder.SetChecked(cfg.GetBool("make_target_folder", false))
	}

	// Load format-specific settings
	// MP3
	if m.selectMP3Bitrate != nil {
		if mp3Bitrate := cfg.Get("mp3_bitrate", ""); mp3Bitrate != "" {
			m.selectMP3Bitrate.SetSelected(mp3Bitrate)
		}
	}

	if m.selectMP3SampleRate != nil {
		if mp3SampleRate := cfg.Get("mp3_samplerate", ""); mp3SampleRate != "" {
			m.selectMP3SampleRate.SetSelected(mp3SampleRate)
		}
	}

	// FLAC
	if m.selectFLACCompression != nil {
		if flacCompression := cfg.Get("flac_compression", ""); flacCompression != "" {
			m.selectFLACCompression.SetSelected(flacCompression)
		}
	}

	if m.selectFLACSampleRate != nil {
		if flacSampleRate := cfg.Get("flac_samplerate", ""); flacSampleRate != "" {
			m.selectFLACSampleRate.SetSelected(flacSampleRate)
		}
	}

	if m.selectFLACBitDepth != nil {
		if flacBitDepth := cfg.Get("flac_bitdepth", ""); flacBitDepth != "" {
			m.selectFLACBitDepth.SetSelected(flacBitDepth)
		}
	}

	// WAV
	if m.selectWAVSampleRate != nil {
		if wavSampleRate := cfg.Get("wav_samplerate", ""); wavSampleRate != "" {
			m.selectWAVSampleRate.SetSelected(wavSampleRate)
		}
	}
	if m.selectWAVBitDepth != nil {
		if wavBitDepth := cfg.Get("wav_bitdepth", ""); wavBitDepth != "" {
			m.selectWAVBitDepth.SetSelected(wavBitDepth)
		}
	}

	// Ensure metadata map is loaded
	if m.metadataMap == nil {
		var err error
		m.metadataMap, err = m.loadMetadataMap()
		if err != nil {
			m.debugLog("ERROR: Failed to load metadata map: %v", err)
		}
	}
}

// SaveConfig saves the current module configuration
func (m *MusicConverterModule) SaveConfig() common.ModuleConfig {
	cfg := common.ModuleConfig{Extra: make(map[string]string)}

	// Save source and target folder paths
	if m.entrySourceFolder != nil {
		cfg.Set("source_folder", m.entrySourceFolder.Text)
	}
	if m.entryTargetFolder != nil {
		cfg.Set("target_folder", m.entryTargetFolder.Text)
	}

	// Save format selections
	if m.selectSourceFormat != nil {
		cfg.Set("source_format", m.selectSourceFormat.Selected)
	}
	if m.selectTargetFormat != nil {
		cfg.Set("target_format", m.selectTargetFormat.Selected)
	}

	// Save checkboxes
	if m.checkboxRewriteExisting != nil {
		cfg.SetBool("rewrite_existing", m.checkboxRewriteExisting.Checked)
	}
	if m.checkboxMakeTargetFolder != nil {
		cfg.SetBool("make_target_folder", m.checkboxMakeTargetFolder.Checked)
	}

	// Save format-specific settings
	// MP3
	if m.selectMP3Bitrate != nil {
		cfg.Set("mp3_bitrate", m.selectMP3Bitrate.Selected)
	}

	if m.selectMP3SampleRate != nil {
		cfg.Set("mp3_samplerate", m.selectMP3SampleRate.Selected)
	}

	// FLAC
	if m.selectFLACCompression != nil {
		cfg.Set("flac_compression", m.selectFLACCompression.Selected)
	}

	if m.selectFLACSampleRate != nil {
		cfg.Set("flac_samplerate", m.selectFLACSampleRate.Selected)
	}

	if m.selectFLACBitDepth != nil {
		cfg.Set("flac_bitdepth", m.selectFLACBitDepth.Selected)
	}

	// WAV
	if m.selectWAVSampleRate != nil {
		cfg.Set("wav_samplerate", m.selectWAVSampleRate.Selected)
	}
	if m.selectWAVBitDepth != nil {
		cfg.Set("wav_bitdepth", m.selectWAVBitDepth.Selected)
	}

	// Store to config manager
	m.ConfigMgr.SaveModuleConfig(m.GetConfigName(), cfg)
	return cfg
}

// IsCancelled returns whether the current operation has been cancelled
func (m *MusicConverterModule) IsCancelled() bool {
	return m.isCancelled
}

// startConversion begins the conversion process
func (m *MusicConverterModule) startConversion() {
	// Check if already converting
	if m.isConverting {
		return
	}
	m.isConverting = true
	m.isCancelled = false
	defer func() { m.isConverting = false }()

	// Disable the button during processing and set icon after completion
	m.submitButton.Disable()
	defer func() {
		m.submitButton.Enable()
		m.submitButton.SetIcon(theme.ConfirmIcon())
	}()

	// Clear previous status messages
	m.ClearStatusMessages()

	// Validate inputs
	sourceFolder := m.entrySourceFolder.Text
	targetFolder := m.entryTargetFolder.Text
	targetFormat := m.selectTargetFormat.Selected

	if sourceFolder == "" {
		m.ShowError(errors.New(locales.Translate("convert.err.nosource")))
		return
	}

	if targetFolder == "" {
		m.ShowError(errors.New(locales.Translate("convert.err.notarget")))
		return
	}

	if targetFormat == "" {
		m.ShowError(errors.New(locales.Translate("convert.err.noformat")))
		return
	}

	// Get format-specific settings
	formatSettings := make(map[string]string)

	switch targetFormat {
	case "MP3":
		// MP3 settings
		bitrate := m.selectMP3Bitrate.Selected
		sampleRateSetting := m.selectMP3SampleRate.Selected
		formatSettings["bitrate"] = bitrate
		formatSettings["sample_rate"] = sampleRateSetting
	case "FLAC":
		// FLAC settings
		compression := m.selectFLACCompression.Selected
		sampleRate := m.selectFLACSampleRate.Selected
		bitDepth := m.selectFLACBitDepth.Selected
		formatSettings["compression"] = compression
		formatSettings["sample_rate"] = sampleRate
		formatSettings["bit_depth"] = bitDepth
	case "WAV":
		// WAV settings
		sampleRate := m.selectWAVSampleRate.Selected
		bitDepth := m.selectWAVBitDepth.Selected
		formatSettings["sample_rate"] = sampleRate
		formatSettings["bit_depth"] = bitDepth
	}

	// Check if target folder exists, create if needed and option is selected
	if m.checkboxMakeTargetFolder.Checked {
		// Create target folder if it doesn't exist
		if _, err := os.Stat(targetFolder); os.IsNotExist(err) {
			err := os.MkdirAll(targetFolder, 0755)
			if err != nil {
				m.ShowError(fmt.Errorf(locales.Translate("convert.err.createfolder"), err))
				return
			}
			m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.foldercreated"), targetFolder))
		}
	} else {
		// Check if target folder exists
		if _, err := os.Stat(targetFolder); os.IsNotExist(err) {
			m.ShowError(errors.New(locales.Translate("convert.err.nofolder")))
			return
		}
	}

	// Show progress dialog
	m.ShowProgressDialog(locales.Translate("convert.dialog.header"))

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
	// Ensure metadata map is loaded
	if m.metadataMap == nil {
		var err error
		m.metadataMap, err = m.loadMetadataMap()
		if err != nil {
			m.AddErrorMessage(fmt.Sprintf(locales.Translate("convert.err.nomaploaded"), err))
			m.CompleteProgressDialog() // Mark as completed instead of closing
			return
		}
	}

	// Find all audio files in the source folder
	files, err := m.findAudioFiles(sourceFolder, m.selectSourceFormat.Selected)
	if err != nil {
		m.AddErrorMessage(fmt.Sprintf(locales.Translate("convert.err.nosourcefiles"), err))
		m.CompleteProgressDialog() // Mark as completed instead of closing
		return
	}

	if len(files) == 0 {
		m.AddErrorMessage(locales.Translate("convert.err.nosourcefiles"))
		m.CompleteProgressDialog() // Mark as completed instead of closing
		return
	}

	m.AddInfoMessage(fmt.Sprintf(locales.Translate("common.status.filesfound"), len(files)))

	// Track conversion statistics
	successCount := 0
	skippedCount := 0
	failedFiles := []string{}

	// Process each file
	for i, file := range files {
		// Check if cancelled
		if m.IsCancelled() {
			m.AddWarningMessage(locales.Translate("convert.dialog.stop"))
			m.CompleteProgressDialog() // Mark as completed instead of closing
			return
		}

		// Update progress
		progress := float64(i) / float64(len(files))
		statusText := fmt.Sprintf(locales.Translate("convert.status.progress"), i+1, len(files))
		m.UpdateProgressStatus(progress, statusText)

		// Get relative path from source folder
		relPath, err := filepath.Rel(sourceFolder, file)
		if err != nil {
			m.AddWarningMessage(fmt.Sprintf("Error getting relative path for %s: %v", file, err))
			failedFiles = append(failedFiles, file)
			continue
		}

		// Determine target path
		targetPath := targetFolder
		if m.checkboxMakeTargetFolder.Checked {
			// Create target folder if it doesn't exist
			sourceFolderBase := filepath.Base(sourceFolder)
			targetPath = filepath.Join(targetFolder, sourceFolderBase)

			// Ensure target directory exists
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				m.AddErrorMessage(fmt.Sprintf(locales.Translate("convert.err.createfolder"), err))
				m.CompleteProgressDialog() // Mark as completed instead of closing
				return
			}
		}

		// Get directory part of relative path
		relDir := filepath.Dir(relPath)
		if relDir != "." {
			targetPath = filepath.Join(targetPath, relDir)
			// Create subdirectories in target
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				m.AddWarningMessage(fmt.Sprintf("Error creating directory %s: %v", targetPath, err))
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
			// No format selected or unsupported format
			m.AddWarningMessage(fmt.Sprintf("Unsupported target format: %s", targetFormat))
			failedFiles = append(failedFiles, file)
			continue
		}

		// Full target file path
		targetFile := filepath.Join(targetPath, fileNameWithoutExt+targetExt)

		// Check if target file exists and if we should skip it
		if _, err := os.Stat(targetFile); err == nil && !m.checkboxRewriteExisting.Checked {
			m.AddInfoMessage(fmt.Sprintf("Skipping existing file: %s", filepath.Base(targetFile)))
			skippedCount++
			continue
		}

		// Extract metadata from source file using ffprobe
		metadata, err := m.extractMetadata(file)
		if err != nil {
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.err.readmeta"), err))
			failedFiles = append(failedFiles, file)
			continue
		}

		// Add debug info for metadata
		m.debugLog("Input #0, %s, from '%s':", filepath.Ext(file)[1:], file)
		m.debugLog("Metadata:")
		for key, value := range metadata {
			m.debugLog("  %s\t: %s", key, value)
		}

		// Convert file with ffmpeg
		bitDepth, sampleRate, err := m.getAudioProperties(file)
		if err != nil {
			m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.err.readprops"), err))
			failedFiles = append(failedFiles, file)
			continue
		}

		err = m.convertFile(file, targetFile, targetFormat, formatSettings, metadata, bitDepth, sampleRate, m.metadataMap)
		if err != nil {
			m.AddErrorMessage(fmt.Sprintf(locales.Translate("convert.err.duringconv"), err))
			failedFiles = append(failedFiles, file)
			continue
		}

		successCount++
	}

	// Complete the process
	m.UpdateProgressStatus(1.0, fmt.Sprintf(locales.Translate("convert.status.done"), successCount, len(files)))
	m.AddInfoMessage(fmt.Sprintf(locales.Translate("convert.status.doneall"), successCount))

	// Report skipped files
	if skippedCount > 0 {
		m.AddInfoMessage(fmt.Sprintf("Skipped %d existing files", skippedCount))
	}

	// Report failed files
	if len(failedFiles) > 0 {
		m.AddWarningMessage(fmt.Sprintf(locales.Translate("convert.status.unsuppcount"), len(failedFiles)))
		m.AddWarningMessage(locales.Translate("convert.status.unsuppfiles"))
		for _, file := range failedFiles {
			m.AddWarningMessage(fmt.Sprintf("  - %s", filepath.Base(file)))
		}
	}

	// Mark progress dialog as completed
	m.CompleteProgressDialog()
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
		sampleRateSetting := formatSettings["sample_rate"]
		bitDepthSetting := formatSettings["bit_depth"]

		// If bit depth is not set or set to "copy", use bit depth from source file
		if bitDepthSetting == "" || bitDepthSetting == locales.Translate("convert.configpar.copypar") {
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
			switch bitDepthSetting {
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
	}

	// Add metadata arguments based on the mapping
	var targetMetadataMap map[string]string

	switch targetFormat {
	case "MP3":
		targetMetadataMap = metadataMap.InternalToMP3
	case "FLAC":
		targetMetadataMap = metadataMap.InternalToFLAC
	case "WAV":
		targetMetadataMap = metadataMap.InternalToWAV
	}

	// Map metadata from source to target format
	for internalName, targetField := range targetMetadataMap {
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
			// If value contains spaces or special characters, we need to wrap it in quotes
			escapedValue := foundValue

			// Escape special characters
			escapedValue = strings.ReplaceAll(escapedValue, "\\", "\\\\")
			escapedValue = strings.ReplaceAll(escapedValue, "\"", "\\\"")

			// If value contains spaces or special characters, wrap it in quotes
			if strings.ContainsAny(escapedValue, " \t\n\r\"'=;") {
				escapedValue = "\"" + escapedValue + "\""
			}

			args = append(args, "-metadata", fmt.Sprintf("%s=%s", targetField, escapedValue))
		}
	}

	// Add output file path
	args = append(args, targetPath)

	// Log the full command for debugging
	cmdStr := "tools/ffmpeg.exe "
	for _, arg := range args {
		cmdStr += fmt.Sprintf("%s ", arg)
	}
	m.debugLog("DEBUG: Executing command: %s", cmdStr)

	// Create a batch file to run the command
	tmpBatFile := filepath.Join(os.TempDir(), "ffmpeg_convert.bat")

	// Create batch file content with quotes around paths and properly separated parameters
	cmdArgs := make([]string, len(args))
	for i, arg := range args {
		// Add quotes only around file paths (first and last argument)
		// and around metadata values that already contain quotes
		if i == 1 || i == len(args)-1 || (i > 0 && strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"")) {
			// If argument already contains quotes, don't add them again
			if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") {
				cmdArgs[i] = arg
			} else {
				cmdArgs[i] = fmt.Sprintf("\"%s\"", arg)
			}
		} else if i > 0 && strings.Contains(arg, "=") {
			// If it's a metadata (contains '='), we need to ensure the value after '=' is in quotes if it contains spaces
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 && strings.ContainsAny(parts[1], " \t\n\r") &&
				!strings.HasPrefix(parts[1], "\"") && !strings.HasSuffix(parts[1], "\"") {
				cmdArgs[i] = fmt.Sprintf("%s=\"%s\"", parts[0], parts[1])
			} else {
				cmdArgs[i] = arg
			}
		} else {
			cmdArgs[i] = arg
		}
	}

	// Create batch file content with UTF-8 encoding
	cmdContent := fmt.Sprintf("@echo off\r\nchcp 65001 >nul\r\ntools\\ffmpeg.exe %s\r\n", strings.Join(cmdArgs, " "))
	m.debugLog("DEBUG: Writing batch file: %s", tmpBatFile)
	m.debugLog("DEBUG: Batch content: %s", cmdContent)

	if err := os.WriteFile(tmpBatFile, []byte(cmdContent), 0644); err != nil {
		return fmt.Errorf("failed to create batch file: %v", err)
	}

	// Execute the batch file
	cmd := exec.Command("cmd", "/c", tmpBatFile)

	// Capture stderr for error reporting
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}

	// Clean up the temporary batch file
	os.Remove(tmpBatFile)

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

	// Debug output
	m.debugLog("DEBUG: Loading metadata map CSV, size: %d bytes", len(csvContent))

	// Create a new CSV reader from the content
	reader := csv.NewReader(bytes.NewReader(csvContent))

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		m.debugLog("DEBUG: Error reading CSV header: %v", err)
		return nil, err
	}

	// Debug output
	m.debugLog("DEBUG: CSV header: %v", header)

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

	// Debug output
	m.debugLog("DEBUG: Column indices - MP3: %d, FLAC: %d, WAV: %d", mpIndex, flacIndex, wavIndex)

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
			m.debugLog("DEBUG: Error reading CSV row: %v", err)
			return nil, err
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

	// Debug output
	m.debugLog("DEBUG: Loaded metadata mappings - MP3: %d, FLAC: %d, WAV: %d",
		len(result.InternalToMP3), len(result.InternalToFLAC), len(result.InternalToWAV))

	return result, nil
}

// findAudioFiles recursively finds all audio files in the given directory
// If sourceFormat is specified (not "All"), only files of that format are returned
func (m *MusicConverterModule) findAudioFiles(dir string, sourceFormat string) ([]string, error) {
	var files []string

	// Debug output - pouze do konzole
	m.debugLog("DEBUG: Finding audio files in '%s', filter: '%s'", dir, sourceFormat)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

	// Debug output - pouze do konzole
	m.debugLog("DEBUG: Found %d audio files matching format '%s'", len(files), sourceFormat)

	return files, err
}

// extractMetadata extracts metadata from an audio file using ffprobe
func (m *MusicConverterModule) extractMetadata(filePath string) (map[string]string, error) {
	cmd := exec.Command("tools/ffprobe.exe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)

	// Log the command for debugging - pouze do konzole
	m.debugLog("DEBUG: Executing ffprobe: tools/ffprobe.exe -v quiet -print_format json -show_format \"%s\"", filePath)

	// Get command output
	output, err := cmd.Output()
	if err != nil {
		m.debugLog("DEBUG: ffprobe error: %v", err)
		return nil, err
	}

	// Parse JSON output
	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		m.debugLog("DEBUG: JSON parse error: %v", err)
		return nil, err
	}

	// Debug output - pouze do konzole
	m.debugLog("DEBUG: Extracted metadata:")
	for k, v := range result.Format.Tags {
		m.debugLog("DEBUG:   %s: %s", k, v)
	}

	return result.Format.Tags, nil
}

// getAudioProperties extracts audio properties (bit depth, sample rate) from a file using ffprobe
func (m *MusicConverterModule) getAudioProperties(filePath string) (bitDepth string, sampleRate string, err error) {
	cmd := exec.Command("tools/ffprobe.exe", "-v", "quiet", "-print_format", "json", "-show_streams", filePath)

	// Log the command for debugging - pouze do konzole
	m.debugLog("DEBUG: Executing ffprobe for audio properties: tools/ffprobe.exe -v quiet -print_format json -show_streams \"%s\"", filePath)

	// Get command output
	output, err := cmd.Output()
	if err != nil {
		m.debugLog("DEBUG: ffprobe error when getting audio properties: %v", err)
		return bitDepth, sampleRate, err
	}

	// Parse JSON output
	var result struct {
		Streams []struct {
			CodecType   string `json:"codec_type"`
			SampleRate  string `json:"sample_rate"`
			SampleFmt   string `json:"sample_fmt"`
			BitsPerRaw  string `json:"bits_per_raw_sample"`
			BitsPerSamp string `json:"bits_per_sample"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		m.debugLog("DEBUG: JSON parse error for audio properties: %v", err)
		return bitDepth, sampleRate, err
	}

	// Find the audio stream
	for _, stream := range result.Streams {
		if stream.CodecType == "audio" {
			// Get sample rate
			sampleRate = stream.SampleRate

			// Try to determine bit depth
			if stream.BitsPerRaw != "" {
				bitDepth = stream.BitsPerRaw
			} else if stream.BitsPerSamp != "" {
				bitDepth = stream.BitsPerSamp
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

			// Log the detected properties - pouze do konzole
			m.debugLog("DEBUG: Detected audio properties - Bit Depth: %s, Sample Rate: %s", bitDepth, sampleRate)
			return bitDepth, sampleRate, nil
		}
	}

	// If we get here, we couldn't find an audio stream - pouze do konzole
	m.debugLog("DEBUG: No audio stream found in the file")
	return bitDepth, sampleRate, nil
}

// SetDefaultConfig sets the default configuration values for the module
func (m *MusicConverterModule) SetDefaultConfig() common.ModuleConfig {
	// Create new config
	cfg := common.NewModuleConfig()

	// Set default source and target folders to empty strings
	cfg.Set("source_folder", "")
	cfg.Set("target_folder", "")

	// Set default formats
	// cfg.Set("source_format", locales.Translate("convert.srcformats.all"))
	cfg.Set("source_format", "")
	cfg.Set("target_format", "")

	// Set default checkboxes
	cfg.SetBool("rewrite_existing", false)
	cfg.SetBool("make_target_folder", false)

	// Set default MP3 settings
	cfg.Set("mp3_bitrate", "320 kbps")
	cfg.Set("mp3_samplerate", locales.Translate("convert.configpar.copypar"))

	// Set default FLAC settings
	cfg.Set("flac_compression", locales.Translate("convert.configpar.compressfull"))
	cfg.Set("flac_samplerate", locales.Translate("convert.configpar.copypar"))
	cfg.Set("flac_bitdepth", locales.Translate("convert.configpar.copypar"))

	// Set default WAV settings
	cfg.Set("wav_samplerate", locales.Translate("convert.configpar.copypar"))
	cfg.Set("wav_bitdepth", locales.Translate("convert.configpar.copypar"))

	return cfg
}

// debugLog prints debug messages using the module's logger
func (m *MusicConverterModule) debugLog(format string, args ...interface{}) {
	m.Logger.Debug(format, args...)
}
