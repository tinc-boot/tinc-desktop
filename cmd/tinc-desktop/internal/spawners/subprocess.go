package spawners

import (
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal/api"
	"github.com/tinc-boot/tinc-desktop/sudo"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
)

type SubProcess struct {
	ConfigLocation string
}

func (sp *SubProcess) Spawn(network string, done chan<- error) (internal.Port, error) {
	port := 32000 + rand.Intn(32000)

	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	var arguments = []string{executable, "-c", sp.ConfigLocation, "-p", strconv.Itoa(port), "-n", network}
	cmdParams := sudo.WithSudo(arguments)
	cmd := exec.Command(cmdParams[0], cmdParams[1:]...)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	wp := &workerPort{
		client: &api.WorkerClient{BaseURL: "http://127.0.0.1:" + strconv.Itoa(port)},
		done:   make(chan struct{}),
		name:   network,
	}

	go func() {
		close(wp.done)
		done <- cmd.Wait()
	}()

	return wp, nil
}

type workerPort struct {
	client internal.Worker
	done   chan struct{}
	name   string
}

func (wp *workerPort) Name() string          { return wp.name }
func (wp *workerPort) Done() <-chan struct{} { return wp.done }
func (wp *workerPort) API() internal.Worker  { return wp.client }
