package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/ipfs/go-log/v2"
	caskadht "github.com/ipni/caskadht"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

var logger = log.Logger("caskadht/cmd")

const (
	libp2pUserAgent        = "ipni/caskadht"
	libp2pMinConnsOutbound = 0xffff
	libp2pMinBaseLimitFD   = 0xfff
)

func main() {
	libp2pIdentityPath := flag.String("libp2pIdentityPath", "", "The path to the marshalled libp2p host identity. If unspecified a random identity is generated.")
	libp2pListenAddrs := flag.String("libp2pListenAddrs", "", "The comma separated libp2p host listen addrs. If unspecified the default listen addrs are used at ephemeral port.")
	httpListenAddr := flag.String("httpListenAddr", "0.0.0.0:40080", "The caskadht HTTP server listen address.")
	useAcceleratedDHT := flag.Bool("useAcceleratedDHT", true, "Weather to use accelerated DHT client when possible.")
	ipniRequireQueryParam := flag.Bool("ipniRequireQueryParam", false, "Weather to require IPNI `cascade` query parameter with matching label in order to respond to HTTP lookup requests. Not required by default.")
	ipniCascadeLabel := flag.String("ipniCascadeLabel", "ipfs-dht", "The IPNI cascade label associated to this instance.")
	logLevel := flag.String("logLevel", "info", "The logging level. Only applied if GOLOG_LOG_LEVEL environment variable is unset.")
	flag.Parse()

	if _, set := os.LookupEnv("GOLOG_LOG_LEVEL"); !set {
		_ = log.SetLogLevel("*", *logLevel)
	}

	hOpts := []libp2p.Option{
		libp2p.UserAgent(libp2pUserAgent),
	}
	if *libp2pIdentityPath != "" {
		p := filepath.Clean(*libp2pIdentityPath)
		logger := logger.With("path", p)
		logger.Info("Unmarshalling libp2p host identity")
		mid, err := os.ReadFile(p)
		if err != nil {
			logger.Fatalw("Failed to read libp2p host identity file", "err", err)
		}
		id, err := crypto.UnmarshalPrivateKey(mid)
		if err != nil {
			logger.Fatalw("Failed to unmarshal libp2p host identity file", "err", err)
		}
		hOpts = append(hOpts, libp2p.Identity(id))
	}
	if *libp2pListenAddrs != "" {
		hOpts = append(hOpts, libp2p.ListenAddrStrings(strings.Split(*libp2pListenAddrs, ",")...))
	}
	if *useAcceleratedDHT {
		// Adjust outbound connections and base limit FD to allow the accelerated DHT client to
		// (re)load its routing table. Because, currently the client does not gracefully handle
		// Resource Manager throttling.
		//
		// Note, a high outbound connection limit is considered less of a DoS risk than a high
		// inbound connection limit. Further, due to the accelerated DHT client's behaviour, we only
		// need higher connection limit; not more streams.
		//
		// Borrowed from:
		//  - https://github.com/ipfs-shipyard/someguy/blob/84d56168e2dd6aae00e378a3ce8445de2c2f880f/server.go#L72
		// TODO: Revisit limit increases once the the accelerated DHT client tolerates throttling.
		defaultLimits := rcmgr.DefaultLimits
		libp2p.SetDefaultServiceLimits(&defaultLimits)
		if defaultLimits.SystemBaseLimit.ConnsOutbound < libp2pMinConnsOutbound {
			defaultLimits.SystemBaseLimit.ConnsOutbound = libp2pMinConnsOutbound
			if defaultLimits.SystemBaseLimit.Conns < defaultLimits.SystemBaseLimit.ConnsOutbound {
				defaultLimits.SystemBaseLimit.Conns = defaultLimits.SystemBaseLimit.ConnsOutbound
			}
		}
		if defaultLimits.SystemBaseLimit.FD < libp2pMinBaseLimitFD {
			defaultLimits.SystemBaseLimit.FD = libp2pMinBaseLimitFD
		}

		rm, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(defaultLimits.AutoScale()))
		if err != nil {
			logger.Fatalw("Failed to adjust resource manager limits for the accelerated DHT client", "err", err)
		}
		hOpts = append(hOpts, libp2p.ResourceManager(rm))
	}
	h, err := libp2p.New(hOpts...)
	if err != nil {
		logger.Fatalw("Failed to instantiate libp2p host", "err", err)
	}

	c, err := caskadht.New(
		caskadht.WithHost(h),
		caskadht.WithHttpListenAddr(*httpListenAddr),
		caskadht.WithUseAcceleratedDHT(*useAcceleratedDHT),
		caskadht.WithIpniCascadeLabel(*ipniCascadeLabel),
		caskadht.WithIpniRequireCascadeQueryParam(*ipniRequireQueryParam),
	)
	if err != nil {
		logger.Fatalw("Failed to instantiate caskadht", "err", err)
	}
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		logger.Fatalw("Failed to start caskadht", "err", err)
	}
	sch := make(chan os.Signal, 1)
	signal.Notify(sch, os.Interrupt)

	<-sch
	logger.Info("Terminating...")
	if err := c.Shutdown(ctx); err != nil {
		logger.Warnw("Failure occurred while shutting down server.", "err", err)
	} else {
		logger.Info("Shut down server successfully.")
	}
}
