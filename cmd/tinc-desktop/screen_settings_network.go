package main

import (
	"context"
	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/tinc-boot/tincd/network"
	"strconv"
)

type screenSettingsNetwork struct {
	Window  fyne.Window
	Network *network.Network
	Ctx     context.Context
	App     *App
}

func (ssn *screenSettingsNetwork) Show() {
	ssn.Window.SetTitle("Update network")

	self, config, err := ssn.Network.SelfConfig()
	if err != nil {
		dialog.NewInformation("Failed", err.Error(), ssn.Window).Show()
		return
	}

	port := widget.NewEntry()
	port.PlaceHolder = "listening port"
	port.SetText(strconv.Itoa(int(config.Port)))
	port.OnChanged = func(s string) {
		if v, err := strconv.ParseUint(s, 10, 16); err == nil {
			config.Port = uint16(v)
		}
	}

	device := widget.NewEntry()
	device.PlaceHolder = "device name"
	device.SetText(config.Device)
	device.OnChanged = func(s string) {
		config.Device = s
	}

	var addressList addressesList

	ssn.Window.SetContent(widget.NewVBox(
		widget.NewToolbar(
			widget.NewToolbarAction(theme.NavigateBackIcon(), func() {
				ssn.App.ShowNetworkScreen(ssn.Network)
			}),
			widget.NewToolbarSeparator(),
			widget.NewToolbarSpacer(),
			widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
				ssn.update(config.Port, config.Device, addressList.addresses)
			}),
		),
		widget.NewVBox(
			port,
			device,
			addressList.build(self.Address),
		),
	))
}

func (ssn *screenSettingsNetwork) update(port uint16, device string, addreses []*network.Address) {
	var addrs = make([]network.Address, 0, len(addreses))
	for _, a := range addreses {
		addrs = append(addrs, *a)
	}

	updateDialog := dialog.NewProgressInfinite("Updating", "updating... ", ssn.Window)
	updateDialog.Show()

	err := ssn.Network.Upgrade(network.Upgrade{
		Port:    port,
		Address: addrs,
		Device:  device,
	})
	if err != nil {
		updateDialog.Hide()
		dialog.NewInformation("Failed", err.Error(), ssn.Window).Show()
		return
	}
	updateDialog.Hide()
	ssn.App.ShowNetworkScreen(ssn.Network)
}
