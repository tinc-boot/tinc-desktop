package main

import (
	"context"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"github.com/jessevdk/go-flags"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/manager"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/spawners"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
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
