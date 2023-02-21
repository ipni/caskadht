package cascadht

import (
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type (
	Option  func(*options) error
	options struct {
		h                            host.Host
		httpListenAddr               string
		httpAllowOrigin              string
		httpResponsePreferJson       bool
		bootstrapPeers               []peer.AddrInfo
		useAccDHT                    bool
		ipniCascadeLabel             string
		ipniRequireCascadeQueryParam bool
		addrFilterDisabled           bool
		findProvidersLimit           int
		prAttemptCacheMaxSize        int
		prAttemptCacheMaxAge         time.Duration
	}
)

func newOptions(o ...Option) (*options, error) {
	opts := options{
		httpListenAddr:        "0.0.0.0:40080",
		useAccDHT:             false,
		ipniCascadeLabel:      "ipfs-dht",
		httpAllowOrigin:       "*",
		prAttemptCacheMaxSize: 1024,
		prAttemptCacheMaxAge:  20 * time.Minute,
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

func WithIpniCascadeLabel(l string) Option {
	return func(o *options) error {
		o.ipniCascadeLabel = l
		return nil
	}
}

func WithHttpAllowOrigin(ao string) Option {
	return func(o *options) error {
		o.httpAllowOrigin = ao
		return nil
	}
}

// WithIpniRequireCascadeQueryParam sets whether the server should require IPNI cascade query
// parameter with the matching label in order to respond to HTTP lookup requests.
// See: WithIpniCascadeLabel
func WithIpniRequireCascadeQueryParam(p bool) Option {
	return func(o *options) error {
		o.ipniRequireCascadeQueryParam = p
		return nil
	}
}

// WithHttpResponsePreferJson sets whether to prefer non-streaming json response over streaming
// ndjosn when the Accept header uses `*/*` wildcard. By default, in such case ndjson streaming
// response is preferred.
func WithHttpResponsePreferJson(b bool) Option {
	return func(o *options) error {
		o.httpResponsePreferJson = b
		return nil
	}
}

// WithAddrFilterDisabled sets whether to filter out addresses that are not publicly dialable.
// By default such address are excluded from results.
// See: IsPubliclyDialableAddr.
func WithAddrFilterDisabled(b bool) Option {
	return func(o *options) error {
		o.addrFilterDisabled = b
		return nil
	}
}

// WithFindProvidersLimit sets the limit on number of providers to find.
// Defaults to zero, i.e. no limit.
func WithFindProvidersLimit(l int) Option {
	return func(o *options) error {
		o.findProvidersLimit = l
		return nil
	}
}
