package cascadht

import (
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type (
	Option  func(*options) error
	options struct {
		h              host.Host
		httpListenAddr string
		bootstrapPeers []peer.AddrInfo
		useAccDHT      bool
	}
)

func newOptions(o ...Option) (*options, error) {
	opts := options{
		httpListenAddr: "0.0.0.0:40080",
		useAccDHT:      false,
	}
	for _, apply := range o {
		if err := apply(&opts); err != nil {
			return nil, err
		}
	}

	var err error
	if opts.h == nil {
		opts.h, err = libp2p.New()
		if err != nil {
			return nil, err
		}
	}
	if len(opts.bootstrapPeers) == 0 {
		opts.bootstrapPeers = make([]peer.AddrInfo, 0, len(dht.DefaultBootstrapPeers))
		for _, p := range dht.DefaultBootstrapPeers {
			pa, err := peer.AddrInfoFromP2pAddr(p)
			if err != nil {
				return nil, err
			}
			opts.bootstrapPeers = append(opts.bootstrapPeers, *pa)
		}
	}
	return &opts, nil
}

func WithHost(h host.Host) Option {
	return func(o *options) error {
		o.h = h
		return nil
	}
}

func WithHttpListenAddr(a string) Option {
	return func(o *options) error {
		o.httpListenAddr = a
		return nil
	}
}

func WithBootstrapPeers(p ...peer.AddrInfo) Option {
	return func(o *options) error {
		o.bootstrapPeers = p
		return nil
	}
}

func WithUseAcceleratedDHT(b bool) Option {
	return func(o *options) error {
		o.useAccDHT = b
		return nil
	}
}
