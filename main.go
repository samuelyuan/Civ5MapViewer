package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/samuelyuan/Civ5MapViewer/internal/ui"
)

func main() {
	e := ui.NewEditor()

	a := app.NewWithID("io.fyne.civ5mapviewer")
	w := a.NewWindow("Civ5 Map Viewer")
	e.BuildUI(w)
	w.Resize(fyne.NewSize(520, 320))

	w.ShowAndRun()
}
