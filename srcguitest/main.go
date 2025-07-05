package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Testovací Aplikace")

	myWindow.SetContent(widget.NewLabel("Ahoj, jestli mě vidíš, Fyne funguje!"))

	myWindow.ShowAndRun()
}
