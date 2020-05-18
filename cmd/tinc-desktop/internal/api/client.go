package api

import (
	"context"
	client "github.com/reddec/jsonrpc2/client"
	"sync/atomic"
)

func Default() *WorkerClient {
	return &WorkerClient{BaseURL: "http://127.0.0.1:9999"}
}

type WorkerClient struct {
	BaseURL  string
	sequence uint64
}

//
func (impl *WorkerClient) Kill(ctx context.Context) (reply bool, err error) {
	err = client.CallHTTP(ctx, impl.BaseURL, "Worker.Kill", atomic.AddUint64(&impl.sequence, 1), &reply)
	return
}

//
func (impl *WorkerClient) Peers(ctx context.Context) (reply []string, err error) {
	err = client.CallHTTP(ctx, impl.BaseURL, "Worker.Peers", atomic.AddUint64(&impl.sequence, 1), &reply)
	return
}
