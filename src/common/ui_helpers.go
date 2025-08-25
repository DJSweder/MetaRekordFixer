// common/ui_helpers.go

package common

import (
	"MetaRekordFixer/locales"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	nativedialog "github.com/sqweek/dialog"
)

// ProgressDialog represents a progress dialog with a progress bar and status label
type ProgressDialog struct {
	dialog        *dialog.CustomDialog
	window        fyne.Window
	progressBar   *widget.ProgressBar
	statusLabel   *widget.Label
	stopButton    *widget.Button
	cancelHandler func()
	isCompleted   bool
}

// NewProgressDialog creates a new progress dialog with optional cancel handler
func NewProgressDialog(window fyne.Window, title, initialStatus string, cancelHandler func()) *ProgressDialog {
	pd := &ProgressDialog{
		window:        window,
		progressBar:   widget.NewProgressBar(),
		statusLabel:   widget.NewLabel(initialStatus),
		cancelHandler: cancelHandler,
		isCompleted:   false,
	}

	// Create stop button with square icon
	pd.stopButton = widget.NewButtonWithIcon(locales.Translate("common.button.stop"), theme.MediaStopIcon(), func() {
		if pd.isCompleted {
			// If process is completed, close the dialog
			pd.Hide()
		} else if pd.cancelHandler != nil {
			// If process is running and cancel handler exists, call it
			pd.cancelHandler()
		}
	})
	pd.stopButton.Importance = widget.HighImportance

	// Create and initialize status label
	content := container.NewVBox(pd.progressBar, pd.statusLabel)
	content.Add(container.NewHBox(layout.NewSpacer(), pd.stopButton, layout.NewSpacer()))

	// Set minimum width for the content to ensure dialog has sufficient width for status messages
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(550, 1))
	content.Add(rect)

	// Use NewCustomWithoutButtons to create a dialog without any default buttons
	pd.dialog = dialog.NewCustomWithoutButtons(title, content, window)

	return pd
}

// Show displays the progress dialog
func (pd *ProgressDialog) Show() {
	pd.dialog.Show()
}

// Hide hides the progress dialog
func (pd *ProgressDialog) Hide() {
	pd.dialog.Hide()
}

// UpdateProgress updates the progress bar value
func (pd *ProgressDialog) UpdateProgress(value float64) {
	pd.progressBar.SetValue(value)
}

// UpdateStatus updates the status text
func (pd *ProgressDialog) UpdateStatus(text string) {
	pd.statusLabel.SetText(text)
}

// MarkCompleted marks the process as completed and changes the stop button to OK button
func (pd *ProgressDialog) MarkCompleted() {
	pd.isCompleted = true
	pd.stopButton.SetText(locales.Translate("common.button.ok"))
	pd.stopButton.SetIcon(theme.ConfirmIcon())

}

// ShowError displays an error message and hides the progress dialog
func (pd *ProgressDialog) ShowError(err error) {
	pd.Hide()
	dialog.ShowError(err, pd.window)
}

// CreateNativeFolderBrowseButton creates a standardized folder browse button using native OS dialog
// This is a replacement for CreateFolderBrowseButton that uses native OS dialogs instead of Fyne dialogs
// to avoid issues with folder selection on Windows platforms
func CreateNativeFolderBrowseButton(title string, buttonText string, changeHandler func(string)) *widget.Button {
	return widget.NewButtonWithIcon(buttonText, theme.FolderOpenIcon(), func() {
		dirname, err := nativedialog.Directory().Title(title).Browse()
		if err == nil && dirname != "" {
			if changeHandler != nil {
				changeHandler(dirname)
			}
		}
	})
}

// CreateFolderSelectionField creates a standardized folder selection field with browse button
func CreateFolderSelectionField(title string, entryField *widget.Entry, changeHandler func(string)) fyne.CanvasObject {
	// Create entry field if not provided
	if entryField == nil {
		entryField = widget.NewEntry()
	}

	// Set placeholder using localization key - always set it regardless of whether the entry field is new or existing
	entryField.SetPlaceHolder(locales.Translate("common.entry.placeholderpath"))

	// Set change handler if provided
	if changeHandler != nil {
		entryField.OnChanged = func(value string) {
			changeHandler(value)
		}
	}

	// Create browse button (icon only)
	browseBtn := CreateNativeFolderBrowseButton(
		title,
		"", // Empty text, only icon
		func(path string) {
			entryField.SetText(path)
			if changeHandler != nil {
				changeHandler(path)
			}
		},
	)

	// Create container with entry field and browse button
	return container.NewBorder(nil, nil, nil, browseBtn, entryField)
}

