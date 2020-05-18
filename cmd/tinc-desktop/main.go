package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/browser"
	"github.com/tinc-boot/tinc-desktop/api/tincwebmajordomo"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/manager"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/spawners"
	"github.com/tinc-boot/tincd"
	"github.com/tinc-boot/tincd/network"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	joinTimeout = 15 * time.Second
)

type Config struct {
	ConfigDir string `short:"c" long:"config-dir" env:"CONFIG_DIR" description:"Configuration directory (empty - default for OS)"`
	Port      int    `short:"p" long:"port" env:"PORT" description:"Port for runner"`
	Network   string `short:"n" long:"network" env:"NETWORK" description:"Network name for runner"`
}

func (cfg *Config) configure() error {
	if cfg.ConfigDir == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		cfg.ConfigDir = filepath.Join(dir, "tinc-desktop")
	}
	return os.MkdirAll(cfg.ConfigDir, 0755)
}

func (cfg *Config) logfile() string {
	return filepath.Join(cfg.ConfigDir, "log.txt")
}

func main() {
	var cfg Config
	_, err := flags.Parse(&cfg)
	if err != nil {
		os.Exit(1)
	}
	err = cfg.configure()
	if err != nil {
		log.Fatal(err)
	}
	logfile, err := os.Create(cfg.logfile())
	if err != nil {
		log.Println(err)
	} else {
		log.SetOutput(io.MultiWriter(logfile, os.Stderr))
		defer logfile.Close()
	}
	gctx, closer := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Kill, os.Interrupt)
		for range c {
			closer()
			break
		}
	}()
	defer closer()

	defer func() {
		if r := recover(); r != nil {
			log.Println(string(debug.Stack()))
			if logfile != nil {
				logfile.Close()
			}
			os.Exit(1)
		}
	}()
	if cfg.Port == 0 {
		err = run(gctx, cfg)
	} else {
		err = runNetwork(gctx, cfg.Port, filepath.Join(cfg.ConfigDir, cfg.Network))
	}
	if err != nil {
		log.Println(err)
	}
}

func run(ctx context.Context, cfg Config) error {
	a := app.New()
	w := a.NewWindow("Tinc desktop")
	wapp := &App{
		Window: w,
		Ctx:    ctx,
		Config: cfg,
		App:    a,
		Pool:   manager.Manager{Spawner: spawners.SelectSpawner(cfg.ConfigDir)},
	}
	w.Resize(fyne.NewSize(320, 480))
	w.CenterOnScreen()
	wapp.ShowMainScreen()
	go func() {
		<-ctx.Done()
		a.Quit()
	}()
	w.ShowAndRun()
	for _, name := range wapp.Pool.Names() {
		if ntw := wapp.Pool.Find(name); ntw != nil {
			log.Println("stopping", name)
			_, _ = ntw.API().Kill(context.Background())
			<-ntw.Done()
		}
	}
	return nil
}

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
		running = true
		action = widget.NewToolbarAction(theme.MediaPauseIcon(), func() {
			sc.stop(inst)
		})
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
		dialog.NewInformation("Failed", err.Error(), sc.Window).Show()
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
		dialog.NewInformation("Failed", err.Error(), sc.Window).Show()
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
	defer stoppingDialog.Hide()
	log.Println("stop", instance.Name())
	_, _ = instance.API().Kill(context.Background())
	<-instance.Done()

	sc.toolbar.Items[2] = widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
		sc.start()
	})
	sc.refreshToolbar()
}

func (sc *screenNetwork) refreshToolbar() {
	sc.Show()
}

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
