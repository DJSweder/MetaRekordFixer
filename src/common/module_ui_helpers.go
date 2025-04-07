// common/module_ui_helpers.go

package common

import (
	"MetaRekordFixer/locales"
	"fmt"
	"image/color"

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

// ShowSuccess displays a success message and hides the progress dialog
func (pd *ProgressDialog) ShowSuccess(message string) {
	pd.Hide()
	dialog.ShowInformation(locales.Translate("common.diag.success"), message, pd.window)
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

// CreateFileBrowseButton creates a standardized file browse button with filter
func CreateFileBrowseButton(window fyne.Window, entry *widget.Entry, buttonText string, changeHandler func(string), filter []string) *widget.Button {
	return widget.NewButtonWithIcon(buttonText, theme.FileIcon(), func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err == nil && uri != nil {
				entry.SetText(uri.URI().Path())
				if changeHandler != nil {
					changeHandler(uri.URI().Path())
				}
			}
		}, window)
	})
}

// CreateActionButton creates a standardized action button
func CreateActionButton(text string, icon fyne.Resource, action func()) *widget.Button {
	return widget.NewButtonWithIcon(text, icon, action)
}

// CreateButtonBar creates a button container with equal spacing
func CreateButtonBar(buttons ...*widget.Button) fyne.CanvasObject {
	container := container.NewHBox(layout.NewSpacer())

	for _, button := range buttons {
		container.Add(button)
	}

	container.Add(layout.NewSpacer())
	return container
}

// CreateLoadingOverlay creates an overlay with a loading indicator
func CreateLoadingOverlay(parent fyne.Window, message string) *dialog.CustomDialog {
	progress := widget.NewProgressBarInfinite()
	label := widget.NewLabel(message)
	label.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(progress, label)
	d := dialog.NewCustom("", "", content, parent)
	d.SetDismissText("")

	return d
}

// ShowConfirmDialogWithCancel displays a confirmation dialog with cancel option
func ShowConfirmDialogWithCancel(title, message string, onConfirm, onCancel func(), window fyne.Window) *dialog.CustomDialog {
	confirmBtn := widget.NewButtonWithIcon(locales.Translate("common.button.confirm"), theme.ConfirmIcon(), onConfirm)
	cancelBtn := widget.NewButtonWithIcon(locales.Translate("common.button.cancel"), theme.CancelIcon(), onCancel)

	content := container.NewVBox(
		widget.NewLabel(message),
		container.NewHBox(layout.NewSpacer(), cancelBtn, confirmBtn),
	)

	return dialog.NewCustom(title, "", content, window)
}