// CreateFolderSelectionFieldWithDelete creates a standardized folder selection field with browse and delete buttons
func CreateFolderSelectionFieldWithDelete(title string, entryField *widget.Entry, changeHandler func(string), deleteHandler func()) fyne.CanvasObject {
	// Create entry field if not provided
	if entryField == nil {
		entryField = widget.NewEntry()
	}

	// Set placeholder using localization key - always set it regardless of whether the entry field is new or existing
	entryField.SetPlaceHolder(locales.Translate("common.entry.placeholderpath"))

	// Set change handler if provided
	if changeHandler != nil {
		entryField.OnChanged = func(value string) {
			changeHandler(value)
		}
	}

	// Create browse button (icon only)
	browseBtn := CreateNativeFolderBrowseButton(
		title,
		"", // Empty text, only icon
		func(path string) {
			entryField.SetText(path)
			if changeHandler != nil {
				changeHandler(path)
			}
		},
	)

	// Create delete button (icon only)
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if deleteHandler != nil {
			deleteHandler()
		}
	})

	// Create container with entry field between delete and browse buttons
	return container.NewBorder(nil, nil, deleteBtn, browseBtn, entryField)
}

// CreateSubmitButton creates a standardized submit button with high importance
// This button is used to start a process or submit a form
func CreateSubmitButton(title string, handler func()) *widget.Button {
	btn := widget.NewButton(title, handler)
	btn.Importance = widget.HighImportance
	return btn
}

// CreateSubmitButtonWithIcon creates a standardized submit button with an icon and high importance
// This button is used after a process or form has been completed or canceled
func CreateSubmitButtonWithIcon(title string, icon fyne.Resource, handler func()) *widget.Button {
	btn := widget.NewButtonWithIcon(title, icon, handler)
	btn.Importance = widget.HighImportance
	return btn
}

// CreateDescriptionLabel creates a standardized description label with wrapping and bold text
// This label is used for module descriptions and other informational text
func CreateDescriptionLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

// CreateCalendarDayButton creates a standardized calendar day button
// This button is used in calendar widgets for day selection
func CreateCalendarDayButton(day int, onSelected func()) *widget.Button {
	btn := widget.NewButton(fmt.Sprintf("%d", day), onSelected)
	btn.Importance = widget.HighImportance
	return btn
}

// ShowStandardError displays a standardized error dialog with log folder access
func ShowStandardError(window fyne.Window, err error, context *ErrorContext) *dialog.CustomDialog {
	// Get header based on severity
	var header string
	if context != nil {
		switch context.Severity {
		case SeverityWarning:
			header = locales.Translate("common.dialog.warningheader")
		case SeverityCritical:
			header = locales.Translate("common.dialog.criticalheader")
		case SeverityError:
			header = locales.Translate("common.dialog.fatalerror")
		default:
			header = locales.Translate("common.dialog.errorheader")
		}
	} else {
		header = locales.Translate("common.dialog.errorheader")
	}

	// Only use localized error message in dialog, without technical details
	var errorMsg string
	if err != nil {
		// If localized string of error message is in format "Some text: %v", we extract first part only (because character ":" is used as separator)
		errParts := strings.SplitN(err.Error(), ":", 2)
		errKey := strings.TrimSpace(errParts[0])
		errorMsg = locales.Translate(errKey)
	} else {
		errorMsg = locales.Translate("common.err.unknown")
	}

	// Message label with word wrap
	messageLabel := widget.NewLabel(errorMsg)
	messageLabel.Wrapping = fyne.TextWrapWord

	// Log info button - right aligned
	openLogsBtn := widget.NewButtonWithIcon(
		locales.Translate("common.button.openlogs"),
		theme.FolderOpenIcon(),
		func() {
			// Open log viewer window
			ShowLogViewerWindow(window)
		},
	)

	// OK button with high importance
	var dlg *dialog.CustomDialog
	okBtn := widget.NewButton(
		locales.Translate("common.button.ok"),
		func() {
			dlg.Hide()
		},
	)
	okBtn.Importance = widget.HighImportance

	// Create content with properly aligned buttons
	content := container.NewVBox(
		messageLabel,
		container.NewHBox(layout.NewSpacer(), openLogsBtn),
		container.NewHBox(layout.NewSpacer(), okBtn, layout.NewSpacer()),
	)

	// Create and show dialog without default buttons
	dlg = dialog.NewCustomWithoutButtons(header, content, window)
	dlg.Resize(fyne.NewSize(400, 200))
	dlg.Show()
	return dlg
}

