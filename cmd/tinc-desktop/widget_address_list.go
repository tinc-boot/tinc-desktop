package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/tinc-boot/tincd/network"
	"strconv"
)

type addressesList struct {
	addresses []*network.Address
	container *widget.Box
}

func (add *addressesList) build(addresses []network.Address) fyne.CanvasObject {
	add.container = widget.NewVBox()

	for _, addr := range addresses {
		add.addAddress(&addr)
	}

	return widget.NewVBox(
		add.container,
		widget.NewButton("Add public address", func() {
			add.addAddress(&network.Address{})
		}),
	)
}

func (add *addressesList) addAddress(addr *network.Address) {
	txt := widget.NewEntry()
	txt.PlaceHolder = "public address"
	txt.SetText(addr.Host)
	txt.OnChanged = func(s string) {
		addr.Host = s
	}

	p := widget.NewEntry()
	p.PlaceHolder = "port num"
	if addr.Port != 0 {
		p.SetText(strconv.Itoa(int(addr.Port)))
	}
	p.OnChanged = func(s string) {
		if s == "" {
			addr.Port = 0
		} else if sp, err := strconv.ParseUint(s, 10, 16); err == nil {
			addr.Port = uint16(sp)
		}
	}
	var line *widget.Box

	b := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
		for i, v := range add.addresses {
			if v == addr {
				add.addresses = append(add.addresses[:i], add.addresses[i+1:]...)
				add.container.Children = append(add.container.Children[:i], add.container.Children[i+1:]...)
				add.container.Refresh()
				break
			}
		}
	})
	line = widget.NewHBox(txt, p, b)

	add.addresses = append(add.addresses, addr)
	add.container.Append(line)
}