// ShowTextInputDialog displays a text input dialog
func ShowTextInputDialog(title, message, defaultValue string, onSubmit func(string), window fyne.Window) *dialog.CustomDialog {
	// Create entry field with default value
	entry := widget.NewEntry()
	entry.SetText(defaultValue)

	// Create submit button
	submitBtn := widget.NewButtonWithIcon(locales.Translate("common.button.submit"), theme.ConfirmIcon(), func() {
		if onSubmit != nil {
			onSubmit(entry.Text)
		}
	})
	submitBtn.Importance = widget.HighImportance

	// Create cancel button
	cancelBtn := widget.NewButtonWithIcon(locales.Translate("common.button.cancel"), theme.CancelIcon(), nil)

	// Create content container
	content := container.NewVBox(
		widget.NewLabel(message),
		entry,
		container.NewHBox(layout.NewSpacer(), cancelBtn, submitBtn),
	)

	// Create dialog
	d := dialog.NewCustom(title, "", content, window)
	cancelBtn.OnTapped = func() { d.Hide() }

	return d
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

// CreateRightAlignedSubmitButton creates a standardized submit button with high importance,
// aligned to the right using a spacer
func CreateRightAlignedSubmitButton(title string, handler func()) *fyne.Container {
	btn := CreateSubmitButton(title, handler)
	return container.NewHBox(layout.NewSpacer(), btn)
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

// ErrorDialogDetails represents details shown in the error dialog
type ErrorDialogDetails struct {
	Module      string
	Operation   string
	Error       error
	Severity    ErrorSeverity
	Recoverable bool
	Timestamp   string
	StackTrace  string
}

// ShowStandardError displays a standardized error dialog with optional details
func ShowStandardError(window fyne.Window, err error, context *ErrorContext) *dialog.CustomDialog {
	// Get header based on severity
	var header string
	if context != nil {
		switch context.Severity {
		case ErrorWarning:
			header = locales.Translate("common.dialog.warningheader")
		case ErrorCritical:
			header = locales.Translate("common.dialog.criticalheader")
		case ErrorFatal:
			header = locales.Translate("common.dialog.fatalerror")
		default:
			header = locales.Translate("common.dialog.errorheader")
		}
	} else {
		header = locales.Translate("common.dialog.errorheader")
	}

	// Create error message label
	messageLabel := widget.NewLabel(err.Error())
	messageLabel.Wrapping = fyne.TextWrapWord

	// Create buttons container
	var buttons *fyne.Container
	if context != nil && context.Recoverable {
		// Create details button
		detailsBtn := widget.NewButtonWithIcon(
			locales.Translate("common.button.showdetails"),
			theme.InfoIcon(),
			func() {
				ShowErrorDetails(window, &ErrorDialogDetails{
					Module:      context.Module,
					Operation:   context.Operation,
					Error:       context.Error,
					Severity:    context.Severity,
					Recoverable: context.Recoverable,
					Timestamp:   context.Timestamp.Format("2006-01-02 15:04:05"),
					StackTrace:  context.StackTrace,
				})
			},
		)

		// Create OK and Cancel buttons
		okBtn := widget.NewButtonWithIcon(locales.Translate("common.button.ok"), theme.ConfirmIcon(), nil)
		cancelBtn := widget.NewButtonWithIcon(locales.Translate("common.button.cancel"), theme.CancelIcon(), nil)

		buttons = container.NewHBox(layout.NewSpacer(), detailsBtn, cancelBtn, okBtn)
	} else {
		// Create OK button only
		okBtn := widget.NewButtonWithIcon(locales.Translate("common.button.ok"), theme.ConfirmIcon(), nil)
		buttons = container.NewHBox(layout.NewSpacer(), okBtn)
	}

	// Create content container
	content := container.NewVBox(
		messageLabel,
		buttons,
	)

	// Create and return dialog
	return dialog.NewCustom(header, "", content, window)
}

// ShowErrorDetails displays a dialog with detailed error information
func ShowErrorDetails(window fyne.Window, details *ErrorDialogDetails) *dialog.CustomDialog {
	// Create text area for details
	detailsText := fmt.Sprintf(
		"Module: %s\nOperation: %s\nError: %v\nSeverity: %v\nRecoverable: %v\nTimestamp: %s\n\nStack Trace:\n%s",
		details.Module,
		details.Operation,
		details.Error,
		details.Severity,
		details.Recoverable,
		details.Timestamp,
		details.StackTrace,
	)

	textArea := widget.NewMultiLineEntry()
	textArea.SetText(detailsText)
	textArea.Disable() // Make it read-only

	// Create copy button
	copyBtn := widget.NewButtonWithIcon(
		locales.Translate("common.button.copy"),
		theme.ContentCopyIcon(),
		func() {
			window.Clipboard().SetContent(detailsText)
		},
	)

	// Create close button
	closeBtn := widget.NewButtonWithIcon(
		locales.Translate("common.button.close"),
		theme.CancelIcon(),
		nil,
	)

	// Create buttons container
	buttons := container.NewHBox(layout.NewSpacer(), copyBtn, closeBtn)

	// Create content container with minimum size
	content := container.NewVBox(
		textArea,
		buttons,
	)
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(600, 400))
	content.Add(rect)

	// Create and return dialog
	return dialog.NewCustom(
		locales.Translate("common.dialog.errordetails"),
		"",
		content,
		window,
	)
}

// CreateDisabledSelect creates a select widget that is disabled by default.
// Used for components that require database access to be populated.
// placeholderKey is an optional localization key for the placeholder text shown when no option is selected.
// If placeholderKey is empty, default placeholder from common.select.placeholderiact will be used.
func CreateDisabledSelect(options []string, changed func(string), placeholderKey string) *widget.Select {
	selectWidget := widget.NewSelect(options, changed)
	if placeholderKey != "" {
		selectWidget.PlaceHolder = locales.Translate(placeholderKey)
	} else {
		selectWidget.PlaceHolder = locales.Translate("common.select.placeholderinact")
	}
	selectWidget.Disable()
	return selectWidget
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
