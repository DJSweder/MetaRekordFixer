package modules

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"MetaRekordFixer/assets"
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
)

// MusicConverterModule handles audio file format conversion functionality.
// It provides a GUI interface for converting audio files between different formats
// while preserving metadata and allowing custom format-specific settings.
type MusicConverterModule struct {
	*common.ModuleBase
	configMgr                *common.ConfigManager
	sourceFormat             *widget.Select               // Source audio format selector
	sourceFolderEntry        *widget.Entry                // Source folder path input
	sourceBrowse             *widget.Button               // Source folder browse button
	targetFormat             *widget.Select               // Target audio format selector
	targetFolderEntry        *widget.Entry                // Target folder path input
	targetBrowse             *widget.Button               // Target folder browse button
	sourceContainer          *fyne.Container              // Container for source selection UI
	targetContainer          *fyne.Container              // Container for target selection UI
	makeTargetFolderCheckbox *widget.Check                // Option to create target subfolder
	rewriteFilesCheckbox     *widget.Check                // Option to overwrite existing files
	stopConversion           bool                         // Flag to stop ongoing conversion
	metadataMap              map[string]map[string]string // Metadata mapping configuration

	// Format-specific UI containers and controls
	mp3Container            *fyne.Container // Container for MP3 format settings
	flacContainer           *fyne.Container // Container for FLAC format settings
	wavContainer            *fyne.Container // Container for WAV format settings
	mp3Bitrate              *widget.Select  // MP3 bitrate selector
	mp3SampleRate           *widget.Select  // MP3 sample rate selector
	flacCompression         *widget.Select  // FLAC compression level selector
	flacSampleRate          *widget.Select  // FLAC sample rate selector
	flacBitDepth            *widget.Select  // FLAC bit depth selector
	wavSampleRate           *widget.Select  // WAV sample rate selector
	wavBitDepth             *widget.Select  // WAV bit depth selector
	formatSettingsContainer *fyne.Container // Main container for format settings
	formatLabels            map[string]*widget.Button
	mainContent             *fyne.Container // Main scrollable container
	isLoadingConfig         bool
}

// NewMusicConverterModule creates and initializes a new music converter module.
// It sets up the UI components, loads initial configuration, and prepares the module for use.
func NewMusicConverterModule(window fyne.Window, configMgr *common.ConfigManager, errorHandler *common.ErrorHandler) *MusicConverterModule {
	m := &MusicConverterModule{
		ModuleBase: common.NewModuleBase(window, configMgr, errorHandler),
		configMgr:  configMgr,
	}

	// Initialize format settings
	m.initializeFormatSettings()

	// Initialize UI components first
	m.initializeUI()

	// Then load configuration
	m.LoadConfig(m.ConfigMgr.GetModuleConfig(m.GetConfigName()))

	return m
}

// GetName returns the localized module name
func (m *MusicConverterModule) GetName() string {
	return locales.Translate("convert.mod.name")
}

// GetConfigName returns the module's configuration identifier
func (m *MusicConverterModule) GetConfigName() string {
	return "music_converter"
}

// GetIcon returns the module's icon resource
func (m *MusicConverterModule) GetIcon() fyne.Resource {
	return theme.FileAudioIcon()
}

// GetContent returns the module content
func (m *MusicConverterModule) GetContent() fyne.CanvasObject {
	return m.mainContent
}

