package main

import (
	"nuxui.org/nuxui/nux"
	"nuxui.org/nuxui/ui"
)

func main() {
	nux.Main(func(app nux.App) {
		w := app.NewWindow(nux.Attr{
			"title":  "NuxUI Test",
			"width":  400,
			"height": 300,
		})

		label := ui.NewText(nux.Attr{
			"text": "Ahoj, jestli mě vidíš, NuxUI funguje!",
		})

		w.SetContent(label)
		w.Show()
	})
}
