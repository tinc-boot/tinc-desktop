package spawners

import (
	"context"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
	"github.com/tinc-boot/tincd"
	"path/filepath"
)

type SameProcess struct {
	ConfigLocation string
}

func (sp *SameProcess) Spawn(network string, done chan<- error) (internal.Port, error) {
	instance, err := tincd.StartFromDir(context.Background(), filepath.Join(sp.ConfigLocation, network), false)
	if err != nil {
		return nil, err
	}

	port := &samePort{
		client: tincdPort{client: instance},
		done:   make(chan struct{}),
		name:   network,
	}

	go func() {
		defer close(port.done)
		<-instance.Done()
		done <- instance.Error()
	}()

	return port, nil
}

type samePort struct {
	client tincdPort
	done   chan struct{}
	name   string
}

func (wp *samePort) Name() string          { return wp.name }
func (wp *samePort) Done() <-chan struct{} { return wp.done }
func (wp *samePort) API() internal.Worker  { return &wp.client }

type tincdPort struct {
	client tincd.Tincd
}

func (t *tincdPort) Kill(ctx context.Context) (bool, error) {
	t.client.Stop()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-t.client.Done():
	}
	return true, t.client.Error()
}

func (t *tincdPort) Peers(ctx context.Context) ([]string, error) {
	return t.client.Peers(), nil
}
