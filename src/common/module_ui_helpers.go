// common/module_ui_helpers.go

package common

import (
	"MetaRekordFixer/locales"
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

// Úpravy podle doporučení v refactoring_result.txt

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

// CreateStandardForm creates a standard form layout
func CreateStandardForm(items ...interface{}) fyne.CanvasObject {
	form := container.NewVBox()

	for i := 0; i < len(items); i += 3 {
		if i+2 < len(items) {
			label := items[i].(string)
			input := items[i+1].(fyne.CanvasObject)
			button := items[i+2].(fyne.CanvasObject)

			row := container.NewBorder(nil, nil, nil, button, input)
			formItem := container.NewHBox(widget.NewLabel(label), row)

			form.Add(formItem)
		}
	}

	return form
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

// CreateFolderBrowseButton creates a standardized folder browse button
// DEPRECATED: Use CreateNativeFolderBrowseButton instead to avoid issues with folder selection on Windows
func CreateFolderBrowseButton(window fyne.Window, entry *widget.Entry, buttonText string, changeHandler func(string)) *widget.Button {
	return widget.NewButtonWithIcon(buttonText, theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				entry.SetText(uri.Path())
				if changeHandler != nil {
					changeHandler(uri.Path())
				}
			}
		}, window)
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

// CreateInfoSection creates an informational text section
func CreateInfoSection(text string) fyne.CanvasObject {
	infoLabel := widget.NewLabel(text)
	infoLabel.Wrapping = fyne.TextWrapWord
	return container.NewVBox(infoLabel, widget.NewSeparator())
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
	entry := widget.NewEntry()
	entry.SetText(defaultValue)

	submitBtn := widget.NewButtonWithIcon(locales.Translate("common.button.submit"), theme.ConfirmIcon(), func() {
		onSubmit(entry.Text)
	})

	cancelBtn := widget.NewButtonWithIcon(locales.Translate("common.button.cancel"), theme.CancelIcon(), nil)

	content := container.NewVBox(widget.NewLabel(message), entry, container.NewHBox(layout.NewSpacer(), cancelBtn, submitBtn))
	d := dialog.NewCustom(title, "", content, window)

	cancelBtn.OnTapped = func() { d.Hide() }

	return d
}

// CreateModuleContainer creates a standard UI container for a module
func CreateModuleContainer(title string, content, actions fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewBorder(
		container.NewVBox(container.NewHBox(titleLabel, layout.NewSpacer()), widget.NewSeparator()),
		actions, nil, nil, content,
	)
}

// CreateVisibilityToggler toggles widget visibility based on a condition
func CreateVisibilityToggler(condition func() bool, widgets ...fyne.CanvasObject) func() {
	return func() {
		visible := condition()
		for _, w := range widgets {
			w.Hide()
			if visible {
				w.Show()
			}
		}
	}
}

// CreateFolderSelectionField creates a standardized folder selection field with browse button
// This creates a container with an entry field and a browse button for folder selection
// If entryField is nil, creates a new entry field, otherwise uses the provided one
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

// CreateSubmitButton creates a standardized submit button with high importance
// This button is used to start a process or submit a form
func CreateSubmitButton(title string, handler func()) *widget.Button {
	btn := widget.NewButton(title, handler)
	btn.Importance = widget.HighImportance
	return btn
}

// CreateSubmitButtonWithIcon creates a standardized submit button with an icon and high importance
// This button is used to start a process or submit a form
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

// CreateStandardModuleLayout creates a standardized layout for a module
// This layout includes a description at the top, custom content in the middle, and a submit button at the bottom
// The custom content can be any CanvasObject, allowing for flexibility in the module's layout
func CreateStandardModuleLayout(description string, content fyne.CanvasObject, submitButton *widget.Button) fyne.CanvasObject {
	// Create description label
	descLabel := CreateDescriptionLabel(description)

	// Create main container for the module content
	mainContent := container.NewVBox(
		descLabel,
		widget.NewSeparator(),
		content,
	)

	// Add submit button with right alignment if provided
	if submitButton != nil {
		buttonBox := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), submitButton)
		mainContent.Add(buttonBox)
	}

	// Create a container for status messages that will expand to fill available space
	// This container will be populated by the module's status messages container
	// when the module is initialized
	statusMessagesSpace := container.NewVBox()

	// Vytvoříme layout, kde hlavní obsah má fixní velikost a prostor pro statusové zprávy
	// se rozšiřuje, aby vyplnil zbývající prostor
	// Použijeme container.NewBorderLayout, který umožňuje dynamické rozšiřování prvku
	return container.New(
		layout.NewBorderLayout(mainContent, nil, nil, nil),
		mainContent,
		statusMessagesSpace,
	)
}
