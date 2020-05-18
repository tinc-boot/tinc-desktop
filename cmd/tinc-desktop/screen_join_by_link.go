package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/tinc-boot/tinc-desktop/api/tincwebmajordomo"
	"github.com/tinc-boot/tincd"
	"log"
	"path/filepath"
	"strings"
)

type screenJoinByLink struct {
	Window fyne.Window
	Ctx    context.Context
	App    *App
}

func (sjl *screenJoinByLink) Show() {
	url := widget.NewEntry()
	url.PlaceHolder = "URL"
	sjl.Window.SetContent(widget.NewVBox(
		widget.NewToolbar(
			widget.NewToolbarAction(theme.NavigateBackIcon(), func() {
				sjl.App.ShowMainScreen()
			}),
			widget.NewToolbarSeparator(),
			widget.NewToolbarSpacer(),
		),
		widget.NewHScrollContainer(url),
		widget.NewButton("Join", func() {
			sjl.join(strings.TrimSpace(url.Text))
		}),
	))
}

func (sjl *screenJoinByLink) join(url string) {
	parts := strings.Split(url, "/")
	token := parts[len(parts)-1]

	if len(token) == 0 || !strings.Contains(token, ".") {
		return
	}

	data := strings.Split(token, ".")[1]
	bindata, err := base64.RawStdEncoding.DecodeString(data)
	if err != nil {
		dialog.NewInformation("Failed", err.Error(), sjl.Window).Show()
		return
	}

	var share struct {
		Network string `json:"network"`
		Subnet  string `json:"subnet"`
	}

	err = json.Unmarshal(bindata, &share)
	if err != nil {
		dialog.NewInformation("Failed", err.Error(), sjl.Window).Show()
		return
	}

	remote := &tincwebmajordomo.TincWebMajordomoClient{BaseURL: url}

	progress := dialog.NewProgressInfinite("Creating", "creating "+share.Network+" network", sjl.Window)
	progress.Show()

	ntw, err := tincd.Create(filepath.Join(sjl.App.Config.ConfigDir, share.Network), share.Subnet)
	if err != nil {
		progress.Hide()
		dialog.NewInformation("Failed", err.Error(), sjl.Window).Show()
		return
	}

	self, err := ntw.Self()
	if err != nil {
		progress.Hide()
		dialog.NewInformation("Failed", err.Error(), sjl.Window).Show()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), joinTimeout)
	defer cancel()
	sharedNet, err := remote.Join(ctx, share.Network, self)
	if err != nil {
		progress.Hide()
		dialog.NewInformation("Failed", err.Error(), sjl.Window).Show()
		return
	}

	for _, node := range sharedNet.Nodes {
		err = ntw.Put(node)
		if err != nil {
			log.Println(node.Name, err)
		}
	}
	progress.Hide()

	sjl.App.ShowNetworkScreen(ntw)
}
