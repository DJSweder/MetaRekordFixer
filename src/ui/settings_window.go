package ui

import (
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"strings"

	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	nativedialog "github.com/sqweek/dialog"
)

type languageItem struct {
	Code string
	Name string
}

// ShowSettingsWindow creates and displays the settings window.
func ShowSettingsWindow(parent fyne.Window, configMgr *common.ConfigManager) {
	// Load current configuration
	config := configMgr.GetGlobalConfig()

	// Declare the save button in advance
	var saveButton *widget.Button

	// Create UI components
	dbPathEntry := widget.NewEntry()
	dbPathEntry.SetText(config.DatabasePath)
	dbPathEntry.OnChanged = func(string) {
		if saveButton != nil {
			saveButton.SetIcon(nil)
			saveButton.SetText(locales.Translate("settings.write.settings"))
		}
	}

	statusLabel := widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Create the browse button for the database path
	dbPathBrowseButton := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		filename, err := nativedialog.File().Filter(locales.Translate("settings.browse.filter"), "db").Load()
		if err == nil && filename != "" {
			dbPathEntry.SetText(filename)
		}
	})

	// Language selection setup
	availableLangCodes := locales.GetAvailableLanguages()
	var langItems []languageItem
	for _, code := range availableLangCodes {
		name := locales.Translate("settings.lang." + code)
		if strings.HasPrefix(name, "settings.lang.") {
			name = code // Fallback to language code if translation is missing
		}
		langItems = append(langItems, languageItem{Code: code, Name: name})
	}

	langOptions := make([]string, len(langItems))
	for i, lang := range langItems {
		langOptions[i] = lang.Name
	}

	languageSelect := widget.NewSelect(langOptions, func(string) {
		if saveButton != nil {
			saveButton.SetIcon(nil)
			saveButton.SetText(locales.Translate("settings.write.settings"))
		}
	})
	for _, lang := range langItems {
		if lang.Code == config.Language {
			languageSelect.SetSelected(lang.Name)
			break
		}
	}

	// Create the save button with dynamic icon
	saveButton = widget.NewButtonWithIcon(locales.Translate("settings.write.settings"), nil, func() {
		// Update and save config
		config.DatabasePath = dbPathEntry.Text

		// Find selected language code
		for _, lang := range langItems {
			if lang.Name == languageSelect.Selected {
				config.Language = lang.Code
				break
			}
		}

		// Save the configuration
		err := configMgr.SaveGlobalConfig(config)
		if err != nil {
			log.Printf("error saving configuration: %v", err)
			statusLabel.SetText(locales.Translate("settings.err.save"))
			return
		}

		// Show warning if database path is empty
		if dbPathEntry.Text == "" {
			statusLabel.SetText(locales.Translate("settings.err.missing"))
		} else {
			statusLabel.SetText("") // Clear status label if no error
		}

		// Change button to saved state
		saveButton.SetIcon(theme.ConfirmIcon())
		saveButton.SetText(locales.Translate("settings.status.saved"))
	})
	saveButton.Importance = widget.HighImportance

	// Update window content
	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(locales.Translate("settings.rbxdb.loc"), container.NewBorder(nil, nil, nil, dbPathBrowseButton, dbPathEntry)),
			widget.NewFormItem(locales.Translate("settings.lang.sel"), languageSelect),
		),
		container.NewHBox(layout.NewSpacer(), saveButton),
		statusLabel,
	)

	// Create modal dialog instead of new window
	settingsDialog := dialog.NewCustom(
		locales.Translate("settings.win.title"),
		"", // Clear text for default button
		form,
		parent,
	)

	// Create own close button
	closeButton := widget.NewButton(locales.Translate("settings.window.close"), func() {
		settingsDialog.Hide()
	})
	closeButton.Importance = widget.DangerImportance

	// Add close button to dialog
	settingsDialog.SetButtons([]fyne.CanvasObject{closeButton})

	// Set dialog size
	settingsDialog.Resize(fyne.NewSize(800, 400))

	// Show dialog as modal
	settingsDialog.Show()
}
