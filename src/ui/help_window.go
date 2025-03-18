// Package ui provides user interface components for the application
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ShowHelpWindow creates and displays the help window.
func ShowHelpWindow(parent fyne.Window) {
	content := widget.NewLabel("Help content will be added here.")

	window := fyne.CurrentApp().NewWindow("Help")
	window.SetContent(container.NewVBox(content))
	window.Resize(fyne.NewSize(600, 400))
	window.CenterOnScreen()
	window.Show()
}
