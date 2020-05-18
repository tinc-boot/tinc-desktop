package main

import (
	"context"
	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/tinc-boot/tincd"
	"path/filepath"
)

type screenNew struct {
	Window fyne.Window
	Ctx    context.Context
	App    *App
}

func (sn *screenNew) Show() {
	sn.Window.SetTitle("New network")

	netName := widget.NewEntry()
	netName.PlaceHolder = "network name"

	subnet := widget.NewEntry()
	subnet.Text = "10.152.0.0/16"

	sn.Window.SetTitle("New network")
	sn.Window.SetContent(widget.NewVBox(
		widget.NewToolbar(
			widget.NewToolbarAction(theme.NavigateBackIcon(), func() {
				sn.App.ShowMainScreen()
			}),
			widget.NewToolbarSeparator(),
			widget.NewToolbarSpacer(),
		),
		widget.NewVBox(
			netName,
			subnet,
		),
		widget.NewButton("Create", func() {
			sn.create(subnet.Text, netName.Text)
		}),
	))
}

func (sn *screenNew) create(subnet string, netName string) {
	progress := dialog.NewProgressInfinite("Creating", "creating... ", sn.Window)
	progress.Show()
	ntw, err := tincd.Create(filepath.Join(sn.App.Config.ConfigDir, netName), subnet)
	if err != nil {
		progress.Hide()
		dialog.NewInformation("Failed", err.Error(), sn.Window).Show()
	} else {
		progress.Hide()
		sn.App.ShowNetworkScreen(ntw)
	}
}
