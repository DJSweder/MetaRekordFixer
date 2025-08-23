package ui

import (
	"MetaRekordFixer/common"
	"MetaRekordFixer/locales"
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type languageItem struct {
	Code string
	Name string
}

// ShowSettingsWindow creates and displays the settings window.
func ShowSettingsWindow(parent fyne.Window, configMgr *common.ConfigManager, errorHandler *common.ErrorHandler) {
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

	// Create file path entry with browse button using existing abstraction
	dbPathContainer := common.CreateFolderSelectionField(locales.Translate("settings.browse.title"), dbPathEntry, nil)

	// Create detect button using abstraction - empty completedText to preserve original text
	detectButton := common.CreateActionButton(
		locales.Translate("settings.button.autodetectdb"),
		func() {
			detectedPath, err := common.DetectRekordboxDatabase()
			if err != nil {
				// Use centralized error handling with proper error wrapping
				context := &common.ErrorContext{
					Module:      "Settings",
					Operation:   "Database Autodetection",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				}
				// Wrap the error with user-friendly context while preserving technical details
				wrappedErr := fmt.Errorf("%s: %w", locales.Translate("common.err.autodetectdb"), err)
				errorHandler.ShowStandardError(wrappedErr, context)
			} else {
				dbPathEntry.SetText(detectedPath)
			}
		},
		"", // Prázdný text pro zachování původního textu tlačítka
		theme.ConfirmIcon(),
	)

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

	// Create save button using abstraction
	saveButton = common.CreateActionButton(
		locales.Translate("settings.write.settings"),
		func() {
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
				// Use centralized error handling for configuration save failure
				context := &common.ErrorContext{
					Module:      "Settings",
					Operation:   "Save Configuration",
					Severity:    common.SeverityError,
					Recoverable: true,
				}
				wrappedErr := fmt.Errorf("%s: %w", locales.Translate("settings.err.save"), err)
				errorHandler.ShowStandardError(wrappedErr, context)
				return
			}

			// Show warning if database path is empty using centralized error handling
			if dbPathEntry.Text == "" {
				context := &common.ErrorContext{
					Module:      "Settings",
					Operation:   "Database Path Validation",
					Severity:    common.SeverityWarning,
					Recoverable: true,
				}
				err := errors.New(locales.Translate("settings.err.missing"))
				errorHandler.ShowStandardError(err, context)
			}
		},
		locales.Translate("settings.status.saved"),
		theme.ConfirmIcon(),
	)

	// Update window content
	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(locales.Translate("settings.rbxdb.loc"), container.NewBorder(nil, nil, nil, detectButton, dbPathContainer)),
			widget.NewFormItem(locales.Translate("settings.lang.sel"), languageSelect),
		),
		container.NewHBox(layout.NewSpacer(), saveButton),
	)

	// Create modal dialog instead of new window
	settingsDialog := dialog.NewCustom(
		locales.Translate("settings.win.title"),
		"", // Clear text for default button
		form,
		parent,
	)

	// Create own close button
	closeButton := widget.NewButton(locales.Translate("common.button.close"), func() {
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
