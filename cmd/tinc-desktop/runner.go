package main

import (
	"context"
	"github.com/reddec/jsonrpc2"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/api"
	"github.com/tinc-boot/tincd"
	"github.com/tinc-boot/tincd/network"
	"net/http"
	"strconv"
)

func runNetwork(global context.Context, port int, directory string) error {
	ctx, cancel := context.WithCancel(global)
	defer cancel()

	inst, err := tincd.Start(ctx, &network.Network{Root: directory}, false)
	if err != nil {
		return err
	}

	var run = runner{instance: inst}

	var router jsonrpc2.Router
	api.RegisterWorker(&router, &run)

	go func() {
		wh := jsonrpc2.HandlerRestContext(ctx, &router)
		_ = http.ListenAndServe("127.0.0.1:"+strconv.Itoa(port), wh)
		cancel()
	}()

	<-inst.Done()
	return inst.Error()
}

type runner struct {
	instance tincd.Tincd
}

func (r *runner) Kill(ctx context.Context) (bool, error) {
	r.instance.Stop()
	return true, r.instance.Error()
}

func (r *runner) Peers(ctx context.Context) ([]string, error) {
	return r.instance.Peers(), r.instance.Error()
}
