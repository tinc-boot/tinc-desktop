package main

import (
	"context"
	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/pkg/browser"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
	"github.com/tinc-boot/tincd/network"
	"log"
	"path/filepath"
)

type screenNetwork struct {
	Window  fyne.Window
	Network *network.Network
	Ctx     context.Context
	App     *App
	toolbar *widget.Toolbar
}

func (sc *screenNetwork) Show() {
	self, config, err := sc.Network.SelfConfig()
	if err != nil {
		dialog.NewInformation("Failed", err.Error(), sc.Window).Show()
		return
	}
	var running bool
	action := widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
		sc.start()
	})
	if inst := sc.App.Pool.Find(sc.Network.Name()); inst != nil {
		select {
		case <-inst.Done():
		default:
			running = true
		}
		if running {
			action = widget.NewToolbarAction(theme.MediaPauseIcon(), func() {
				sc.stop(inst)
			})
		}
	}

	sc.toolbar = widget.NewToolbar(
		widget.NewToolbarAction(theme.NavigateBackIcon(), func() {
			sc.App.ShowMainScreen()
		}),
		widget.NewToolbarSeparator(),
		action,
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			sc.destroy()
		}),
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			err := browser.OpenFile(sc.Network.Root)
			if err != nil {
				log.Println(err)
				dialog.NewInformation("Failed open config dir", err.Error(), sc.Window).Show()
			}
		}),
		widget.NewToolbarAction(theme.InfoIcon(), func() {
			err := browser.OpenFile(filepath.Join(sc.Network.Root, "log.txt"))
			if err != nil {
				log.Println("open log file:", err)
			}
		}),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			sc.App.ShowNetworkSettingsScreen(sc.Network)
		}),
	)
	peers := widget.NewVBox()

	sc.Window.SetTitle(sc.Network.Name())

	var elements = []fyne.CanvasObject{
		sc.toolbar,
		widget.NewLabel(config.Name),
		fyne.NewContainerWithLayout(layout.NewGridLayout(2),
			widget.NewLabel("VPN IP"), widget.NewLabel(self.IP),
			widget.NewLabel("Subnet"), widget.NewLabel(self.Subnet),
		),
	}
	if running {
		elements = append(elements, widget.NewGroup("Active peers",
			widget.NewButtonWithIcon("refresh", theme.ViewRefreshIcon(), func() {
				sc.listActivePeers(peers)
			}),
			peers,
		))
		sc.listActivePeers(peers)
	}
	sc.Window.SetContent(widget.NewVBox(elements...))
}

func (sc *screenNetwork) destroy() {
	progress := dialog.NewProgressInfinite("Removing", "removing... ", sc.Window)
	progress.Show()

	if ntw := sc.App.Pool.Find(sc.Network.Name()); ntw != nil {
		_, _ = ntw.API().Kill(context.Background())
		<-ntw.Done()
	}

	err := sc.Network.Destroy()
	progress.Hide()

	if err != nil {
		dialog.NewInformation("Failed", err.Error(), sc.Window).Show()
		return
	}

	sc.App.ShowMainScreen()

}

func (sc *screenNetwork) listActivePeers(container *widget.Box) {

	ntw := sc.App.Pool.Find(sc.Network.Name())
	if ntw == nil {
		container.Children = nil
		container.Refresh()
		return
	}

	updateDialog := dialog.NewProgressInfinite("Updating", "updating... ", sc.Window)
	updateDialog.Show()

	peers, err := ntw.API().Peers(sc.Ctx)
	updateDialog.Hide()
	if err != nil {
		log.Println("list active peers:", err)
		return
	}

	grid := layout.NewGridLayout(2)

	var items []fyne.CanvasObject
	for _, name := range peers {
		info, err := sc.Network.Node(name)
		if err != nil {
			log.Println(name, err)
			continue
		}
		items = append(items, widget.NewLabel(name), widget.NewLabel(info.IP))
	}

	container.Children = []fyne.CanvasObject{fyne.NewContainerWithLayout(grid, items...)}
	container.Refresh()
}

func (sc *screenNetwork) start() {
	if !internal.CanStart() {
		dialog.NewInformation("Oops", "Please start application as Administrator", sc.Window).Show()
		return
	}
	startingDialog := dialog.NewProgressInfinite("Starting", "starting... ", sc.Window)
	startingDialog.Show()

	worker, err := sc.App.Pool.SpawnSudoContext(sc.Network.Name())

	if err != nil {
		startingDialog.Hide()
		log.Println("start", sc.Network.Name(), err)
		dialog.NewInformation("Failed to start", err.Error(), sc.Window).Show()
		return
	}
	startingDialog.Hide()
	sc.toolbar.Items[2] = widget.NewToolbarAction(theme.MediaPauseIcon(), func() {
		_, _ = worker.API().Kill(context.Background())
	})
	sc.refreshToolbar()

	go func() {
		<-worker.Done()
		sc.stop(worker)
	}()
}

func (sc *screenNetwork) stop(instance internal.Port) {
	stoppingDialog := dialog.NewProgressInfinite("Stopping", "stopping...", sc.Window)

	stoppingDialog.Show()
	log.Println("stop", instance.Name())
	_, _ = instance.API().Kill(context.Background())
	<-instance.Done()

	sc.toolbar.Items[2] = widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
		sc.start()
	})
	stoppingDialog.Hide()
	sc.refreshToolbar()
}

func (sc *screenNetwork) refreshToolbar() {
	sc.Show()
}