// CreatePlaylistSelect creates a select widget for playlist selection.
// Used for components that require database access to be populated with playlists.
// placeholderKey is an optional localization key for the placeholder text shown when no playlist is selected.
// If placeholderKey is empty, default placeholder from common.select.playlist.placeholder will be used.
func CreatePlaylistSelect(changed func(string), placeholderKey string) *widget.Select {
	selectWidget := widget.NewSelect([]string{}, changed)
	if placeholderKey != "" {
		selectWidget.PlaceHolder = locales.Translate(placeholderKey)
	} else {
		selectWidget.PlaceHolder = locales.Translate("common.select.plsplaceholder")
	}
	selectWidget.Disable()
	return selectWidget
}

// CreateDisabledSubmitButton creates a submit button that is disabled by default.
// Used for actions that require database access to be executed.
func CreateDisabledSubmitButton(title string, handler func()) *widget.Button {
	btn := CreateSubmitButton(title, handler)
	btn.Disable()
	return btn
}

// UpdateButtonToCompleted updates a button to show completion state with a confirm icon.
// This is typically used for submit buttons after a process has completed successfully.
func UpdateButtonToCompleted(button *widget.Button) {
	button.SetIcon(theme.ConfirmIcon())
}

// DisableModuleControls disables multiple UI components at once
// This is typically used when the module is in a state where user interaction should be prevented
// For example, when database is not connected or when a process is running
func DisableModuleControls(components ...fyne.Disableable) {
	for _, component := range components {
		component.Disable()
	}
}

// SetPlaylistSelectState updates the state of a playlist select widget.
// This helper ensures consistent behavior when enabling/disabling playlist selects across modules.
// Parameters:
//   - selectWidget: The select widget to update
//   - enabled: Whether the select should be enabled
//   - selectedValue: Optional value to select (only applied if enabled is true)
func SetPlaylistSelectState(selectWidget *widget.Select, enabled bool, selectedValue string) {
	if enabled {
		selectWidget.Enable()
		selectWidget.PlaceHolder = locales.Translate("common.select.plsplaceholder")
		if selectedValue != "" {
			selectWidget.SetSelected(selectedValue)
		}
	} else {
		selectWidget.PlaceHolder = locales.Translate("common.select.plsplacehldrinact")
		selectWidget.Disable()
	}
}

// CreateCheckbox creates a standardized checkbox with a label.
// The checkbox is created with a given label text and change handler.
// Parameters:
//   - labelText: Text to display next to the checkbox
//   - onChanged: Function to call when the checkbox state changes
func CreateCheckbox(labelText string, onChanged func(bool)) *widget.Check {
	checkbox := widget.NewCheck(labelText, onChanged)
	return checkbox
}

// GetLogFilePath returns the path to the log file
func GetLogFilePath() string {
	// Get the application data directory
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}

	// Construct the path to the log file
	logDir := filepath.Join(appDataDir, AppName, FolderNameLog)
	logFile := filepath.Join(logDir, FileNameLog)

	return logFile
}

// ShowLogViewerWindow creates and displays a window with the log file content.
// The log content is displayed in a scrollable text area with monospace font.
// The window includes a refresh button to reload the log content.
func ShowLogViewerWindow(parent fyne.Window) {
	// Get log file path
	logPath := GetLogFilePath()

	// Create text widget for log content
	logText := widget.NewEntry()
	logText.MultiLine = true
	logText.TextStyle = fyne.TextStyle{Monospace: true}
	logText.Wrapping = fyne.TextWrapBreak

	// Make the text read-only
	logText.Disable()

	// Create scroll container for the text
	var scrollContainerRef *container.Scroll
	scrollContainer := container.NewScroll(logText)
	scrollContainerRef = scrollContainer

	// Create window
	logWindow := fyne.CurrentApp().NewWindow(locales.Translate("common.logviewer.header"))

	// Create refresh button
	refreshBtn := widget.NewButtonWithIcon(
		locales.Translate("common.button.refresh"),
		theme.ViewRefreshIcon(),
		func() {
			loadLogContent(logPath, logText, scrollContainerRef)
		},
	)
	refreshBtn.Importance = widget.HighImportance
	// Create close button
	closeBtn := widget.NewButtonWithIcon(
		locales.Translate("common.button.close"),
		theme.CancelIcon(),
		func() {
			// Close the window
			logWindow.Close()
		},
	)

	// Create button container
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		refreshBtn,
		closeBtn,
	)

	// Create main content container
	content := container.NewBorder(
		nil,
		buttonContainer,
		nil,
		nil,
		scrollContainer,
	)

	// Set content and configure window
	logWindow.SetContent(content)
	logWindow.Resize(fyne.NewSize(800, 600))
	logWindow.CenterOnScreen()

	// Load log content
	loadLogContent(logPath, logText, scrollContainerRef)

	// Show window
	logWindow.Show()
}

