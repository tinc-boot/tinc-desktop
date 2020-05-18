package main

import (
	"context"
	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/pkg/browser"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/manager"
	"github.com/tinc-boot/tincd/network"
	"log"
)

type App struct {
	Window fyne.Window
	Ctx    context.Context
	Config Config
	App    fyne.App
	Pool   manager.Manager
}

func (app *App) ShowMainScreen() {
	app.Window.SetTitle("Tinc desktop")
	networks, err := network.List(app.Config.ConfigDir)
	if err != nil {
		log.Println("failed list networks:", err)
	}

	var links []fyne.CanvasObject
	for _, ntw := range networks {
		var cp = ntw
		links = append(links, widget.NewButton(ntw.Name(), func() {
			app.ShowNetworkScreen(cp)
		}))
	}

	app.Window.SetContent(widget.NewVBox(
		widget.NewToolbar(
			widget.NewToolbarAction(theme.NavigateBackIcon(), func() {
				app.App.Quit()
			}),
			widget.NewToolbarSeparator(),
			widget.NewToolbarSpacer(),
			widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
				err := browser.OpenFile(app.Config.ConfigDir)
				if err != nil {
					dialog.NewInformation("Failed open config dir", err.Error(), app.Window).Show()
				}
			}),
			widget.NewToolbarAction(theme.InfoIcon(), func() {
				err := browser.OpenFile(app.Config.logfile())
				if err != nil {
					dialog.NewInformation("Failed open log file", err.Error(), app.Window).Show()
				}
			}),
			widget.NewToolbarAction(theme.MoveDownIcon(), func() {
				app.ShowJoinByURLScreen()
			}),
			widget.NewToolbarAction(theme.ContentAddIcon(), func() {
				app.ShowNewNetworkScreen()
			}),
		),
		widget.NewVBox(links...),
	))
}

func (app *App) ShowNetworkScreen(ntw *network.Network) {
	screen := &screenNetwork{
		Window:  app.Window,
		Network: ntw,
		Ctx:     app.Ctx,
		App:     app,
	}
	screen.Show()
}

func (app *App) ShowNetworkSettingsScreen(ntw *network.Network) {
	screen := &screenSettingsNetwork{
		Window:  app.Window,
		Network: ntw,
		Ctx:     app.Ctx,
		App:     app,
	}
	screen.Show()
}

func (app *App) ShowNewNetworkScreen() {
	var sn = &screenNew{
		Window: app.Window,
		Ctx:    app.Ctx,
		App:    app,
	}
	sn.Show()
}

func (app *App) ShowJoinByURLScreen() {
	var screen = &screenJoinByLink{
		Window: app.Window,
		Ctx:    app.Ctx,
		App:    app,
	}

	screen.Show()
}
