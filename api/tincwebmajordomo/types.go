package tincwebmajordomo

import "github.com/tinc-boot/tincd/network"

type Network struct {
	Name    string          `json:"name"`
	Running bool            `json:"running"`
	Config  *network.Config `json:"config,omitempty"` // only for specific request
}

type PeerInfo struct {
	Name          string       `json:"name"`
	Online        bool         `json:"online"`
	Configuration network.Node `json:"config"`
}

type Sharing struct {
	Name   string          `json:"name"`
	Subnet string          `json:"subnet"`
	Nodes  []*network.Node `json:"node,omitempty"`
}