// initializeFormatSettings sets up the format-specific UI components
func (m *MusicConverterModule) initializeFormatSettings() {
	// Ensure all UI components are initialized before using them
	if m.mp3Bitrate == nil {
		m.mp3Bitrate = widget.NewSelect([]string{
			"128k", "192k", "256k", "320k",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.mp3Bitrate.SetSelected("320k")
	}

	if m.mp3SampleRate == nil {
		m.mp3SampleRate = widget.NewSelect([]string{
			locales.Translate("convert.configpar.copypar"),
			"44,1 kHz", "48 kHz", "96 kHz", "192 kHz",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.mp3SampleRate.SetSelected(locales.Translate("convert.configpar.copypar"))
	}

	if m.flacCompression == nil {
		m.flacCompression = widget.NewSelect([]string{
			"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.flacCompression.SetSelected("12")
	}

	if m.flacSampleRate == nil {
		m.flacSampleRate = widget.NewSelect([]string{
			locales.Translate("convert.configpar.copypar"),
			"44,1 kHz", "48 kHz", "96 kHz", "192 kHz",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.flacSampleRate.SetSelected(locales.Translate("convert.configpar.copypar"))
	}

	if m.flacBitDepth == nil {
		m.flacBitDepth = widget.NewSelect([]string{
			locales.Translate("convert.configpar.copypar"),
			"16", "24", "32",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.flacBitDepth.SetSelected(locales.Translate("convert.configpar.copypar"))
	}

	if m.wavSampleRate == nil {
		m.wavSampleRate = widget.NewSelect([]string{
			locales.Translate("convert.configpar.copypar"),
			"44,1 kHz", "48 kHz", "96 kHz", "192 kHz",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.wavSampleRate.SetSelected(locales.Translate("convert.configpar.copypar"))
	}

	if m.wavBitDepth == nil {
		m.wavBitDepth = widget.NewSelect([]string{
			locales.Translate("convert.configpar.copypar"),
			"16", "24", "32",
		}, func(s string) {
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		})
		m.wavBitDepth.SetSelected(locales.Translate("convert.configpar.copypar"))
	}

	// Initialize MP3 container
	mp3BitrateForm := widget.NewFormItem(locales.Translate("convert.configpar.bitrate"), m.mp3Bitrate)
	mp3SampleRateForm := widget.NewFormItem(locales.Translate("convert.configpar.samplerate"), m.mp3SampleRate)
	mp3Form := &widget.Form{
		Items: []*widget.FormItem{
			mp3BitrateForm,
			mp3SampleRateForm,
		},
	}
	m.mp3Container = container.NewVBox(mp3Form)

	// Initialize FLAC container
	flacCompressionForm := widget.NewFormItem(locales.Translate("convert.configpar.compress"), m.flacCompression)
	flacSampleRateForm := widget.NewFormItem(locales.Translate("convert.configpar.samplerate"), m.flacSampleRate)
	flacBitDepthForm := widget.NewFormItem(locales.Translate("convert.configpar.bitdepth"), m.flacBitDepth)
	flacForm := &widget.Form{
		Items: []*widget.FormItem{
			flacCompressionForm,
			flacSampleRateForm,
			flacBitDepthForm,
		},
	}
	m.flacContainer = container.NewVBox(flacForm)

	// Initialize WAV container
	wavSampleRateForm := widget.NewFormItem(locales.Translate("convert.configpar.samplerate"), m.wavSampleRate)
	wavBitDepthForm := widget.NewFormItem(locales.Translate("convert.configpar.bitdepth"), m.wavBitDepth)
	wavForm := &widget.Form{
		Items: []*widget.FormItem{
			wavSampleRateForm,
			wavBitDepthForm,
		},
	}
	m.wavContainer = container.NewVBox(wavForm)

	// Create format setting tabs
	m.formatLabels = make(map[string]*widget.Button)

	m.formatLabels["MP3"] = widget.NewButton("MP3", func() {
		m.updateFormatSettingsVisibility("MP3")
	})
	m.formatLabels["FLAC"] = widget.NewButton("FLAC", func() {
		m.updateFormatSettingsVisibility("FLAC")
	})
	m.formatLabels["WAV"] = widget.NewButton("WAV", func() {
		m.updateFormatSettingsVisibility("WAV")
	})

	// Create a container with buttons
	formatButtons := container.NewHBox(
		m.formatLabels["MP3"],
		m.formatLabels["FLAC"],
		m.formatLabels["WAV"],
	)

	// Create a container with settings, all initially hidden
	formatsContainer := container.NewStack(
		m.mp3Container,
		m.flacContainer,
		m.wavContainer,
	)

	// Hide all initially
	m.mp3Container.Hide()
	m.flacContainer.Hide()
	m.wavContainer.Hide()

	// Create a container that will hold both the format buttons and the settings
	m.formatSettingsContainer = container.NewVBox(
		widget.NewLabel(locales.Translate("convert.label.formatsettings")),
		formatButtons,
		formatsContainer,
	)

	// Ensure format settings visibility is correctly updated
	m.updateFormatSettingsVisibility("MP3")
}

// updateFormatSettingsVisibility shows the settings for the selected format and hides the others
func (m *MusicConverterModule) updateFormatSettingsVisibility(format string) {
	// Reset all buttons to default style
	for key, btn := range m.formatLabels {
		if key == format {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}
	}

	// Hide all containers first
	m.mp3Container.Hide()
	m.flacContainer.Hide()
	m.wavContainer.Hide()

	// Show only the selected one
	switch format {
	case "MP3":
		m.mp3Container.Show()
	case "FLAC":
		m.flacContainer.Show()
	case "WAV":
		m.wavContainer.Show()
	}

	// Refresh the container
	m.formatSettingsContainer.Refresh()
}

// initializeUI creates and configures all UI elements for the module.
// It sets up the main layout including format selectors, folder inputs,
// conversion settings, and progress indicators.
func (m *MusicConverterModule) initializeUI() {
	// Module description and separator
	descLabel := widget.NewLabel(locales.Translate("convert.label.info"))
	descLabel.Wrapping = fyne.TextWrapWord
	descLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Initialize source format selector and folder input
	m.sourceFormat = widget.NewSelect([]string{
		locales.Translate("convert.srcformats.all"),
		"MP3", "FLAC", "WAV", "M4A", "AAC", "OGG", "AIFF",
	}, func(s string) {
		if !m.isLoadingConfig {
			m.SaveConfig()
		}
	})
	m.sourceFormat.PlaceHolder = locales.Translate("convert.formatsel.default")
	m.sourceFolderEntry = widget.NewEntry()
	m.sourceFolderEntry.TextStyle = fyne.TextStyle{Monospace: true}
	m.sourceFolderEntry.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	// Create source folder selection field using standardized function
	sourceFolderField := common.CreateFolderSelectionField(
		locales.Translate("convert.wintitle.folderselsource"),
		m.sourceFolderEntry,
		func(path string) {
			m.sourceFolderEntry.SetText(path)
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		},
	)

	// Store the button reference for backward compatibility
	m.sourceBrowse = sourceFolderField.(*fyne.Container).Objects[1].(*widget.Button)

	// Initialize target format selector and folder input
	m.targetFormat = widget.NewSelect([]string{"MP3", "FLAC", "WAV"}, func(s string) {
		m.updateFormatSettingsVisibility(s)
		if !m.isLoadingConfig {
			m.SaveConfig()
		}
	})
	m.targetFormat.PlaceHolder = locales.Translate("convert.formatsel.default")
	m.targetFolderEntry = widget.NewEntry()
	m.targetFolderEntry.TextStyle = fyne.TextStyle{Monospace: true}
	m.targetFolderEntry.OnChanged = m.CreateChangeHandler(func() { m.SaveConfig() })

	// Create target folder selection field using standardized function
	targetFolderField := common.CreateFolderSelectionField(
		locales.Translate("convert.wintitle.folderseltarget"),
		m.targetFolderEntry,
		func(path string) {
			m.targetFolderEntry.SetText(path)
			if !m.isLoadingConfig {
				m.SaveConfig()
			}
		},
	)

	// Store the button reference for backward compatibility
	m.targetBrowse = targetFolderField.(*fyne.Container).Objects[1].(*widget.Button)

	// Create labels for source and target folders
	sourceLabel := widget.NewLabelWithStyle(locales.Translate("convert.label.source"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	targetLabel := widget.NewLabelWithStyle(locales.Translate("convert.label.target"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Create containers for source and target folders
	sourceInputContainer := container.NewBorder(
		nil, nil,
		m.sourceFormat, nil,
		sourceFolderField,
	)

	m.sourceContainer = container.NewVBox(
		sourceLabel,
		sourceInputContainer,
	)

	targetInputContainer := container.NewBorder(
		nil, nil,
		m.targetFormat, nil,
		targetFolderField,
	)

	m.targetContainer = container.NewVBox(
		targetLabel,
		targetInputContainer,
	)

	// Add checkboxes with proper change handlers
	m.makeTargetFolderCheckbox = widget.NewCheck(
		locales.Translate("convert.chkbox.maketargetfolder"),
		m.CreateBoolChangeHandler(func() {
			m.SaveConfig()
		}),
	)
	m.rewriteFilesCheckbox = widget.NewCheck(
		locales.Translate("convert.chkbox.rewrite"),
		m.CreateBoolChangeHandler(func() {
			m.SaveConfig()
		}),
	)

	// Create submit button using standardized function
	var submitBtn *widget.Button
	submitBtn = common.CreateSubmitButtonWithIcon(locales.Translate("convert.button.start"), nil, func() {
		if !m.isLoadingConfig {
			m.SaveConfig()
			m.stopConversion = false
			originalOnTapped := submitBtn.OnTapped
			stopFunc := func() {
				m.stopConversion = true
				submitBtn.SetIcon(nil)
				submitBtn.SetText(locales.Translate("convert.button.start"))
				submitBtn.OnTapped = originalOnTapped
			}
			submitBtn.SetIcon(theme.MediaStopIcon())
			submitBtn.SetText(locales.Translate("convert.button.stop"))
			submitBtn.OnTapped = stopFunc
			go func() {
				err := m.convertFiles(submitBtn, originalOnTapped)
				if err != nil && !m.stopConversion {
					if err.Error() != locales.Translate("convert.status.cancelled") {
						// Create error context with module name and operation
						context := common.NewErrorContext(m.GetConfigName(), "Music Conversion")
						context.Severity = common.ErrorWarning
						m.ErrorHandler.HandleError(err, context, m.Window, m.Status)
					}
					submitBtn.SetIcon(nil)
				} else if !m.stopConversion {
					submitBtn.SetIcon(theme.ConfirmIcon())
					dialog.ShowInformation(
						locales.Translate("convert.dialog.successtitle"),
						locales.Translate("convert.dialog.successmsg"),
						m.Window,
					)
				}
				if m.stopConversion {
					m.Status.SetText(locales.Translate("convert.dialog.stop"))
					submitBtn.SetIcon(nil)
				}
				submitBtn.SetText(locales.Translate("convert.button.start"))
				submitBtn.OnTapped = originalOnTapped
			}()
		}
	})
	submitBtn.Importance = widget.HighImportance

	// Create checkboxes container
	checkboxesContainer := container.NewVBox(
		m.makeTargetFolderCheckbox,
		m.rewriteFilesCheckbox,
	)

	// Create form content with all components
	formContent := container.NewVBox(
		m.sourceContainer,
		m.targetContainer,
		widget.NewSeparator(),
		m.formatSettingsContainer,
		widget.NewSeparator(),
		checkboxesContainer,
	)

	// Additional widgets for the standard layout
	additionalWidgets := []fyne.CanvasObject{
		m.Progress,
		m.Status,
	}

	// Create content container with form and additional widgets
	contentContainer := container.NewVBox(
		formContent,
	)

	// Add additional widgets to content container
	for _, widget := range additionalWidgets {
		contentContainer.Add(widget)
	}

	// Create standard module layout
	mainBox := common.CreateStandardModuleLayout(
		locales.Translate("convert.label.info"),
		contentContainer,
		submitBtn,
	)

	// Create scrollable container with fixed minimum size
	scroll := container.NewVScroll(mainBox)
	scroll.SetMinSize(fyne.NewSize(600, 600))
	// Pack scroll into Stack container for proper expansion
	m.mainContent = container.NewStack(scroll)

	// Initial format visibility update
	if m.targetFormat != nil && m.targetFormat.Selected != "" {
		m.updateFormatSettingsVisibility(m.targetFormat.Selected)
	}
}

// Main conversion logic for all files
func (m *MusicConverterModule) convertFiles(_ *widget.Button, _ func()) error {
	if m.stopConversion {
		return errors.New(locales.Translate("convert.dialog.stop"))
	}

	if m.sourceFolderEntry == nil || m.sourceFolderEntry.Text == "" || m.targetFolderEntry == nil || m.targetFolderEntry.Text == "" {
		return errors.New(locales.Translate("convert.err.nosourcefiles"))
	}

	// Check if metadata map is loaded
	if m.metadataMap == nil {
		if err := m.loadMetadataMap(); err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("convert.err.nomaploaded"), err)
		}
	}

	// Check target folder
	if m.makeTargetFolderCheckbox != nil && m.makeTargetFolderCheckbox.Checked {
		sourceBase := filepath.Base(m.sourceFolderEntry.Text)
		m.targetFolderEntry.SetText(filepath.Join(m.targetFolderEntry.Text, sourceBase))
	}

	m.Status.SetText(locales.Translate("convert.status.starting"))
	m.Progress.SetValue(0)

	ffmpegPath := filepath.Join("tools", "ffmpeg.exe")
	ffprobePath := filepath.Join("tools", "ffprobe.exe")

	var fileList []string
	var unsupportedCount int
	var unsupportedFiles []string

	// Supported formats
	supportedFormats := map[string]bool{
		".flac": true, ".mp3": true, ".wav": true,
		".m4a": true, ".aac": true, ".ogg": true,
		".aiff": true,
	}

	// Collect list of files to convert
	err := filepath.Walk(m.sourceFolderEntry.Text, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if m.sourceFormat != nil && m.sourceFormat.Selected == locales.Translate("convert.srcformats.all") {
			if supportedFormats[ext] {
				fileList = append(fileList, path)
			} else {
				unsupportedCount++
				unsupportedFiles = append(unsupportedFiles, path)
			}
		} else if m.sourceFormat != nil && strings.EqualFold(ext, "."+strings.ToLower(m.sourceFormat.Selected)) {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(fileList) == 0 {
		return errors.New(locales.Translate("convert.err.nosourcefiles"))
	}

	targetFormat := strings.ToLower(m.targetFormat.Selected)
	params := m.getFormatParams(targetFormat)

	var totalDuration float64
	fileDurations := make(map[string]float64)
	for _, path := range fileList {
		dur, err := getAudioDuration(ffprobePath, path)
		if err != nil {
			return fmt.Errorf("%s: %s - %w", locales.Translate("convert.err.noduration"), path, err)
		}
		fileDurations[path] = dur
		totalDuration += dur
	}

	m.Progress.Max = 1.0
	var alreadyProcessedSeconds float64
	processedCount := 0
	totalCount := len(fileList)

	for _, inputPath := range fileList {
		if m.stopConversion {
			break
		}

		outputPath, err := m.buildTargetPath(inputPath, targetFormat)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
			return fmt.Errorf("%s: %w", locales.Translate("convert.err.createfolder"), err)
		}

		processedCount++
		m.Status.SetText(fmt.Sprintf(locales.Translate("convert.status.progress"), processedCount, totalCount))
		if err := m.convertAudioFile(ffmpegPath, inputPath, outputPath, targetFormat,
			params, fileDurations[inputPath], alreadyProcessedSeconds, totalDuration); err != nil {
			return err
		}

		alreadyProcessedSeconds += fileDurations[inputPath]
	}

	// Done
	var statusMessage string
	if processedCount == totalCount {
		statusMessage = fmt.Sprintf(locales.Translate("convert.status.doneall"), totalCount)
	} else {
		statusMessage = fmt.Sprintf(locales.Translate("convert.status.done"), processedCount, totalCount)
	}

	// Add message about unsupported files
	if unsupportedCount > 0 {
		statusMessage += "\n" + fmt.Sprintf(locales.Translate("convert.status.unsuppcount"), unsupportedCount)
		statusMessage += "\n" + locales.Translate("convert.status.unsuppfiles")
		for i, file := range unsupportedFiles {
			fileName := filepath.Base(file)
			statusMessage += fmt.Sprintf("\n%d. %s", i+1, fileName)
		}
	}

	m.Status.SetText(statusMessage)
	return nil
}

// Prepare target path
func (m *MusicConverterModule) buildTargetPath(inputPath, targetFormat string) (string, error) {
	sourceFolder := m.sourceFolderEntry.Text
	targetFolder := m.targetFolderEntry.Text
	relPath, err := filepath.Rel(sourceFolder, inputPath)
	if err != nil {
		return "", err
	}

	if m.makeTargetFolderCheckbox != nil && m.makeTargetFolderCheckbox.Checked {
		sourceBase := filepath.Base(sourceFolder)
		targetFolder = filepath.Join(targetFolder, sourceBase)
	}

	targetPath := filepath.Join(targetFolder, relPath)
	ext := filepath.Ext(targetPath)
	baseName := strings.TrimSuffix(filepath.Base(targetPath), ext)
	newName := baseName + "." + targetFormat
	outputPath := filepath.Join(filepath.Dir(targetPath), newName)

	// If file exists and we are not overwriting, add _copy
	if m.rewriteFilesCheckbox != nil && !m.rewriteFilesCheckbox.Checked {
		if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
			newName = baseName + "_copy." + targetFormat
			outputPath = filepath.Join(filepath.Dir(targetPath), newName)
		}
	}

	return outputPath, nil
}

// getAudioDuration returns the duration of an audio file in seconds using ffprobe.
// It takes the path to ffprobe executable and the input audio file path as parameters.
func getAudioDuration(ffprobePath, inputPath string) (float64, error) {
	cmd := exec.Command(
		ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	durationStr := strings.TrimSpace(out.String())
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}
	return duration, nil
}

// Reads metadata from audio file using ffprobe
func (m *MusicConverterModule) readMetadata(_ string, path string) (map[string]string, error) {
	ffprobePath := filepath.Join("tools", "ffprobe.exe")
	cmd := exec.Command(ffprobePath, "-v", "error", "-print_format", "flat", "-show_entries", "format_tags", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error reading metadata: %v", err)
	}

	metadata := make(map[string]string)
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "format.tags.") {
			line = strings.TrimPrefix(line, "format.tags.")
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				metadata[key] = value
			}
		}
	}

	return metadata, nil
}

// Maps metadata from one format to another using the CSV mapping table
// Note: For MP3 format, ffmpeg automatically handles TXXX frames:...
// - If the tag name matches a standard ID3v2 frame, it will be used directly
// - If the tag name is not a standard frame, it will be automatically wrapped in a TXXX frame
func (m *MusicConverterModule) mapMetadata(sourceFormat, targetFormat string, sourceMetadata map[string]string) map[string]string {
	// First, map source metadata to internal names
	internalMetadata := make(map[string]string)
	// Step 1: Map from source format to internal names
	for sourceKey, value := range sourceMetadata {
		lowerSourceKey := strings.ToLower(sourceKey)
		for internalName, formatMap := range m.metadataMap {
			sourceTag := formatMap[sourceFormat]
			if sourceTag == lowerSourceKey {
				internalMetadata[internalName] = value
				break
			}
		}
	}

	// Step 2: Map from internal names to target format
	targetMetadata := make(map[string]string)
	for internalName, value := range internalMetadata {
		if targetTag := m.metadataMap[internalName][targetFormat]; targetTag != "" {
			targetMetadata[targetTag] = value
		}
	}

	return targetMetadata
}

// Loads metadata mapping from CSV file
// The CSV structure:
// InternalName,MP3,FLAC,WAV
// where:
// - InternalName: normalized name used internally
// - MP3: ID3v2 tag name (without TXXX prefix, ffmpeg handles it automatically)
// - FLAC: FLAC tag name
// - WAV: WAV tag name (if supported)
func (m *MusicConverterModule) loadMetadataMap() error {
	// Use embedded resource instead of file
	reader := csv.NewReader(bytes.NewReader(assets.ResourceMetadataMapCSV.Content()))

	//Skip header
	if _, err := reader.Read(); err != nil {
		return err
	}

	m.metadataMap = make(map[string]map[string]string)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		// Use lowercase internal name as map key
		internalName := strings.ToLower(record[0])
		m.metadataMap[internalName] = map[string]string{
			"mp3":  strings.ToLower(record[1]),
			"flac": strings.ToLower(record[2]),
			"wav":  strings.ToLower(record[3]),
		}
	}

	return nil
}

// Converts audio file using ffmpeg with metadata mapping
func (m *MusicConverterModule) convertAudioFile(
	ffmpegPath string,
	inputPath string,
	outputPath string,
	targetFormat string,
	params map[string]string,
	fileDuration float64,
	alreadyProcessedSeconds float64,
	totalDuration float64,
) error {
	targetFormat = strings.ToLower(targetFormat)
	if m.metadataMap == nil {
		return errors.New(locales.Translate("convert.err.unablecontnomap"))
	}

	// Read source metadata
	sourceFormat := strings.ToLower(filepath.Ext(inputPath)[1:])
	sourceMetadata, err := m.readMetadata(sourceFormat, inputPath)
	if err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("convert.err.readmeta"), err)
	}

	// Map metadata according to CSV mapping
	targetMetadata := m.mapMetadata(sourceFormat, targetFormat, sourceMetadata)

	// Basic ffmpeg parameters
	cmdArgs := []string{
		"-i", inputPath,
		"-map_metadata", "-1", // Clear all existing metadata
		"-write_id3v2", "1", // Enable ID3v2 tags
		"-id3v2_version", "4", // Use ID3v2.4
	}

	// Add metadata parameters in exact order
	for key, value := range targetMetadata {
		cmdArgs = append(cmdArgs, "-metadata", fmt.Sprintf("%s=%s", key, value))
	}

	// Add format-specific parameters
	switch targetFormat {
	case "mp3":
		if bitrate, ok := params["bitrate"]; ok {
			cmdArgs = append(cmdArgs, "-b:a", bitrate)
		}

		if sampleRate, ok := params["sample_rate"]; ok && sampleRate != locales.Translate("convert.configpar.copypar") {
			sampleRate = strings.ReplaceAll(sampleRate, "kHz", "000") // Convert kHz to Hz
			sampleRate = strings.ReplaceAll(sampleRate, ",", "")      // Remove commas
			cmdArgs = append(cmdArgs, "-ar", sampleRate)
		}
	case "flac":
		if compression, ok := params["compression"]; ok {
			cmdArgs = append(cmdArgs, "-compression_level", compression)
		}

		if bitDepthStr, ok := params["bit_depth"]; ok && bitDepthStr != locales.Translate("convert.configpar.copypar") {
			bitDepth, err := strconv.Atoi(bitDepthStr)
			if err == nil {
				cmdArgs = append(cmdArgs, "-sample_fmt", fmt.Sprintf("s%d", bitDepth))
			}
		}

		if sampleRate, ok := params["sample_rate"]; ok && sampleRate != locales.Translate("convert.configpar.copypar") {
			sampleRate = strings.ReplaceAll(sampleRate, "kHz", "000") // Convert kHz to Hz
			sampleRate = strings.ReplaceAll(sampleRate, ",", "")      // Remove commas
			cmdArgs = append(cmdArgs, "-ar", sampleRate)
		}
	case "wav":
		if sampleRate, ok := params["sample_rate"]; ok && sampleRate != locales.Translate("convert.configpar.copypar") {
			sampleRate = strings.ReplaceAll(sampleRate, "kHz", "000") // Convert kHz to Hz
			sampleRate = strings.ReplaceAll(sampleRate, ",", "")      // Remove commas
			cmdArgs = append(cmdArgs, "-ar", sampleRate)
		}

		if bitDepthStr, ok := params["bit_depth"]; ok && bitDepthStr != locales.Translate("convert.configpar.copypar") {
			bitDepth, err := strconv.Atoi(bitDepthStr)
			if err == nil {
				cmdArgs = append(cmdArgs, "-sample_fmt", fmt.Sprintf("s%d", bitDepth))
			}
		}
	}

	// Add output file
	cmdArgs = append(cmdArgs, "-y", outputPath)

	// Execute ffmpeg
	cmd := exec.Command(ffmpegPath, cmdArgs...)

	// Redirect output to null
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", locales.Translate("convert.err.ffmpegstart"), err)
	}

	// Progress bar handling
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		startTime := time.Now()
		for range ticker.C {
			if m.stopConversion {
				cmd.Process.Kill()
				time.Sleep(100 * time.Millisecond)
				os.Remove(outputPath)
				done <- true
				return
			}

			elapsed := time.Since(startTime).Seconds()
			progressFraction := (alreadyProcessedSeconds + elapsed) / totalDuration
			if progressFraction > 1.0 {
				progressFraction = 1.0
			}

			m.Window.Canvas().Refresh(m.Progress)
			m.Progress.SetValue(progressFraction)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if m.stopConversion {
			<-done
			return nil
		}

		return fmt.Errorf("%s: %w", locales.Translate("convert.err.duringconv"), err)
	}

	m.Window.Canvas().Refresh(m.Progress)
	current := alreadyProcessedSeconds + fileDuration
	if current > totalDuration {
		current = totalDuration
	}

	m.Progress.SetValue(current / totalDuration)
	return nil
}

// LoadModuleConfig loads the module configuration from the config manager
func (m *MusicConverterModule) LoadModuleConfig() {
	config := m.configMgr.GetModuleConfig(m.GetConfigName())
	m.LoadConfig(config)
}

// LoadConfig loads the module configuration
func (m *MusicConverterModule) LoadConfig(config common.ModuleConfig) {
	// Ensure UI components are initialized before trying to load config
	if m.sourceFormat == nil || m.targetFormat == nil {
		return
	}

	m.isLoadingConfig = true
	defer func() {
		m.isLoadingConfig = false
		if m.targetFormat != nil && m.targetFormat.Selected != "" {
			m.updateFormatSettingsVisibility(m.targetFormat.Selected)
		}
	}()

	if config.Extra == nil {
		config.Extra = make(map[string]string)
	}

	// Load basic values to configuration only if UI components are initialized
	if m.sourceFolderEntry != nil {
		if sourcePath := config.Extra["source_path"]; sourcePath != "" {
			m.sourceFolderEntry.SetText(sourcePath)
		}
	}

	if m.targetFolderEntry != nil {
		if targetPath := config.Extra["target_path"]; targetPath != "" {
			m.targetFolderEntry.SetText(targetPath)
		}
	}

	// For formats, set Selected directly and call OnChanged handler
	if m.sourceFormat != nil {
		if sourceFormat := config.Extra["source_format"]; sourceFormat != "" {
			m.sourceFormat.Selected = sourceFormat
			if m.sourceFormat.OnChanged != nil {
				m.sourceFormat.OnChanged(sourceFormat)
			}
			m.sourceFormat.Refresh()
		}
	}

	if m.targetFormat != nil {
		if targetFormat := config.Extra["target_format"]; targetFormat != "" {
			m.targetFormat.Selected = targetFormat
			if m.targetFormat.OnChanged != nil {
				m.targetFormat.OnChanged(targetFormat)
			}
			m.targetFormat.Refresh()
		}
	}

	// Load checkboxes with value validation
	if m.makeTargetFolderCheckbox != nil {
		m.makeTargetFolderCheckbox.SetChecked(config.Extra["make_target_folder"] == "true")
	}
	if m.rewriteFilesCheckbox != nil {
		m.rewriteFilesCheckbox.SetChecked(config.Extra["rewrite_files"] == "true")
	}

	// Load format parameters from specific format sections
	// MP3 settings
	if m.mp3Bitrate != nil {
		if bitrate := config.Extra["mp3_bitrate"]; bitrate != "" {
			m.mp3Bitrate.SetSelected(bitrate)
		}
	}
	if m.mp3SampleRate != nil {
		if sampleRate := config.Extra["mp3_sample_rate"]; sampleRate != "" {
			m.mp3SampleRate.SetSelected(sampleRate)
		}
	}

	// FLAC settings
	if m.flacCompression != nil {
		if compression := config.Extra["flac_compression"]; compression != "" {
			m.flacCompression.SetSelected(compression)
		}
	}
	if m.flacSampleRate != nil {
		if sampleRate := config.Extra["flac_sample_rate"]; sampleRate != "" {
			m.flacSampleRate.SetSelected(sampleRate)
		}
	}
	if m.flacBitDepth != nil {
		if bitDepth := config.Extra["flac_bit_depth"]; bitDepth != "" {
			m.flacBitDepth.SetSelected(bitDepth)
		}
	}

	// WAV settings
	if m.wavSampleRate != nil {
		if sampleRate := config.Extra["wav_sample_rate"]; sampleRate != "" {
			m.wavSampleRate.SetSelected(sampleRate)
		}
	}
	if m.wavBitDepth != nil {
		if bitDepth := config.Extra["wav_bit_depth"]; bitDepth != "" {
			m.wavBitDepth.SetSelected(bitDepth)
		}
	}
}

// SaveConfig saves the module configuration
func (m *MusicConverterModule) SaveConfig() common.ModuleConfig {
	// Create a map for configuration data
	configData := map[string]string{}

	// Save basic values to configuration only if UI components are initialized
	if m.sourceFolderEntry != nil {
		configData["source_path"] = filepath.FromSlash(m.sourceFolderEntry.Text)
	}
	if m.targetFolderEntry != nil {
		configData["target_path"] = filepath.FromSlash(m.targetFolderEntry.Text)
	}
	if m.sourceFormat != nil && m.sourceFormat.Selected != "" {
		configData["source_format"] = m.sourceFormat.Selected
	}
	if m.targetFormat != nil && m.targetFormat.Selected != "" {
		configData["target_format"] = m.targetFormat.Selected
	}
	if m.makeTargetFolderCheckbox != nil {
		configData["make_target_folder"] = fmt.Sprintf("%v", m.makeTargetFolderCheckbox.Checked)
	}
	if m.rewriteFilesCheckbox != nil {
		configData["rewrite_files"] = fmt.Sprintf("%v", m.rewriteFilesCheckbox.Checked)
	}

	// Save format-specific parameters
	// MP3 parameters
	if m.mp3Bitrate != nil && m.mp3Bitrate.Selected != "" {
		configData["mp3_bitrate"] = m.mp3Bitrate.Selected
	}
	if m.mp3SampleRate != nil && m.mp3SampleRate.Selected != "" {
		configData["mp3_sample_rate"] = m.mp3SampleRate.Selected
	}

	// FLAC parameters
	if m.flacCompression != nil && m.flacCompression.Selected != "" {
		configData["flac_compression"] = m.flacCompression.Selected
	}
	if m.flacSampleRate != nil && m.flacSampleRate.Selected != "" {
		configData["flac_sample_rate"] = m.flacSampleRate.Selected
	}
	if m.flacBitDepth != nil && m.flacBitDepth.Selected != "" {
		configData["flac_bit_depth"] = m.flacBitDepth.Selected
	}

	// WAV parameters
	if m.wavSampleRate != nil && m.wavSampleRate.Selected != "" {
		configData["wav_sample_rate"] = m.wavSampleRate.Selected
	}
	if m.wavBitDepth != nil && m.wavBitDepth.Selected != "" {
		configData["wav_bit_depth"] = m.wavBitDepth.Selected
	}

	// Save to configuration manager
	moduleConfig := common.ModuleConfig{Extra: configData}
	if m.configMgr != nil {
		m.configMgr.SaveModuleConfig(m.GetConfigName(), moduleConfig)
	}

	return moduleConfig
}

// getFormatParams retrieves format-specific parameters from configuration
func (m *MusicConverterModule) getFormatParams(targetFormat string) map[string]string {
	config := m.configMgr.GetModuleConfig(m.GetConfigName())

	switch targetFormat {
	case "mp3":
		return map[string]string{
			"bitrate":     config.Extra["mp3_bitrate"],
			"sample_rate": config.Extra["mp3_sample_rate"],
		}
	case "flac":
		return map[string]string{
			"compression": config.Extra["flac_compression"],
			"sample_rate": config.Extra["flac_sample_rate"],
			"bit_depth":   config.Extra["flac_bit_depth"],
		}
	case "wav":
		return map[string]string{
			"sample_rate": config.Extra["wav_sample_rate"],
			"bit_depth":   config.Extra["wav_bit_depth"],
		}
	default:
		return map[string]string{}
	}
}

func LogError(err error, message string) {
	log.Printf("Error %s: %v", message, err)
}