// loadLogContent loads the content of the log file into the text widget
// and scrolls to the end of the content.
func loadLogContent(logPath string, logText *widget.Entry, scrollContainer *container.Scroll) {
	// Read log file content
	content, err := os.ReadFile(logPath)
	if err != nil {
		logText.SetText(fmt.Sprintf(locales.Translate("common.err.readlog"), err))
		return
	}

	// Set text content
	logText.SetText(string(content))

	// Scroll to end (last line)
	lineCount := strings.Count(string(content), "\n")
	if lineCount > 0 {
		// Set cursor to last line
		logText.CursorRow = lineCount

		// Ensure UI updates
		logText.Refresh()

		// Use a timer to ensure scrolling happens after the content is rendered
		go func() {
			// Wait a short time for the UI to update
			time.Sleep(100 * time.Millisecond)

			// Scroll to bottom
			scrollContainer.ScrollToBottom()
		}()
	}
}

// ShowPanicDialog creates and shows a custom dialog for panic errors, allowing a custom title.
func ShowPanicDialog(window fyne.Window, title, content string) {
	dismissText := locales.Translate("common.button.ok")
	panicDialog := dialog.NewCustom(title, dismissText, widget.NewLabel(content), window)
	panicDialog.Show()
}

// CreateDynamicEntryList creates and manages a dynamic list of folder entry fields.
// This function encapsulates the entire logic for adding, removing, and displaying folder entries.
// Parameters:
// - parent: The parent window for dialogs.
// - initialEntries: A slice of strings to populate the initial entry fields.
// - maxEntries: The maximum number of entries allowed in the list.
//
// Returns:
// - A container with the complete, self-managed UI for the dynamic list.
// - A slice of entry widgets for the calling module to read/write values.
func CreateDynamicEntryList(
	parent fyne.Window,
	initialEntries []string,
	maxEntries int,
	onChanged func([]*widget.Entry),
) (*fyne.Container, []*widget.Entry) {
	entryContainer := container.NewVBox()
	var entries []*widget.Entry

	// addEntry creates a new folder entry row and adds it to the container.
	var addEntry func(path string) *widget.Entry
	addEntry = func(path string) *widget.Entry {
		if len(entries) >= maxEntries {
			return nil // Do not add more than the max limit
		}

		newEntry := widget.NewEntry()
		newEntry.SetPlaceHolder(locales.Translate("common.folder.placeholder"))
		if path != "" {
			newEntry.SetText(path)
		}

		// Create a folder selection field with delete button
		folderField := CreateFolderSelectionFieldWithDelete(
			locales.Translate("common.button.browsefolder"),
			newEntry,
			func(path string) {
				newEntry.SetText(path)

				// If this is the last entry and it's not empty, add a new one
				if path != "" && newEntry == entries[len(entries)-1] && len(entries) < maxEntries {
					addEntry("")
				}

				// Notify about the change
				if onChanged != nil {
					onChanged(entries)
				}
			},
			func() {
				// Find and remove this entry
				for i, entry := range entries {
					if entry == newEntry {
						// Remove from container and entries slice
						entryContainer.Remove(entryContainer.Objects[i])
						entries = append(entries[:i], entries[i+1:]...)
						entryContainer.Refresh()
						break
					}
				}

				// If no entries remain, add an empty one
				if len(entries) == 0 {
					addEntry("")
				}

				// Notify about the change
				if onChanged != nil {
					onChanged(entries)
				}
			},
		)

		entries = append(entries, newEntry)
		entryContainer.Add(folderField)
		entryContainer.Refresh()

		return newEntry
	}

	// Populate with initial entries
	if len(initialEntries) > 0 {
		for _, path := range initialEntries {
			if path != "" {
				addEntry(path)
			}
		}
	}

	// Always ensure there is at least one entry field or add empty entry if not at max limit
	if len(entries) == 0 || (len(entries) < maxEntries && entries[len(entries)-1].Text != "") {
		addEntry("")
	}

	return entryContainer, entries
}

// CreateActionButton creates a standardized action button that changes state after execution.
// Used for buttons that perform an action and show completion state with an icon.
//
// Parameters:
//   - initialText: Text to display initially (without icon)
//   - onAction: Function to call when button is clicked
//   - completedText: Text to display after action completion (if empty, original text is preserved)
//   - completedIcon: Icon to display after action completion
//
// Returns:
//   - A button configured with HighImportance that can change state
func CreateActionButton(initialText string, onAction func(), completedText string, completedIcon fyne.Resource) *widget.Button {
	btn := widget.NewButton(initialText, nil)
	btn.Importance = widget.HighImportance

	// Set the actual handler that will change state after execution
	btn.OnTapped = func() {
		if onAction != nil {
			onAction()
		}
		// Change button to completed state
		if completedText != "" {
			btn.SetText(completedText)
		}
		btn.SetIcon(completedIcon)
	}

	return btn
}
